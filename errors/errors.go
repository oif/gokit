package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	class   string
	message string
}

func Declare(class, message string) error {
	return &Error{
		class:   class,
		message: message,
	}
}

func New(message string) error {
	return errors.New(message)
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s", e.class, e.message)
}

func (e *Error) Class() string {
	return e.class
}

func (e *Error) Message() string {
	return e.message
}

func IsClass(err error, class string) bool {
	switch err.(type) {
	case *Error:
		e := err.(*Error)
		return e.class == class
	default:
		return false
	}
}

func Parse(err error) (*Error, bool) {
	e, ok := err.(*Error)
	return e, ok
}

func Is(one error, theOther error) bool {
	if one == theOther {
		return true
	}
	parsedOne, ok := Parse(one)
	if !ok {
		return false
	}
	parsedTheOther, ok := Parse(theOther)
	if !ok {
		return false
	}
	return parsedOne.Class() == parsedTheOther.Class() && parsedOne.Message() == parsedTheOther.Message()
}
