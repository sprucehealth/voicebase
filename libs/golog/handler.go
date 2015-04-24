package golog

import (
	"io"
	"log/syslog"
	"sync"
)

// A Handler can be registered as log entry destinations.
type Handler interface {
	// Log is called for every log entry for which the log level is enabled.
	// The provided Entry object is reused and should never be held longer than
	// the duration of the call.
	Log(e *Entry) error
}

// The HandlerFunc type is an adapter to allow the use of an ordinary functionsas a Handler.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object
// that calls f.
type HandlerFunc func(e *Entry) error

// WriterHandler formats all log entries with the provided
// formatter and writes the formatted entry to the writer.
func WriterHandler(w io.Writer, fmtr Formatter) Handler {
	return &writerHandler{w: w, fmtr: fmtr}
}

// SyslogHandler formats all log entries with the provided
// formatter and writes the formatted entry to the local syslog
// with priority of USER and the given tag.
func SyslogHandler(tag string, fmtr Formatter) (Handler, error) {
	w, err := syslog.New(syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &syslogHandler{w: w, fmtr: fmtr}, nil
}

// SplitHandler sends all entries with level above lvl to
// the high handler, and all other entries to the low handler.
func SplitHandler(lvl Level, low, high Handler) Handler {
	return &splitHandler{lvl: lvl, low: low, high: high}
}

// Log calls f(e)
func (h HandlerFunc) Log(e *Entry) error {
	return h(e)
}

type writerHandler struct {
	w    io.Writer
	fmtr Formatter
	mu   sync.Mutex
}

func (h *writerHandler) Log(e *Entry) error {
	h.mu.Lock()
	_, err := h.w.Write(h.fmtr.Format(e))
	h.mu.Unlock()
	return err
}

type syslogHandler struct {
	w    *syslog.Writer
	fmtr Formatter
	mu   sync.Mutex
}

func (h *syslogHandler) Log(e *Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	msg := string(h.fmtr.Format(e))
	switch e.Lvl {
	case CRIT:
		return h.w.Crit(msg)
	case ERR:
		return h.w.Err(msg)
	case WARN:
		return h.w.Warning(msg)
	case INFO:
		return h.w.Info(msg)
	case DEBUG:
		return h.w.Debug(msg)
	}
	return h.w.Debug(msg)
}

type splitHandler struct {
	low, high Handler
	lvl       Level
}

func (h *splitHandler) Log(e *Entry) error {
	if e.Lvl <= h.lvl {
		return h.low.Log(e)
	}
	return h.high.Log(e)
}
