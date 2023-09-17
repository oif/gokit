package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	class   string
	message string
	i18n    I18N
}

func Declare(class, message string, opts ...DeclareOptional) error {
	err := &Error{
		class:   class,
		message: message,
	}

	for _, opt := range opts {
		opt.Apply(err)
	}

	return err
}

type I18N map[Lang]string

func (i18n I18N) Apply(err *Error) {
	cpy := make(I18N)
	for k, v := range i18n {
		cpy[k] = v
	}
	err.i18n = cpy
}

type DeclareOptional interface {
	Apply(err *Error)
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

func (e *Error) I18NMessage(lang Lang) string {
	if e.i18n != nil {
		if msg, ok := e.i18n[lang]; ok {
			return msg
		}
	}
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
