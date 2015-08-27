package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
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

func NewTestStore(objects map[string]*TestObject) DeterministicStore {
	if objects == nil {
		objects = make(map[string]*TestObject)
	}
	return &testStore{
		objects: objects,
	}
}

func (s *testStore) IDFromName(name string) string {
	return name
}

func (s *testStore) Put(name string, data []byte, contentType string, meta map[string]string) (string, error) {
	s.mu.Lock()
	headers := http.Header{}
	headers.Set("Content-Type", contentType)
	for k, v := range meta {
		headers.Set(k, v)
	}
	s.objects[name] = &TestObject{data, headers}
	s.mu.Unlock()
	return name, nil
}

func (s *testStore) PutReader(name string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return s.Put(name, data, contentType, meta)
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

func (s *testStore) GetReader(id string) (io.ReadCloser, http.Header, error) {
	data, headers, err := s.Get(id)
	if err != nil {
		return nil, nil, err
	}
	return readCloser{bytes.NewReader(data)}, headers, nil
}

func (s *testStore) SignedURL(id string, expires time.Duration) (string, error) {
	return id, nil
}

func (s *testStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.objects, id)
	s.mu.Unlock()
	return nil
}
