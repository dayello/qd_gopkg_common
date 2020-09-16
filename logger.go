package common

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

func NewLogger(dir string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(NewFormatter())
	logger.SetReportCaller(true)
	logger.AddHook(&LogHookGlobal{})
	logger.SetOutput(GetMultiLogWriter(dir))
	return logger
}

func GetMultiLogWriter(dir string) io.Writer {
	writer, err := rotatelogs.New(dir+"/app-%Y%m%d.log", rotatelogs.WithRotationCount(30))
	if err != nil {
		panic(err)
	}
	writers := []io.Writer{
		writer,
		os.Stdout,
	}
	return io.MultiWriter(writers...)
}

func GetJSONFormatter() *logrus.JSONFormatter {
	return &logrus.JSONFormatter{
		DisableTimestamp: true,
		TimestampFormat:  "2006-01-02 15:04:05",
	}
}

type LogHookGlobal struct {
}

func (h *LogHookGlobal) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogHookGlobal) Fire(e *logrus.Entry) error {
	e.Data["@"] = time.Now().Format("2006-01-02 15:04:05") //展示时间@作为JSON第一个Key
	return nil
}
