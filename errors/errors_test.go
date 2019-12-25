package errors_test

import (
	"testing"

	"github.com/oif/gokit/errors"
)

func TestDeclare(t *testing.T) {
	err := errors.Declare("NOT_FOUND", "some error")
	if err.Error() != "NOT_FOUND - some error" {
		t.Fatalf("Unexpected error output: %s", err.Error())
	}
}
