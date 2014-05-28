package analytics

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileLogger(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "spruce-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	l, err := NewFileLogger(tmpDir, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Start(); err != nil {
		t.Fatal(err)
	}
	defer l.Stop()

	l.WriteEvents([]Event{
		&ClientEvent{ID: 1, Timestamp: Time(time.Now())},
		&ClientEvent{ID: 2, Timestamp: Time(time.Now())},
	})
	l.WriteEvents([]Event{
		&ClientEvent{ID: 3, Timestamp: Time(time.Now())},
		&ClientEvent{ID: 4, Timestamp: Time(time.Now())},
		&ClientEvent{ID: 5, Timestamp: Time(time.Now())},
	})
	l.WriteEvents([]Event{
		&ClientEvent{ID: 6, Timestamp: Time(time.Now())},
		&ClientEvent{ID: 7, Timestamp: Time(time.Now())},
	})
	l.WriteEvents([]Event{
		&ClientEvent{ID: 8, Timestamp: Time(time.Now().AddDate(0, 0, -1))},
		&ClientEvent{ID: 9, Timestamp: Time(time.Now().AddDate(0, 0, -1))},
	})

	time.Sleep(time.Millisecond * 10)

	var liveFiles []string
	var jsFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".js") {
			jsFiles = append(jsFiles, path)
		} else if strings.HasSuffix(path, ".live") {
			liveFiles = append(liveFiles, path)
		}
		return nil
	})
	if len(liveFiles) != 2 {
		t.Log("Live files:", liveFiles)
		t.Errorf("Expected 2 live files. Got %d", len(liveFiles))
	}
	if len(jsFiles) != 1 {
		t.Log("JS Files:", jsFiles)
		t.Errorf("Expected 1 js file. Got %d", len(jsFiles))
	}

	l.(*fileLogger).recover()

	liveFiles = liveFiles[:0]
	jsFiles = jsFiles[:0]
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".js") {
			jsFiles = append(jsFiles, path)
		} else if strings.HasSuffix(path, ".live") {
			liveFiles = append(liveFiles, path)
		}
		return nil
	})
	if len(liveFiles) != 0 {
		t.Log("Live files:", liveFiles)
		t.Errorf("Expected 0 live files. Got %d", len(liveFiles))
	}
	if len(jsFiles) != 3 {
		t.Log("JS Files:", jsFiles)
		t.Errorf("Expected 3 js files. Got %d", len(jsFiles))
	}
}
