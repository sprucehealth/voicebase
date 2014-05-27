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

	l.WriteEvents("cat1", []interface{}{
		&ClientEvent{ID: 1},
		&ClientEvent{ID: 2},
	})
	l.WriteEvents("cat1", []interface{}{
		&ClientEvent{ID: 3},
		&ClientEvent{ID: 4},
		&ClientEvent{ID: 5},
	})
	l.WriteEvents("cat1", []interface{}{
		&ClientEvent{ID: 6},
		&ClientEvent{ID: 7},
	})
	l.WriteEvents("cat2", []interface{}{
		&ClientEvent{ID: 8},
		&ClientEvent{ID: 9},
	})

	time.Sleep(time.Millisecond * 10)

	var files []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".js") {
			files = append(files, path)
		}
		return nil
	})

	t.Log(files)

	if len(files) != 3 {
		t.Errorf("Expected 3 log files. Got %d", len(files))
	}
}
