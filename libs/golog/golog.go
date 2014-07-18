package golog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
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

	Logf(calldepth int, l Level, format string, args ...interface{})
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
	defaultL.Infof(m)
	return len(m), nil
}

func init() {
	defaultL = &logger{
		ctx: nil,
		hnd: DefaultHandler,
		lvl: INFO,
	}
}

var DefaultHandler = IOHandler(os.Stdout, os.Stderr, LogfmtFormatter())

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

func (l *logger) Logf(calldepth int, lvl Level, format string, args ...interface{}) {
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
	l.Logf(2, CRIT, format, args...)
	os.Exit(255)
}

func (l *logger) Criticalf(format string, args ...interface{}) {
	l.Logf(2, CRIT, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.Logf(2, ERR, format, args...)
}

func (l *logger) Warningf(format string, args ...interface{}) {
	l.Logf(2, WARN, format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.Logf(-1, INFO, format, args...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.Logf(-1, DEBUG, format, args...)
}

func Logf(calldepth int, lvl Level, format string, args ...interface{}) {
	defaultL.Logf(calldepth, lvl, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultL.Fatalf(format, args...)
}

func Criticalf(format string, args ...interface{}) {
	defaultL.Criticalf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultL.Errorf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	defaultL.Warningf(format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultL.Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	defaultL.Debugf(format, args...)
}
