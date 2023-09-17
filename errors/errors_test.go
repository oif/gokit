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

func TestDeclareI18n(t *testing.T) {
	predefinedErrorType := errors.Declare("NOT_FOUND", "404", errors.I18N{
		errors.EN: "not found", // const
		"zh":      "未找到",       // or literal
	})
	if predefinedErrorType.Error() != "NOT_FOUND - 404" {
		t.Fatal("Unexpected error output")
	}

	casted, ok := predefinedErrorType.(*errors.Error)
	if !ok {
		t.Fatal("Unexpected type")
	}
	if casted.I18NMessage(errors.EN) != "not found" {
		t.Fatal("Unexpected i18n EN message")
	}
	if casted.I18NMessage(errors.ZH) != "未找到" {
		t.Fatal("Unexpected i18n ZH message")
	}
}
