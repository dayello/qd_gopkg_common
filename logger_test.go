package common

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLogger(t *testing.T) {
	logrus.SetOutput(GetMultiLogWriter("./"))
	logrus.SetFormatter(NewFormatter())
	logrus.SetReportCaller(true)
	logrus.AddHook(&LogHookGlobal{})
	logrus.WithField("phone", 13026996654).WithField("idCard","350822199102221115").Info("hello", "phone:177506739205,id:350822199102115558")
}