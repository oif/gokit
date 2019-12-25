package httpresponse

import (
	"github.com/oif/gokit/errors"
)

const (
	ClassUnknown = "UNKNOWN"
)

type ResponseError struct {
	Message string                 `json:"message"`
	Class   string                 `json:"class"`
	Details map[string]interface{} `json:"details"`
}

type LocalizedMessage struct {
	// The locale used following the specification defined at
	// http://www.rfc-editor.org/rfc/bcp/bcp47.txt.
	// Examples are: "en-US", "fr-CH", "es-MX"
	Locale string `json:"locale"`
	// The localized error message in the above locale.
	Message string `json:"message"`
}

func NewResponseError() *ResponseError {
	return &ResponseError{
		Class:   ClassUnknown,
		Details: make(map[string]interface{}),
	}
}

type Response struct {
	Error   *ResponseError `json:"error,omitempty"`
	Data    interface{}    `json:"data,omitempty"`
	Message *string        `json:"message,omitempty"`
}

type WithFunc func(*Response)

func WithError(err error) WithFunc {
	return func(r *Response) {
		r.Error = NewResponseError()
		e, ok := errors.Parse(err)
		if ok {
			r.Error.Message = e.Message()
			r.Error.Class = e.Class()
		} else {
			r.Error.Message = err.Error()
		}
	}
}

func WithErrorDetail(key string, value interface{}) WithFunc {
	return func(r *Response) {
		r.Error.Details[key] = value
	}
}

func WithErrorDetails(details map[string]interface{}) WithFunc {
	return func(r *Response) {
		r.Error.Details = details
	}
}

func WithData(data interface{}) WithFunc {
	return func(r *Response) {
		r.Data = data
	}
}

func WithMessage(message string) WithFunc {
	return func(r *Response) {
		r.Message = &message
	}
}

func Construct(options ...WithFunc) *Response {
	r := new(Response)
	for _, opt := range options {
		opt(r)
	}
	return r
}
