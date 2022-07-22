package log

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	Error Level = iota
	Info
	Warning
	Debug
)

func (l Level) String() string {
	switch l {
	case Error:
		return "ERROR"
	case Info:
		return "INFO"
	case Warning:
		return "WARNING"
	case Debug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func StringToLevel(str string) (Level, error) {
	switch str {
	case Error.String():
		return Error, nil
	case Info.String():
		return Info, nil
	case Warning.String():
		return Warning, nil
	case Debug.String():
		return Debug, nil
	default:
		return 0, fmt.Errorf("%s is not a supported log level", str)
	}
}

type Logger struct {
	mu     sync.Mutex
	level  Level
	indent string
	out    io.Writer
}

func New(out io.Writer, level Level, indent string) *Logger {
	return &Logger{out: out, level: level, indent: indent}
}

func (l *Logger) log(calldepth int, level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < level || l.out == nil {
		return
	}
	hour, min, sec := time.Now().Clock()
	clock := fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	msg := fmt.Sprintf(strings.ReplaceAll(format, "\n", "\n"+l.indent), args...)
	fmt.Fprintf(l.out, "%s - %s:%d (%s):\r\n%s%s\r\n", level, filepath.Base(file), line, clock, l.indent, msg)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(2, Error, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(2, Info, format, args...)
}

func (l *Logger) Warning(format string, args ...interface{}) {
	l.log(2, Warning, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(2, Debug, format, args...)
}

func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = out
}

func (l *Logger) SetLevel(level Level) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level < Error || level > Debug {
		return fmt.Errorf("unsupported log level: %v", level)
	}
	l.level = level
	return nil
}

func (l *Logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func (l *Logger) SupportedLevels() []Level {
	return []Level{Error, Info, Warning, Debug}
}
