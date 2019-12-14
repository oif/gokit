package logs_test

import (
	"testing"

	"github.com/oif/gokit/logs"

	"github.com/sirupsen/logrus"
)

func TestSetup(t *testing.T) {
	logger, err := logs.Setup(
		logs.WithLogLevel(logrus.DebugLevel),
		logs.WithSourceHook(),
		logs.WithSTDSplit(),
	)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("Howdy!")
}
