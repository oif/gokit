package hook_test

import (
	"testing"
)

func TestSourceHook(t *testing.T) {
	logger := getSourceLogger()
	logger.Info("testing source hook")
}
