package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// local is a store that uses the local filesystem.
type local struct {
	path string
}

// NewLocalStore initializes a new local file storage creating the path if necessary.
// WARNING: It is not safe to use this in production. There are no checks that files
// aren't read outside of the intended path. It should be safe if the media ID is
// only from a trusted source.
func NewLocalStore(path string) (Store, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("storage.NewLocalStore: failed to make path '%s' absolute: %s", path, err)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("storage.NewLocalStore: failed create path '%s': %s", path, err)
	}
	return &local{
		path: path,
	}, nil
}

func (s *local) IDFromName(name string) string {
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
	return filepath.Join(s.path, name)
}

func (s *local) Put(name string, data []byte, contentType string, meta map[string]string) (string, error) {
	return s.PutReader(name, bytes.NewReader(data), int64(len(data)), contentType, meta)
}

func (s *local) PutReader(name string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	// TODO: support contentType & meta
	fullPath := s.IDFromName(name)
	if !strings.HasPrefix(fullPath, s.path) {
		return "", fmt.Errorf("storage.Local: invalid name '%s'", name)
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(fullPath) // cleanup on failure
		return "", err
	}
	if err := f.Sync(); err != nil {
		return "", err
	}
	f, err = os.Create(fullPath + ".meta")
	defer f.Close()
	if meta == nil {
		meta = map[string]string{}
	}
	meta["Content-Length"] = strconv.FormatInt(size, 10)
	meta["Content-Type"] = contentType
	if err := json.NewEncoder(f).Encode(meta); err != nil {
		os.Remove(fullPath)
		os.Remove(fullPath + ".meta")
		return "", err
	}
	return fullPath, f.Sync()
}

func (s *local) Get(id string) ([]byte, http.Header, error) {
	rdc, headers, err := s.GetReader(id)
	if err != nil {
		return nil, nil, err
	}
	defer rdc.Close()
	b, err := ioutil.ReadAll(rdc)
	return b, headers, err
}

func (s *local) GetHeader(id string) (http.Header, error) {
	return localHeader(id)
}

func localHeader(id string) (http.Header, error) {
	f, err := os.Open(id + ".meta")
	if os.IsNotExist(err) {
		return nil, ErrNoObject
	} else if err != nil {
		return nil, err
	}
	defer f.Close()
	var meta map[string]string
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return nil, err
	}
	h := http.Header{}
	for k, v := range meta {
		h.Set(k, v)
	}
	return h, nil
}

func (s *local) GetReader(id string) (io.ReadCloser, http.Header, error) {
	h, err := localHeader(id)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(id)
	if err != nil {
		return nil, nil, err
	}
	return f, h, nil
}

func (s *local) Delete(id string) error {
	os.Remove(id + ".meta")
	return os.Remove(id)
}

func (s *local) ExpiringURL(id string, expiration time.Duration) (string, error) {
	return id, nil
}
