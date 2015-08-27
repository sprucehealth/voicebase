package storage

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"testing"
)

func TestLocalStore(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "storetest-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpPath)

	tmpPath = path.Join(tmpPath, "storage")

	t.Logf("Path: %s", tmpPath)
	// Non existant path should be created
	store, err := NewLocalStore(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	// Using same path again should be fine
	if _, err := NewLocalStore(tmpPath); err != nil {
		t.Fatal(err)
	}

	id, err := store.Put("foo", []byte("bar"), "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %s", id)

	b, _, err := store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "bar" {
		t.Fatalf("Expected 'bar' got '%s'", string(b))
	}

	// Just make sure this doesn't blow up. Nothing much to test since it's not a real URL as such.
	url, err := store.SignedURL(id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SignedURL: %s", url)

	if err := store.Delete(id); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Get(id); err != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %+v", err)
	}
}
