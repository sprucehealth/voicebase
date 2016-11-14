package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type readCloser struct {
	io.Reader
}

func (readCloser) Close() error {
	return nil
}

type TestObject struct {
	Data    []byte
	Headers http.Header
}

type testStore struct {
	objects map[string]*TestObject
	mu      sync.Mutex
}

func NewTestStore(objects map[string]*TestObject) Store {
	if objects == nil {
		objects = make(map[string]*TestObject)
	}
	return &testStore{
		objects: objects,
	}
}

func (s *testStore) Put(id string, data []byte, contentType string, meta map[string]string) (string, error) {
	s.mu.Lock()
	headers := http.Header{}
	headers.Set("Content-Length", strconv.Itoa(len(data)))
	headers.Set("Content-Type", contentType)
	for k, v := range meta {
		headers.Set(k, v)
	}
	s.objects[id] = &TestObject{data, headers}
	s.mu.Unlock()
	return id, nil
}

func (s *testStore) PutReader(id string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return s.Put(id, data, contentType, meta)
}

func (s *testStore) Get(id string) ([]byte, http.Header, error) {
	s.mu.Lock()
	o := s.objects[id]
	s.mu.Unlock()
	if o == nil {
		return nil, nil, ErrNoObject
	}
	return o.Data, o.Headers, nil
}

func (s *testStore) GetHeader(id string) (http.Header, error) {
	_, headers, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func (s *testStore) GetReader(id string) (io.ReadCloser, http.Header, error) {
	data, headers, err := s.Get(id)
	if err != nil {
		return nil, nil, err
	}
	return readCloser{bytes.NewReader(data)}, headers, nil
}

func (s *testStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.objects, id)
	s.mu.Unlock()
	return nil
}

func (s *testStore) ExpiringURL(id string, duration time.Duration) (string, error) {
	return id, nil
}

func (s *testStore) Copy(dstID, srcID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[dstID] = s.objects[srcID]
	return nil
}
