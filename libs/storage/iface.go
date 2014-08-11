package storage

import (
	"io"
	"net/http"
)

type Store interface {
	Put(name string, data []byte, headers http.Header) (string, error)
	PutReader(name string, r io.Reader, size int64, headers http.Header) (string, error)
	Get(id string) ([]byte, http.Header, error)
	GetReader(id string) (io.ReadCloser, http.Header, error)
	GetSignedURL(id string) (string, error)
	Delete(id string) error
}
