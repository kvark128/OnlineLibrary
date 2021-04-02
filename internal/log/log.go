package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Level int

func (l Level) String() string {
	switch l {
	case ErrorLevel:
		return "ERROR"
	case InfoLevel:
		return "INFO"
	case WarningLevel:
		return "WARNING"
	case DebugLevel:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

const (
	ErrorLevel Level = iota
	InfoLevel
	WarningLevel
	DebugLevel
)

var (
	mu                 = new(sync.Mutex)
	logLevel           = InfoLevel
	out      io.Writer = os.Stdout
)

func log(calldepth int, level Level, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logLevel < level || out == nil {
		return
	}
	hour, min, sec := time.Now().Clock()
	clock := fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s - %s:%d (%s):\r\n%s\r\n", level, filepath.Base(file), line, clock, msg)
}

func Error(format string, args ...interface{}) {
	log(2, ErrorLevel, format, args...)
}

func Info(format string, args ...interface{}) {
	log(2, InfoLevel, format, args...)
}

func Warning(format string, args ...interface{}) {
	log(2, WarningLevel, format, args...)
}

func Debug(format string, args ...interface{}) {
	log(2, DebugLevel, format, args...)
}

func SetOutput(newOut io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = newOut
}

func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	switch {
	case level < ErrorLevel:
		logLevel = ErrorLevel
	case level > DebugLevel:
		logLevel = DebugLevel
	default:
		logLevel = level
	}
}
