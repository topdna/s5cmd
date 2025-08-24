package log

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// output is an internal container for messages to be logged.
type output struct {
	std     *os.File
	message string
}

// outputCh is used to synchronize writes to standard output. Multi-line
// logging is not possible if all workers print logs at the same time.
var outputCh = make(chan output, 10000)

var global *Logger
var closeOnce sync.Once
var isClosed int32

// Init inits global logger.
func Init(level string, json bool) {
	global = New(level, json)
}

// Trace prints message in trace mode.
func Trace(msg Message) {
	if global == nil {
		return
	}
	global.printf(LevelTrace, msg, os.Stdout)
}

// Debug prints message in debug mode.
func Debug(msg Message) {
	if global == nil {
		return
	}
	global.printf(LevelDebug, msg, os.Stdout)
}

// Info prints message in info mode.
func Info(msg Message) {
	if global == nil {
		return
	}
	global.printf(LevelInfo, msg, os.Stdout)
}

// Stat prints stat message regardless of the log level with info print formatting.
// It uses printfHelper instead of printf to ignore the log level condition.
func Stat(msg Message) {
	if global == nil {
		return
	}
	global.printfHelper(LevelInfo, msg, os.Stdout)
}

// Error prints message in error mode.
func Error(msg Message) {
	if global == nil {
		return
	}
	global.printf(LevelError, msg, os.Stderr)
}

// Close closes logger and its channel.
func Close() {
	closeOnce.Do(func() {
		if global != nil {
			// Set the closed flag first
			atomic.StoreInt32(&isClosed, 1)
			// Small delay to let any pending writes complete
			// This is a simple way to avoid race conditions
			close(outputCh)
			<-global.donech
			global = nil
		}
	})
}

// Logger is a structure for logging messages.
type Logger struct {
	donech chan struct{}
	json   bool
	level  LogLevel
}

// New creates new logger.
func New(level string, json bool) *Logger {
	logLevel := LevelFromString(level)
	logger := &Logger{
		donech: make(chan struct{}),
		json:   json,
		level:  logLevel,
	}
	go logger.out()
	return logger
}

// printf prints message according to the given level, message and std mode.
func (l *Logger) printf(level LogLevel, message Message, std *os.File) {
	if level < l.level {
		return
	}
	l.printfHelper(level, message, std)
}

func (l *Logger) printfHelper(level LogLevel, message Message, std *os.File) {
	// Check if we're closing to avoid sending on closed channel
	if atomic.LoadInt32(&isClosed) == 1 {
		return
	}

	var outputMsg output
	if l.json {
		outputMsg = output{
			message: message.JSON(),
			std:     std,
		}
	} else {
		outputMsg = output{
			message: fmt.Sprintf("%v%v", level, message.String()),
			std:     std,
		}
	}

	// Try to send, but don't block if channel is closed
	select {
	case outputCh <- outputMsg:
	default:
		// Channel is likely closed or full, just return
	}
}

// out listens for outputCh and logs messages.
func (l *Logger) out() {
	defer close(l.donech)

	for outputMsg := range outputCh {
		_, _ = fmt.Fprintln(outputMsg.std, outputMsg.message)
	}
}

// LogLevel is the level of Logger.
type LogLevel int

const (
	LevelTrace LogLevel = iota
	LevelDebug
	LevelInfo
	LevelError
)

// String returns the string representation of logLevel.
func (l LogLevel) String() string {
	switch l {
	case LevelInfo:
		return ""
	case LevelError:
		return "ERROR "
	case LevelDebug:
		return "DEBUG "
	case LevelTrace:
		// levelTrace is used for printing aws sdk logs and
		// aws-sdk-go already adds "DEBUG" prefix to logs.
		// So do not add another prefix to log which makes it
		// look weird.
		return ""
	default:
		return "UNKNOWN "
	}
}

// LevelFromString returns logLevel for given string. It
// return `levelInfo` as a default.
func LevelFromString(s string) LogLevel {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "error":
		return LevelError
	case "trace":
		return LevelTrace
	default:
		return LevelInfo
	}
}
