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
		&ClientEvent{ID: 1},
		&ClientEvent{ID: 2},
	})
	l.WriteEvents([]Event{
		&ClientEvent{ID: 3},
		&ClientEvent{ID: 4},
		&ClientEvent{ID: 5},
	})
	l.WriteEvents([]Event{
		&ClientEvent{ID: 6},
		&ClientEvent{ID: 7},
	})

	time.Sleep(time.Millisecond * 10)

	var liveFiles int
	var jsFiles int
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".js") {
			jsFiles++
		} else if strings.HasSuffix(path, ".live") {
			liveFiles++
		}
		return nil
	})
	if liveFiles != 1 {
		t.Errorf("Expected 1 live file. Got %d", liveFiles)
	}
	if jsFiles != 1 {
		t.Errorf("Expected 1 js file. Got %d", jsFiles)
	}

	l.(*fileLogger).recover()

	liveFiles = 0
	jsFiles = 0
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".js") {
			jsFiles++
		} else if strings.HasSuffix(path, ".live") {
			liveFiles++
		}
		return nil
	})
	if liveFiles != 0 {
		t.Errorf("Expected 0 live files. Got %d", liveFiles)
	}
	if jsFiles != 2 {
		t.Errorf("Expected 2 js files. Got %d", jsFiles)
	}
}
