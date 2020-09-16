package common

import (
	"github.com/sirupsen/logrus"
	"time"
	"regexp"
)

var (
	regIdNumber = regexp.MustCompile(`([1-9])([0-7]\d{4})(19[0-9][0-9]|20[0-3][0-9])\d{4}(\d{3})(\d|X|x)`)
	regPhone = regexp.MustCompile("(1\\d{2})(\\d{4})(\\d{4})")
)

const (
	replaceIdNumberStr = "$1$2********$4$5"
	replacePhoneStr = "$1****$3"
)
// Format configuration of the logrus formatter output.
type Format func(*Formatter) error

type Formatter struct {
	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat func(logrus.Fields, time.Time) error

	// SeverityMap allows for customizing the names for keys of the log level field.
	SeverityMap map[string]string

	// PrettyPrint will indent all json logs
	PrettyPrint bool
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	jsonFormatter := &logrus.JSONFormatter{
		DisableTimestamp: true,
		TimestampFormat:  "2006-01-02 15:04:05",
	}
	serialized, err := jsonFormatter.Format(entry)
	if err != nil {
		return nil, err
	}
	serialized = regIdNumber.ReplaceAll(serialized, []byte(replaceIdNumberStr))
	serialized = regPhone.ReplaceAll(serialized, []byte(replacePhoneStr))
	return serialized, nil
}
func NewFormatter(opts ...Format) *Formatter {
	f := Formatter{}
	if len(opts) == 0 {
		opts = append(opts, DefaultFormat)
	}
	for _, apply := range opts {
		if err := apply(&f); err != nil {
			panic(err)
		}
	}
	return &f
}

func DefaultFormat(f *Formatter) error {
	f.TimestampFormat = func(fields logrus.Fields, now time.Time) error {
		ts :=  now.Format("2006-01-02 15:04:05")
		fields["timestamp"] = ts
		return nil
	}
	return nil
}