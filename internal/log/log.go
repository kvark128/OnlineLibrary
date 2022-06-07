package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func StringToLevel(str string) (Level, error) {
	switch str {
	case ErrorLevel.String():
		return ErrorLevel, nil
	case InfoLevel.String():
		return InfoLevel, nil
	case WarningLevel.String():
		return WarningLevel, nil
	case DebugLevel.String():
		return DebugLevel, nil
	default:
		return 0, fmt.Errorf("%s is not a supported log level", str)
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
	msg := fmt.Sprintf(strings.ReplaceAll(format, "\n", "\n\t"), args...)
	fmt.Fprintf(out, "%s - %s:%d (%s):\r\n\t%s\r\n", level, filepath.Base(file), line, clock, msg)
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

func SetLevel(level Level) error {
	mu.Lock()
	defer mu.Unlock()
	if level < ErrorLevel || level > DebugLevel {
		return fmt.Errorf("unsupported log level")
	}
	logLevel = level
	return nil
}

func GetLevel() Level {
	mu.Lock()
	defer mu.Unlock()
	return logLevel
}
