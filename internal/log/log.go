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

const (
	InfoLevel Level = iota
	DebugLevel
)

var (
	mu                 = new(sync.Mutex)
	logLevel           = InfoLevel
	out      io.Writer = os.Stdout
)

func log(calldepth int, level, msg string, args []interface{}) {
	if out == nil {
		return
	}
	hour, min, sec := time.Now().Clock()
	clock := fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	msg = fmt.Sprintf(msg, args...)
	s := fmt.Sprintf("%s - %s:%d (%s):\n%s\n", level, filepath.Base(file), line, clock, msg)
	out.Write([]byte(s))
}

func Info(msg string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	log(2, "INFO", msg, args)
}

func Debug(msg string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logLevel < DebugLevel {
		return
	}
	log(2, "DEBUG", msg, args)
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
