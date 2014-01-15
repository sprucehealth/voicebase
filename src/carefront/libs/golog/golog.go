package golog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

type Output interface {
	Log(logType string, l Level, msg []byte) error
}

type Level int32

const (
	CRIT  Level = iota // For panics (code bugs)
	ERR                // General errors (e.g. errors from database, etc)
	WARN               // e.g. correctable but inconsistent state
	INFO               // e.g. web access logs, analytics, ...
	DEBUG              // Normally turned off but can help to track down issues
)

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
	return strconv.FormatInt(int64(l), 10)
}

type loggingT struct {
	mu      sync.RWMutex
	level   Level
	output  Output
	enabled map[string]bool // Enabled log types outside (only applies when current level is lower than the event's level)
}

var logging = loggingT{
	output: DefaultOutput,
	level:  INFO,
}

type Message struct {
	Message string `json:"@message"`
}

type writer struct{}

var Writer io.Writer = writer{}

func (w writer) Write(b []byte) (int, error) {
	m := string(b)
	Infof(m)
	return len(m), nil
}

func SetLevel(l Level) Level {
	return Level(atomic.SwapInt32((*int32)(&logging.level), int32(l)))
}

func GetLevel() Level {
	return Level(atomic.LoadInt32((*int32)(&logging.level)))
}

func SetEnabled(logType string, enabled bool) {
	if logType == "log" {
		return
	}
	logging.mu.Lock()
	logging.enabled[logType] = enabled
	logging.mu.Unlock()
}

func GetEnabled(logType string) bool {
	if logType == "log" {
		return false
	}
	logging.mu.RLock()
	enabled := logging.enabled[logType]
	logging.mu.RUnlock()
	return enabled
}

func SetOutput(o Output) {
	logging.mu.Lock()
	logging.output = o
	logging.mu.Unlock()
}

// Return true if the current level is greater than or equal to l or the logType is explicitly enable
func L(logType string, l Level) bool {
	return GetLevel() >= l || GetEnabled(logType)
}

func log(logType string, l Level, v interface{}) error {
	if s, ok := v.(string); ok {
		v = &Message{Message: s}
	}
	msg, err := json.Marshal(v)
	if err != nil {
		msg, err = json.Marshal(&Message{Message: fmt.Sprintf("%+v", v)})
		if err != nil {
			return err
		}
	}
	logging.mu.Lock()
	err = logging.output.Log(logType, l, msg)
	logging.mu.Unlock()
	return err
}

func Log(logType string, l Level, v interface{}) error {
	if L(logType, l) {
		return log(logType, l, v)
	}
	return nil
}

func Fatalf(format string, args ...interface{}) {
	log("log", CRIT, fmt.Sprintf(format, args...))
	os.Exit(255)
}

func Criticalf(format string, args ...interface{}) {
	if L("log", CRIT) {
		log("log", CRIT, fmt.Sprintf(format, args...))
	}
}

func Errorf(format string, args ...interface{}) {
	if L("log", ERR) {
		log("log", ERR, fmt.Sprintf(format, args...))
	}
}

func Warningf(format string, args ...interface{}) {
	if L("log", WARN) {
		log("log", WARN, fmt.Sprintf(format, args...))
	}
}

func Infof(format string, args ...interface{}) {
	if L("log", INFO) {
		log("log", INFO, fmt.Sprintf(format, args...))
	}
}

func Debugf(format string, args ...interface{}) {
	if L("log", DEBUG) {
		log("log", DEBUG, fmt.Sprintf(format, args...))
	}
}
