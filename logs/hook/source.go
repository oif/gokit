package hook

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Trace:
// log caller -> logrus.[Entry,Logger][Info,others](maybe twice or more)
// -> logrus.Entry.log -> logrus.Entry.fireHooks -> logrus.LevelHooks.Fire -> logrus.Fire
// So, we just skip all the functions that in logrus and catch the first one after logrus

const (
	// Tricky value according to trace
	callerTrickySkipping = 5
	callerTraceDepth     = 3
	callerFieldName      = "caller"
	pathSeparator        = string(os.PathSeparator)
	fileNamePlaceholder  = "<file>"
)

// Source implements logrus.Hook
type Source struct {
	level            logrus.Level
	srcPath          string
	useShortenCaller bool
}

// NewSource currently don't support specify to enable shorten caller or not, set true as fixed
func NewSource(level logrus.Level) *Source {
	return &Source{
		level:            level,
		srcPath:          filepath.Join(build.Default.GOPATH, "src"),
		useShortenCaller: true,
	}
}

// Fire implement logrus.Hook.Fire which is to print log
func (s *Source) Fire(entry *logrus.Entry) error {
	trace := make([]uintptr, callerTraceDepth)
	actualDepth := runtime.Callers(callerTrickySkipping, trace)
	if actualDepth == 0 {
		return nil
	}
	frames := runtime.CallersFrames(trace[:actualDepth])
	var (
		currentFrame runtime.Frame
		next         bool
	)

	for {
		currentFrame, next = frames.Next()
		// Read next frame till the first one after logrus package or the end
		if !next ||
			!strings.Contains(currentFrame.File, "github.com/sirupsen/logrus") {
			break
		}
	}

	// Catch first frame after logrus, construct field
	entry.Data[callerFieldName] = s.makeSourceField(currentFrame)

	return nil
}

// Levels implements logrus.Hook.Levels which return level(s) should fire
func (s *Source) Levels() []logrus.Level {
	levels := make([]logrus.Level, 0)
	for _, level := range logrus.AllLevels {
		if level <= s.level {
			levels = append(levels, level)
		}
	}
	return levels
}

// Format ->  file:line(function)
func (s *Source) makeSourceField(frame runtime.Frame) string {
	funcSlice := strings.Split(frame.Function, ".")
	funcName := funcSlice[len(funcSlice)-1:][0]
	fileName := fileNamePlaceholder
	if s.useShortenCaller {
		// Once shorten caller enabled, we will use the last segment(separated by os.PathSeparator) as file name
		paths := strings.Split(frame.File, pathSeparator)
		if len(paths) > 0 {
			fileName = paths[len(paths)-1]
		}
	} else {
		// Otherwise, trim $GOPATH/src prefix then we get the full path of caller in repo
		fileName = strings.TrimPrefix(frame.File, s.srcPath+pathSeparator)
	}
	return fmt.Sprintf("%s:%d(%s)", fileName, frame.Line, funcName)
}
