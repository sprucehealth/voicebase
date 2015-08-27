package storage

import (
	"errors"
	"io"
	"net/http"
	"time"
)

// ErrNoObject is returned when an trying to Get an object that doesn't exist
var ErrNoObject = errors.New("storage: no object")

// Store is implemented by storage backends
type Store interface {
	Put(name string, data []byte, contentType string, meta map[string]string) (string, error)
	PutReader(name string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error)
	Get(id string) ([]byte, http.Header, error)
	GetReader(id string) (io.ReadCloser, http.Header, error)
	SignedURL(id string, expires time.Duration) (string, error)
	Delete(id string) error
}

// DeterministicStore is a version of Store that uses a deterministric
// value for ID so that it can be generated from the name given to Put(Reader).
type DeterministicStore interface {
	Store
	IDFromName(name string) string
}

// StoreMap is a collection of named stores
type StoreMap map[string]Store

// MustGet returns the store from the map or panics
func (sm StoreMap) MustGet(name string) Store {
	s, ok := sm[name]
	if !ok {
		panic("Storage " + name + " not found")
	}
	return s
}

// Get returns the store from the map if it exists and nil, false otherwise
func (sm StoreMap) Get(name string) (Store, bool) {
	s, ok := sm[name]
	return s, ok
}
