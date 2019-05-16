package sqlbuilder

import (
	"errors"
)

var (
	ErrConstructFirst  = errors.New("construct the sql first then generate it")
	ErrMissingTable    = errors.New("missing table")
	ErrMissingFields   = errors.New("missing fields")
	ErrInvalidObject   = errors.New("invalid object maybe a nil or not a pointer")
	ErrUnsupportedKind = errors.New("target sql kind unsupported")
)
