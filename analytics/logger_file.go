package analytics

import (
	"encoding/json"
	"fmt"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultMaxFileEvents = 100 << 10
	DefaultMaxFileAge    = time.Minute * 10
)

const (
	liveSuffix = ".live"
)

type logFile struct {
	key  string
	path string
	f    *os.File
	enc  *json.Encoder
	t    time.Time
	n    int
}

type fileLogger struct {
	path      string
	eventCh   chan []Event
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
	l.eventCh = make(chan []Event, 32)
	go l.loop()
	return nil
}

func (l *fileLogger) Stop() error {
	close(l.eventCh)
	return nil
}

func (l *fileLogger) WriteEvents(events []Event) {
	l.eventCh <- events
}

func (l *fileLogger) recover() {
	// Rename all files that were previously alive when server was stopped
	filepath.Walk(l.path, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, liveSuffix) {
			// Rename the file to remove the .live suffix
			newPath := path[:len(path)-len(liveSuffix)]
			if err := os.Rename(path, newPath); err != nil {
				golog.Errorf("Failed to rename analytics log: %s", err.Error())
			}
		}
		return nil
	})
}

func (l *fileLogger) loop() {
	l.recover()
	if l.logFiles == nil {
		l.logFiles = make(map[string]*logFile)
	}
	for ev := range l.eventCh {
		l.writeEvents(ev)
	}
	for _, lf := range l.logFiles {
		l.closeFile(lf)
	}
	l.logFiles = nil
}

func (l *fileLogger) writeEvents(events []Event) {
	for _, e := range events {
		l.writeEvent(e)
	}

	// Close files with max events or past max age, and flush/sync
	// files that remain open.

	now := time.Now()
	for _, lf := range l.logFiles {
		if lf.n > l.maxEvents || now.Sub(lf.t) > l.maxAge {
			l.closeFile(lf)
		} else if err := lf.f.Sync(); err != nil {
			golog.Errorf("Failed to sync log file '%s': %s", lf.path, err.Error())
			l.closeFile(lf)
		}
	}
}

func (l *fileLogger) writeEvent(ev Event) {
	lf, err := l.fileForEvent(ev)
	if err != nil {
		golog.Errorf("Failed to get file for event: %s", err.Error())
		return
	}

	if err := lf.enc.Encode(ev); err != nil {
		golog.Errorf("Failed to encode log event: %s", err.Error())
	}
	lf.n++
}

func (l *fileLogger) closeFile(lf *logFile) {
	if lf == nil {
		return
	}
	delete(l.logFiles, lf.key)
	lf.f.Close()
	// Rename the file to remove the .live suffix
	newPath := lf.path[:len(lf.path)-len(liveSuffix)]
	if err := os.Rename(lf.path, newPath); err != nil {
		golog.Errorf("Failed to rename analytics log: %s", err.Error())
	}
}

func (l *fileLogger) fileForEvent(ev Event) (*logFile, error) {
	// Check for an existing file

	pth := filepath.Join(l.path, ev.Category(), ev.Time().UTC().Format("2006/01/02"))
	if lf := l.logFiles[pth]; lf != nil {
		return lf, nil
	}

	// Create a new file

	id, err := idgen.NewID()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(pth, 0700); err != nil {
		golog.Errorf("Failed to create a log path '%s': %s", pth, err.Error())
		return nil, err
	}
	fullPath := filepath.Join(pth, fmt.Sprintf("%d.js%s", id, liveSuffix))
	f, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		golog.Errorf("Failed to create a new log file '%s': %s", fullPath, err.Error())
		return nil, err
	}
	lf := &logFile{
		key:  pth,
		path: fullPath,
		f:    f,
		enc:  json.NewEncoder(f),
		t:    time.Now(),
		n:    0,
	}
	l.logFiles[pth] = lf
	return lf, nil
}

func validateLogPath(logPath string) bool {
	st, err := os.Stat(logPath)
	if err != nil {
		return false
	}
	return st.IsDir()
}
