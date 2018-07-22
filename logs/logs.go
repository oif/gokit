package logs

import (
	"bytes"
	"fmt"
	"os"

	"github.com/oif/gokit/logs/hook"

	"github.com/sirupsen/logrus"
)

const (
	DefaultLogLevel = "debug"
)

var (
	logger *logrus.Logger

	outputSplitKeyword = []byte("level=error")
)

// Split logs to stdout and stderr
type outputSplitter struct{}

func (s *outputSplitter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, outputSplitKeyword) {
		return os.Stderr.Write(p)
	}
	return os.Stdout.Write(p)
}

type Option struct {
	Level              string           // Log level: debug, info, warning(warn), error, fatal, panic
	Formatter          logrus.Formatter // Set log formatter
	SplitErrorToStderr bool             // Split error level log to stderr
	EnableSourceHook   bool             // Enable Source Hook(by @git-hulk)
}

func NewDefaultOption() *Option {
	o := new(Option)
	return o
}

type OptionFunc func(*Option)

func LogLevel(level string) OptionFunc {
	return func(o *Option) {
		o.Level = level
	}
}

func SetFormatter(formatter logrus.Formatter) OptionFunc {
	return func(o *Option) {
		o.Formatter = formatter
	}
}

func SplitErrorToStderr() OptionFunc {
	return func(o *Option) {
		o.SplitErrorToStderr = true
	}
}

func EnableSourceHook() OptionFunc {
	return func(o *Option) {
		o.EnableSourceHook = true
	}
}

// Setup logger with options
func Setup(opts ...OptionFunc) (*logrus.Logger, error) {
	opt := NewDefaultOption()
	for _, set := range opts {
		set(opt)
	}

	logger = logrus.New()

	// Set log level
	if opt.Level == "" {
		opt.Level = DefaultLogLevel
	}
	level, err := logrus.ParseLevel(opt.Level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(level)
	// Set log level finished

	// Split log to stderr and stdout
	if opt.SplitErrorToStderr {
		logger.Out = &outputSplitter{}
	}
	// Done

	// Set formatter
	if opt.Formatter != nil {
		logger.Formatter = opt.Formatter
	}
	// Done

	// Set hooks
	if opt.EnableSourceHook {
		logger.Hooks.Add(hook.NewSource(level))
	}
	// Done

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

// GetLogger return initialized logger, panic when logger is uninitialized
func GetLogger() *logrus.Logger {
	if logger == nil {
		panic("logger uninitialized")
	}
	return logger
}
