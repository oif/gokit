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

func TestIs(t *testing.T) {
	x := errors.Declare("NOT_FOUND", "some")
	y := errors.Declare("NOT_FOUND", "someone")
	z := errors.Declare("NOT_FOUND", "some")
	if errors.Is(x, y) {
		t.Fatalf("Unexpected equal")
	}
	if !errors.Is(x, z) {
		t.Fatalf("Unexpected not equal")
	}
}
