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
	case DebugLevel:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

const (
	ErrorLevel Level = iota
	InfoLevel
	DebugLevel
)

var (
	mu                 = new(sync.Mutex)
	logLevel           = InfoLevel
	out      io.Writer = os.Stdout
)

func log(calldepth int, level Level, format string, args ...interface{}) {
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
	s := fmt.Sprintf("%s - %s:%d (%s):\r\n%s\r\n", level, filepath.Base(file), line, clock, msg)
	out.Write([]byte(s))
}

func Info(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	log(2, InfoLevel, format, args...)
}

func ERROR(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	log(2, ErrorLevel, format, args...)
}

func Debug(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
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
	logLevel = level
}
