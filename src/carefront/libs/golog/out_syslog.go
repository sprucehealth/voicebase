package golog

import (
	"fmt"
	"log/syslog"
)

type SyslogOutput struct {
	w *syslog.Writer
}

func NewSyslogOutput(tag string) (*SyslogOutput, error) {
	w, err := syslog.New(syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &SyslogOutput{w: w}, nil
}

func (o *SyslogOutput) Log(logType string, l Level, msg []byte) error {
	if len(msg) == 0 {
		return nil
	}
	// Inject the logType and level, but first make sure it looks like JSON.
	if msg[len(msg)-1] == '}' {
		msg = append(msg[:len(msg)-1], fmt.Sprintf(`,"_type":"%s","@level":%d}`, logType, int(l))...)
	}
	m := string(msg)
	switch l {
	case CRIT:
		return o.w.Crit(m)
	case ERR:
		return o.w.Err(m)
	case WARN:
		return o.w.Warning(m)
	case INFO:
		return o.w.Info(m)
	case DEBUG:
		return o.w.Debug(m)
	}
	return o.w.Debug(m)
}
