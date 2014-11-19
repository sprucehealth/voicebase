package golog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Level represents a log level (CRIT, ERR, ...)
type Level int32

type Logger interface {
	Context(ctx ...interface{}) Logger

	SetLevel(l Level) Level
	Level() Level
	// L returns true if the current level is greater than or equal to 'l'
	L(l Level) bool

	SetHandler(h Handler)
	Handler() Handler

	LogDepthf(calldepth int, l Level, format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Criticalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Handler interface {
	Log(e *Entry) error
}

type Entry struct {
	Time time.Time
	Lvl  Level
	Msg  string
	Ctx  []interface{}
	Src  string
}

// Log levels
const (
	CRIT  Level = iota // For panics (code bugs)
	ERR                // General errors (e.g. errors from database, etc)
	WARN               // e.g. correctable but inconsistent state
	INFO               // e.g. web access logs, analytics, ...
	DEBUG              // Normally turned off but can help to track down issues
)

// Levels maps log level to a string
var Levels = map[Level]string{
	CRIT:  "CRIT",
	ERR:   "ERR",
	WARN:  "WARN",
	INFO:  "INFO",
	DEBUG: "DEBUG",
}

func (l Level) String() string {
	if s := Levels[l]; s != "" {
		return s
	}
	return strconv.Itoa(int(l))
}

type logger struct {
	mu  sync.Mutex
	ctx []interface{}
	hnd Handler
	lvl Level
}

var defaultL *logger

type writer struct{}

var Writer io.Writer = writer{}

func (w writer) Write(b []byte) (int, error) {
	m := string(b)
	if strings.HasPrefix(m, "[ERR]") || strings.HasPrefix(m, "ERR") {
		defaultL.Errorf(m)
	} else if strings.HasPrefix(m, "[WARN]") || strings.HasPrefix(m, "WARN") {
		defaultL.Warningf(m)
	} else {
		defaultL.Infof(m)
	}
	return len(m), nil
}

func init() {
	fmtLow := LogfmtFormatter()
	fmtHigh := LogfmtFormatter()

	if IsTerminal(os.Stdout.Fd()) {
		fmtLow = TerminalFormatter()
	}
	if IsTerminal(os.Stderr.Fd()) {
		fmtHigh = TerminalFormatter()
	}

	DefaultHandler = SplitHandler(WARN, WriterHandler(os.Stdout, fmtLow), WriterHandler(os.Stderr, fmtHigh))
	defaultL = &logger{
		ctx: nil,
		hnd: DefaultHandler,
		lvl: INFO,
	}
}

var DefaultHandler Handler

func Default() Logger {
	return defaultL
}

func (l *logger) SetLevel(lvl Level) Level {
	return Level(atomic.SwapInt32((*int32)(&l.lvl), int32(lvl)))
}

func (l *logger) Level() Level {
	return Level(atomic.LoadInt32((*int32)(&l.lvl)))
}

func (l *logger) SetHandler(h Handler) {
	l.mu.Lock()
	l.hnd = h
	l.mu.Unlock()
}

func (l *logger) Handler() Handler {
	l.mu.Lock()
	h := l.hnd
	l.mu.Unlock()
	return h
}

func (l *logger) L(lvl Level) bool {
	return l.Level() >= lvl
}

func (l *logger) Context(ctx ...interface{}) Logger {
	if len(l.ctx) != 0 {
		ctx = append(l.ctx, ctx...)
	}
	return &logger{
		ctx: ctx,
		hnd: l.Handler(),
		lvl: l.Level(),
	}
}

func (l *logger) LogDepthf(calldepth int, lvl Level, format string, args ...interface{}) {
	if calldepth >= 0 {
		calldepth++
	}
	if l.L(lvl) {
		entry := &Entry{
			Time: time.Now(),
			Lvl:  lvl,
			Msg:  fmt.Sprintf(format, args...),
			Ctx:  l.ctx,
		}
		if calldepth > 0 {
			_, file, line, ok := runtime.Caller(calldepth)
			if ok {
				short := file
				depth := 0
				for i := len(file) - 1; i > 0; i-- {
					if file[i] == '/' {
						short = file[i+1:]
						depth++
						if depth == 2 {
							break
						}
					}
				}
				file = short
				entry.Src = fmt.Sprintf("%s:%d", file, line)
			}
		}
		l.Handler().Log(entry)
	}
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	l.LogDepthf(1, CRIT, format, args...)
	os.Exit(255)
}

func (l *logger) Criticalf(format string, args ...interface{}) {
	l.LogDepthf(1, CRIT, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.LogDepthf(1, ERR, format, args...)
}

func (l *logger) Warningf(format string, args ...interface{}) {
	l.LogDepthf(1, WARN, format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.LogDepthf(-1, INFO, format, args...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.LogDepthf(-1, DEBUG, format, args...)
}

func Context(ctx ...interface{}) Logger {
	return defaultL.Context(ctx...)
}

// LogDepthf logs a message and includes the function and line number in the
// call stack at the position of calldepth. A calldepth of 0 is the caller
// of LogDepthf, a depth of 1 is its caller, and so forth. A calldepth less than
// 0 disables logging of the source file and line.
func LogDepthf(calldepth int, lvl Level, format string, args ...interface{}) {
	if calldepth >= 0 {
		calldepth++
	}
	defaultL.LogDepthf(calldepth, lvl, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, CRIT, format, args...)
	os.Exit(255)
}

func Criticalf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, CRIT, format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, ERR, format, args...)
}

func Warningf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, WARN, format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultL.LogDepthf(-1, INFO, format, args...)
}

func Debugf(format string, args ...interface{}) {
	defaultL.LogDepthf(-1, DEBUG, format, args...)
}
