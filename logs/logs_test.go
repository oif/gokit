package logs_test

import (
	"testing"

	"github.com/oif/gokit/logs"
)

func TestSetup(t *testing.T) {
	logger, err := logs.Setup(
		logs.LogLevel("debug"),
		logs.EnableSourceHook(),
		logs.SplitErrorToStderr(),
	)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("Howdy!")
}
