// Package mlog is a logrus integration log library
package logs

import (
	"bytes"
	"fmt"
	"os"

	"github.com/oif/gokit/logs/hook"

	"github.com/sirupsen/logrus"
)

var (
	outputSplitKeyword = []byte("level=error")
)

// Split logs to stdout and stderr
type outputSplitter struct{}

// Write implement io.Writer which is used in logrus.Out
// this function will separator logs behind error(included) to stderr, and others to stdout
func (s *outputSplitter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, outputSplitKeyword) {
		return os.Stderr.Write(p)
	}
	return os.Stdout.Write(p)
}

// OptionFunc contains various option setter
type OptionFunc func(*logrus.Logger)

// WithLogLevel set logger printable level
func WithLogLevel(level logrus.Level) OptionFunc {
	return func(l *logrus.Logger) {
		l.SetLevel(level)
	}
}

// WithFormatter to format log output
func WithFormatter(formatter logrus.Formatter) OptionFunc {
	return func(l *logrus.Logger) {
		l.Formatter = formatter
	}
}

// WithSTDSplit if enabled will split error(or higher) logs to stderr, otherwise all to stdout
func WithSTDSplit() OptionFunc {
	return func(l *logrus.Logger) {
		l.Out = &outputSplitter{}
	}
}

// WithSourceHook to print caller in log(field)
func WithSourceHook() OptionFunc {
	return func(l *logrus.Logger) {
		l.Hooks.Add(hook.NewSource(l.GetLevel()))
	}
}

// Setup logger with options
func Setup(opts ...OptionFunc) (*logrus.Logger, error) {
	logger := logrus.New()

	// Default output as stdout
	logger.Out = os.Stdout

	for _, opt := range opts {
		opt(logger)
	}

	return logger, nil
}

// MustSetup return a *logrus.Logger after initialized
// panic if got error
func MustSetup(opts ...OptionFunc) *logrus.Logger {
	logger, err := Setup(opts...)
	if err != nil {
		panic(fmt.Sprintf("Failed to setup logger: %s", err))
	}
	return logger
}
