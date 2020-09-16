package dingtalk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.qufenqi.com/universe/crm/gopkg/common/dingtalk/robot"
	"git.qufenqi.com/universe/crm/gopkg/common/dingtalk/utils"
	"github.com/go-resty/resty/v2"
)

type Option func(*DingTalk)

func WithSecret(secret string) Option {
	return func(dt *DingTalk) {
		dt.secret = secret
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(dt *DingTalk) {
		dt.timeout = timeout
	}
}

type Requester interface {
	GetMethod() string
	GetHeader() map[string]string
	GetBody() ([]byte, error)
	GetSuccessCode() int64
}

type ResponseMsg struct {
	ErrCode         int64  `json:"errcode"`
	ErrMsg          string `json:"errmsg"`
	ApplicationHost string `json:"application_host,omitempty"`
	ServiceHost     string `json:"service_host,omitempty"`
}

func (r ResponseMsg) String() string {
	data, err := json.Marshal(&r)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

type DingTalk struct {
	mu       sync.Mutex
	url      string
	secret   string
	timeout  time.Duration
	client   *resty.Client
	response *resty.Response
}

func New(url string, options ...Option) *DingTalk {
	dt := &DingTalk{
		url:     url,
		timeout: 5 * time.Second,
	}
	for _, option := range options {
		option(dt)
	}
	dt.initClient()
	return dt
}

func (dt *DingTalk) GetSecret() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.secret
}

func (dt *DingTalk) SetSecret(secret string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.secret = secret
}

func (dt *DingTalk) SetTimeout(timeout time.Duration) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.client.SetTimeout(timeout)
}

func (dt *DingTalk) initClient() {
	// 拼接请求参数
	step := "?"
	if strings.Contains(dt.url, "?") {
		step = "&"
	}
	params := dt.genQueryParams()
	dt.url = strings.Join([]string{dt.url, params}, step)
	dt.client = resty.New()
}

func (dt *DingTalk) genQueryParams() string {
	params := url.Values{}
	if dt.secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := utils.ComputeSignature(timestamp, dt.secret)
		params.Add("timestamp", strconv.FormatInt(timestamp, 10))
		params.Add("sign", sign)
	}
	return params.Encode()
}

func (dt *DingTalk) Request(req Requester) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if err := dt.checkURL(); err != nil {
		return err
	}
	_, err := dt.request(req)
	if err != nil {
		return err
	}
	if err := dt.checkResponse(req); err != nil {
		return err
	}
	return nil
}

func (dt *DingTalk) checkURL() error {
	_, err := url.Parse(dt.url)
	if err != nil {
		return err
	}
	return nil
}

func (dt *DingTalk) request(req Requester) (string, error) {
	method := req.GetMethod()
	header := req.GetHeader()
	body, err := req.GetBody()
	if err != nil {
		return "", err
	}

	request := dt.client.R().SetHeaders(header)

	switch method {
	case http.MethodGet:
		dt.response, err = request.Get(dt.url)
	case http.MethodPost:
		dt.response, err = request.SetBody(body).Post(dt.url)
	case http.MethodPut:
		dt.response, err = request.SetBody(body).Put(dt.url)
	}

	return string(body), nil
}

func (dt *DingTalk) checkResponse(req Requester) error {
	data := dt.response.Body()
	if dt.response.StatusCode() != http.StatusOK {
		return fmt.Errorf("invalid http status %d, body: %s", dt.response.StatusCode(), data)
	}

	respMsg := ResponseMsg{}
	if err := json.Unmarshal(data, &respMsg); err != nil {
		return fmt.Errorf("body: %s, %w", data, err)
	}
	respMsg.ApplicationHost = dt.response.Header().Get("Application-Host")
	respMsg.ServiceHost = dt.response.Header().Get("Location-Host")
	if respMsg.ErrCode != req.GetSuccessCode() {
		return fmt.Errorf("%s", respMsg)
	}
	return nil
}

func (dt *DingTalk) GetResponse() (*resty.Response, error) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	return dt.response, nil
}

// RobotSendText text类型的消息
func (dt *DingTalk) RobotSendText(text string, options ...robot.SendOption) error {
	msg := robot.Text{Content: text}
	return dt.Request(robot.NewSend(msg, options...))
}

// RobotSendLink link类型的消息
func (dt *DingTalk) RobotSendLink(title, text, messageURL, picURL string, options ...robot.SendOption) error {
	msg := robot.Link{
		Title:      title,
		Text:       text,
		MessageURL: messageURL,
		PicURL:     picURL,
	}
	return dt.Request(robot.NewSend(msg, options...))
}

// RobotSendMarkdown markdown类型的消息
func (dt *DingTalk) RobotSendMarkdown(title, text string, options ...robot.SendOption) error {
	msg := robot.Markdown{
		Title: title,
		Text:  text,
	}
	return dt.Request(robot.NewSend(msg, options...))
}

// RobotSendEntiretyActionCard 整体跳转ActionCard类型
func (dt *DingTalk) RobotSendEntiretyActionCard(title, text, singleTitle, singleURL, btnOrientation string, options ...robot.SendOption) error {
	msg := robot.ActionCard{
		Title:          title,
		Text:           text,
		SingleTitle:    singleTitle,
		SingleURL:      singleURL,
		BtnOrientation: btnOrientation,
	}
	return dt.Request(robot.NewSend(msg, options...))
}

// RobotSendIndependentActionCard 独立跳转ActionCard类型
func (dt *DingTalk) RobotSendIndependentActionCard(title, text, btnOrientation string, btns map[string]string, options ...robot.SendOption) error {
	var rBtns []robot.Btn
	for title, actionURL := range btns {
		btn := robot.Btn{
			Title:     title,
			ActionURL: actionURL,
		}
		rBtns = append(rBtns, btn)
	}
	msg := robot.ActionCard{
		Title:          title,
		Text:           text,
		Btns:           rBtns,
		BtnOrientation: btnOrientation,
	}
	return dt.Request(robot.NewSend(msg, options...))
}

// RobotSendFeedCard FeedCard类型
func (dt *DingTalk) RobotSendFeedCard(links []robot.FeedCardLink, options ...robot.SendOption) error {
	msg := robot.FeedCard{
		Links: links,
	}
	return dt.Request(robot.NewSend(msg, options...))
}

// parseLinkWithTemplate 解析template
// template:
//     第一行: title
//     第二行: messageURL
//     第三行: picURL
//     其他行: text
func parseLinkWithTemplate(text string, data interface{}) *robot.Link {
	n := 0
	b := []byte{}
	msg := &robot.Link{}
	buf := bufio.NewScanner(strings.NewReader(text))
	for buf.Scan() {
		line := buf.Text()
		switch n {
		case 0:
			msg.Title = line
		case 1:
			msg.MessageURL = line
		case 2:
			msg.PicURL = line
		default:
			b = append(b, line...)
			b = append(b, '\n')
		}
		n++
	}
	msg.Text = string(b)
	return msg
}
