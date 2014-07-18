package golog

import (
	"io"
	"log/syslog"
)

type HandlerFunc func(e *Entry) error

func IOHandler(out, err io.Writer, fmtr Formatter) Handler {
	return &ioHandler{out: out, err: err, fmtr: fmtr}
}

func SyslogHandler(tag string, fmtr Formatter) (Handler, error) {
	w, err := syslog.New(syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &syslogHandler{w: w, f: fmtr}, nil
}

func (h HandlerFunc) Log(e *Entry) error {
	return h(e)
}

type ioHandler struct {
	out, err io.Writer
	fmtr     Formatter
}

func (o *ioHandler) Log(e *Entry) error {
	m := o.fmtr.Format(e)
	if e.Lvl <= WARN {
		_, err := o.err.Write(m)
		return err
	}
	_, err := o.out.Write(m)
	return err
}

type syslogHandler struct {
	w *syslog.Writer
	f Formatter
}

func (o *syslogHandler) Log(e *Entry) error {
	msg := string(o.f.Format(e))
	switch e.Lvl {
	case CRIT:
		return o.w.Crit(msg)
	case ERR:
		return o.w.Err(msg)
	case WARN:
		return o.w.Warning(msg)
	case INFO:
		return o.w.Info(msg)
	case DEBUG:
		return o.w.Debug(msg)
	}
	return o.w.Debug(msg)
}
