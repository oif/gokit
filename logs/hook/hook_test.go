package hook_test

import (
	"os"
	"testing"

	"github.com/oif/gokit/logs/hook"

	"github.com/sirupsen/logrus"
)

var sourceLogger *logrus.Logger

func getSourceLogger() *logrus.Entry {
	return sourceLogger.WithField("test", "yep")
}

func initSourceLogger() {
	sourceLogger = logrus.New()
	sourceLogger.AddHook(hook.NewSource(logrus.DebugLevel))
}

func TestMain(m *testing.M) {
	initSourceLogger()

	os.Exit(m.Run())
}
