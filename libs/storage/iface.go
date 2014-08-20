package storage

import (
	"io"
	"net/http"
	"time"
)

type Store interface {
	Put(name string, data []byte, headers http.Header) (string, error)
	PutReader(name string, r io.Reader, size int64, headers http.Header) (string, error)
	Get(id string) ([]byte, http.Header, error)
	GetReader(id string) (io.ReadCloser, http.Header, error)
	GetSignedURL(id string, expires time.Time) (string, error)
	Delete(id string) error
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
