package errors_test

import (
	"fmt"
	"testing"

	"github.com/oif/gokit/errors"
)

const testStatus = "test"

func TestErrors_WithContext(t *testing.T) {
	e2 := errors.New(2, testStatus, errors.WithContext(testStatus, 666))
	if val, ok := e2.Get(testStatus); !ok || val != 666 {
		t.Fatal("New with context failed")
	}
}

func TestErrors_CodeUnique(t *testing.T) {
	defer func() {
		e := recover()
		if e == nil {
			t.Fatal("Code unique test failed")
		}
	}()
	errors.RequireCodeUnique = errors.Enable

	errors.New(1, testStatus)
	errors.New(1, testStatus)
}

func TestErrors_DeepCopy(t *testing.T) {
	errors.RequireCodeUnique = errors.Disable
	e1 := errors.New(1, testStatus)
	e2 := e1.DeepCopy()
	e2.Set(testStatus, 666)

	if val, ok := e1.Get(testStatus); ok && val == testStatus {
		t.Fatal("Deep copy failed")
	}
}

func TestErrors_Render(t *testing.T) {
	e := errors.New(3, testStatus)
	if echo := e.Error(); echo != "[3_test] map[]" {
		t.Fatalf("Error func broken, expect `[3_test] map[]` got `%s`", echo)
	}
	e.SetRender(func(e errors.E) string {
		return fmt.Sprintf("%d%s", e.Code(), e.Status())
	})
	if echo := e.Error(); echo != "3test" {
		t.Fatalf("Broken render, expect `3test` got `%s`", echo)
	}
}

func TestErrors_Usage(t *testing.T) {
	errors.RequireCodeUnique = errors.Disable
	e1 := errors.New(1, testStatus)
	e2 := errors.New(2, testStatus)
	e3 := errors.New(3, testStatus)

	err := e1.DeepCopy()
	// Set context
	err.Set(testStatus, "hey")
	// when code unique is disable
	err.
		Is(func() {
			t.Fatal("is e2 or e3")
		}, e2, e3).
		Is(func() {
			t.Fatal("is build-int error")
		}, fmt.Errorf("haha")).
		Is(func() {
			t.Log("yep, i'm")
		}, e1).
		Is(func() {
			t.Fatal("mismatch all error")
		}, nil)

	type (
		NotFoundError errors.E
		InternalError errors.E
	)

	e4 := errors.New(4, testStatus).(NotFoundError)
	e5 := errors.New(5, testStatus).(InternalError)

	err = e4.DeepCopy()
	switch err.(type) {
	case NotFoundError:
		// ok
	default:
		t.Fatal("type test error")
	}

	err = e5.DeepCopy()
	switch err.(type) {
	case NotFoundError:
		// ok
	default:
		t.Fatal("type test error")
	}

	errors.RequireCodeUnique = errors.Enable
	e7 := errors.New(404, testStatus)
	err = e7.DeepCopy()

	switch err.Code() {
	case e1.Code():
	case e7.Code():
	default:
		t.Fatal("type test error")
	}
}
