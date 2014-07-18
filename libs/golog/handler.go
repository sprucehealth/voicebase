package golog

import (
	"io"
	"log/syslog"
)

type HandlerFunc func(e *Entry) error

func WriterHandler(w io.Writer, fmtr Formatter) Handler {
	return &writerHandler{w: w, fmtr: fmtr}
}

func SyslogHandler(tag string, fmtr Formatter) (Handler, error) {
	w, err := syslog.New(syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &syslogHandler{w: w, f: fmtr}, nil
}

// SplitHandler sends all entries with level above lvl to
// the high handler, and all other entries to the low handler.
func SplitHandler(lvl Level, low, high Handler) Handler {
	return &splitHandler{lvl: lvl, low: low, high: high}
}

func (h HandlerFunc) Log(e *Entry) error {
	return h(e)
}

type writerHandler struct {
	w    io.Writer
	fmtr Formatter
}

func (h *writerHandler) Log(e *Entry) error {
	_, err := h.w.Write(h.fmtr.Format(e))
	return err
}

type syslogHandler struct {
	w *syslog.Writer
	f Formatter
}

func (h *syslogHandler) Log(e *Entry) error {
	msg := string(h.f.Format(e))
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
