package analytics

import (
	"carefront/libs/golog"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	DefaultMaxFileEvents = 100 << 10
	DefaultMaxFileAge    = time.Minute * 10
)

type logItem struct {
	category string
	events   []interface{}
}

type logFile struct {
	p string
	f *os.File
	e *json.Encoder
	t time.Time
	n int
}

type fileLogger struct {
	path      string
	eventCh   chan logItem
	logFiles  map[string]*logFile
	maxEvents int
	maxAge    time.Duration
}

func NewFileLogger(path string, maxEvents int, maxAge time.Duration) (Logger, error) {
	if !validateLogPath(path) {
		return nil, fmt.Errorf("analytics: path '%s' not valid (must be an existing directory)", path)
	}
	if maxEvents <= 0 {
		maxEvents = DefaultMaxFileEvents
	}
	if maxAge == 0 {
		maxAge = DefaultMaxFileAge
	}
	return &fileLogger{
		path:      path,
		maxEvents: maxEvents,
		maxAge:    maxAge,
	}, nil
}

func (l *fileLogger) Start() error {
	l.eventCh = make(chan logItem, 32)
	go l.loop()
	return nil
}

func (l *fileLogger) Stop() error {
	close(l.eventCh)
	return nil
}

func (l *fileLogger) WriteEvents(category string, events []interface{}) {
	l.eventCh <- logItem{category: category, events: events}
}

func (l *fileLogger) loop() {
	if l.logFiles == nil {
		l.logFiles = make(map[string]*logFile)
	}
	for it := range l.eventCh {
		l.writeEvents(it.category, it.events)
	}
	for _, f := range l.logFiles {
		f.f.Close()
	}
	l.logFiles = nil
}

func (l *fileLogger) writeEvents(category string, events []interface{}) {
	lf := l.logFiles[category]
	if lf == nil || lf.n > l.maxEvents || time.Now().Sub(lf.t) > l.maxAge {
		var err error
		lf, err = l.newFile(category)
		if err != nil {
			l.logFiles[category] = nil
			return
		}
		l.logFiles[category] = lf
	}

	for _, e := range events {
		if err := lf.e.Encode(e); err != nil {
			golog.Errorf("Failed to encode log event: %s", err.Error())
		}
	}

	lf.n += len(events)
	if err := lf.f.Sync(); err != nil {
		golog.Errorf("Failed to sync log file '%s': %s", lf.p, err.Error())
		lf.f.Close()
		l.logFiles[category] = nil
	}
}

func (l *fileLogger) newFile(category string) (*logFile, error) {
	now := time.Now()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%d.js", now.UTC().Format("2006/01/02/150405"), id)
	pth := filepath.Join(l.path, category, name)
	if err := os.MkdirAll(path.Dir(pth), 0700); err != nil {
		golog.Errorf("Failed to create a log path '%s': %s", path.Dir(pth), err.Error())
		return nil, err
	}
	f, err := os.OpenFile(pth, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		golog.Errorf("Failed to create a new log file '%s': %s", pth, err.Error())
		return nil, err
	}
	return &logFile{
		p: pth,
		f: f,
		e: json.NewEncoder(f),
		t: now,
		n: 0,
	}, nil
}

func validateLogPath(logPath string) bool {
	st, err := os.Stat(logPath)
	if err != nil {
		return false
	}
	return st.IsDir()
}
