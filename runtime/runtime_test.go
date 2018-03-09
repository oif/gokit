package runtime_test

import (
	"testing"

	"github.com/oif/gokit/runtime"
)

func TestHandleCrash(t *testing.T) {
	var e []struct{}
	defer runtime.HandleCrash(func(e interface{}) {
		recover()
	})
	t.Log(e[1])
	t.Fatal("Ever trigger panic")
}
