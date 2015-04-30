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

var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{}
	},
}

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

// Entry represents a log line/entry.
type Entry struct {
	Time time.Time
	Lvl  Level
	Msg  string
	Ctx  []interface{}
	Src  string // The file:line that was the source of the log entry
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

// Stats are metrics on counts of lines per log level
type Stats struct {
	Crit  uint64
	Err   uint64
	Warn  uint64
	Info  uint64
	Debug uint64
}

func (l Level) String() string {
	if s := Levels[l]; s != "" {
		return s
	}
	return strconv.Itoa(int(l))
}

type logger struct {
	ctx   []interface{}
	lvl   Level
	stats *Stats
	mu    sync.RWMutex
	hnd   Handler
}

var defaultL *logger

type writer struct{}

// Writer can be used with log.SetOutput(Writer) to have the standard
// log packages output go through golog. By default every log entry
// is logged as INFO unless it starts with [ERR] or [WARN] in which
// case it's logged as ERR or WARN respectively.
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
	defaultL = newLogger(nil, DefaultHandler, INFO, nil)
}

func newLogger(ctx []interface{}, hnd Handler, lvl Level, stats *Stats) *logger {
	if stats == nil {
		stats = &Stats{}
	}
	l := &logger{
		ctx:   ctx,
		lvl:   lvl,
		stats: stats,
	}
	l.SetHandler(hnd)
	return l
}

var DefaultHandler Handler

func Default() Logger {
	return defaultL
}

// SetLevel sets the level for the logger
func (l *logger) SetLevel(lvl Level) Level {
	return Level(atomic.SwapInt32((*int32)(&l.lvl), int32(lvl)))
}

// Level returns the logger's current level
func (l *logger) Level() Level {
	return Level(atomic.LoadInt32((*int32)(&l.lvl)))
}

// SetHandler sets the handler for the logger
func (l *logger) SetHandler(h Handler) {
	l.mu.Lock()
	l.hnd = h
	l.mu.Unlock()
}

// Handler returns the logger's current handler
func (l *logger) Handler() Handler {
	l.mu.RLock()
	h := l.hnd
	l.mu.RUnlock()
	return h
}

// L returns whether the logger's level is greater than or equal to `lvl`
func (l *logger) L(lvl Level) bool {
	return l.Level() >= lvl
}

func (l *logger) Context(ctx ...interface{}) Logger {
	if len(l.ctx) != 0 {
		ctx = append(l.ctx, ctx...)
	}
	return newLogger(ctx, l.Handler(), l.Level(), l.stats)
}

// LogDepthf logs an entry at the requested level. If calldepth >= 0 then the empty
// call stack at that depth is included as the `Src` for the log entry. A calldepth
// of 0 is the file:line that calls this function. Arguments format and args are
// handled in the manner of fmt.Printf.
func (l *logger) LogDepthf(calldepth int, lvl Level, format string, args ...interface{}) {
	if calldepth >= 0 {
		calldepth++
	}
	if l.L(lvl) {
		entry := entryPool.Get().(*Entry)
		entry.Time = time.Now()
		entry.Lvl = lvl
		entry.Ctx = l.ctx
		if len(args) == 0 {
			entry.Msg = format
		} else {
			entry.Msg = fmt.Sprintf(format, args...)
		}
		if calldepth > 0 {
			entry.Src = Caller(calldepth)
		}
		l.Handler().Log(entry)
		entryPool.Put(entry)
	}
	switch lvl {
	case CRIT:
		atomic.AddUint64(&l.stats.Crit, 1)
	case ERR:
		atomic.AddUint64(&l.stats.Err, 1)
	case WARN:
		atomic.AddUint64(&l.stats.Warn, 1)
	case INFO:
		atomic.AddUint64(&l.stats.Info, 1)
	case DEBUG:
		atomic.AddUint64(&l.stats.Debug, 1)
	}
}

// Fatalf is equivalent to LogDepthf(0, CRIT, format, args...). It also
// causes the process to exit with code 255: `os.Exit(255)`
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.LogDepthf(1, CRIT, format, args...)
	os.Exit(255)
}

// Criticalf is equivalent to LogDepthf(0, CRIT, format, args...)
func (l *logger) Criticalf(format string, args ...interface{}) {
	l.LogDepthf(1, CRIT, format, args...)
}

// Errorf is equivalent to LogDepthf(0, ERR, format, args...)
func (l *logger) Errorf(format string, args ...interface{}) {
	l.LogDepthf(1, ERR, format, args...)
}

// Warningf is equivalent to LogDepthf(0, WARN, format, args...)
func (l *logger) Warningf(format string, args ...interface{}) {
	l.LogDepthf(1, WARN, format, args...)
}

// Infof is equivalent to LogDepthf(-1, INFO, format, args...)
func (l *logger) Infof(format string, args ...interface{}) {
	l.LogDepthf(-1, INFO, format, args...)
}

// Debugf is equivalent to LogDepthf(-1, DEBUG, format, args...)
func (l *logger) Debugf(format string, args ...interface{}) {
	l.LogDepthf(-1, DEBUG, format, args...)
}

func (l *logger) readStats(s *Stats) {
	s.Crit = atomic.LoadUint64(&l.stats.Crit)
	s.Err = atomic.LoadUint64(&l.stats.Err)
	s.Warn = atomic.LoadUint64(&l.stats.Warn)
	s.Info = atomic.LoadUint64(&l.stats.Info)
	s.Debug = atomic.LoadUint64(&l.stats.Debug)
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

// Fatalf is equivalent to LogDepthf(0, CRIT, format, args...). It also
// causes the process to exit with code 255: `os.Exit(255)`
func Fatalf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, CRIT, format, args...)
	os.Exit(255)
}

// Criticalf is equivalent to LogDepthf(0, CRIT, format, args...)
func Criticalf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, CRIT, format, args...)
}

// Errorf is equivalent to LogDepthf(0, ERR, format, args...)
func Errorf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, ERR, format, args...)
}

// Warningf is equivalent to LogDepthf(0, WARN, format, args...)
func Warningf(format string, args ...interface{}) {
	defaultL.LogDepthf(1, WARN, format, args...)
}

// Infof is equivalent to LogDepthf(-1, INFO, format, args...)
func Infof(format string, args ...interface{}) {
	defaultL.LogDepthf(-1, INFO, format, args...)
}

// Debugf is equivalent to LogDepthf(-1, DEBUG, format, args...)
func Debugf(format string, args ...interface{}) {
	defaultL.LogDepthf(-1, DEBUG, format, args...)
}

// ReadStats fill in the provided Stats struct with internal metrics.
func ReadStats(s *Stats) {
	defaultL.readStats(s)
}

// Caller returns the file:line of the caller at the requested callstack
// depth. A depth of 0 is the caller of this function.
func Caller(depth int) string {
	if depth < 0 {
		return ""
	}
	_, file, line, ok := runtime.Caller(depth + 1)
	if !ok {
		return ""
	}
	short := file
	depth = 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			depth++
			if depth == 2 {
				break
			}
		}
	}
	return short + ":" + strconv.Itoa(line)
}
