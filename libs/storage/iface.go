package storage

import (
	"errors"
	"io"
	"net/http"
	"time"
)

var ErrNoObject = errors.New("storage: no object")

type Store interface {
	Put(name string, data []byte, headers http.Header) (string, error)
	PutReader(name string, r io.Reader, size int64, headers http.Header) (string, error)
	Get(id string) ([]byte, http.Header, error)
	GetReader(id string) (io.ReadCloser, http.Header, error)
	SignedURL(id string, expires time.Time) (string, error)
	Delete(id string) error
}

// DeterministicStore is a version of Store that uses a deterministric
// value for ID so that it can be generated from the name given to Put(Reader).
type DeterministicStore interface {
	Store
	IDFromName(name string) string
}

type StoreMap map[string]Store

func (sm StoreMap) MustGet(name string) Store {
	s, ok := sm[name]
	if !ok {
		panic("Storage " + name + " not found")
	}
	return s
}

func (sm StoreMap) Get(name string) (Store, bool) {
	s, ok := sm[name]
	return s, ok
}
