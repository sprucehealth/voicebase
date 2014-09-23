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

func NewTestStore(objects map[string]*TestObject) Store {
	if objects == nil {
		objects = make(map[string]*TestObject)
	}
	return &testStore{
		objects: objects,
	}
}

func (s *testStore) Put(name string, data []byte, headers http.Header) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[name] = &TestObject{data, headers}
	return name, nil
}

func (s *testStore) PutReader(name string, r io.Reader, size int64, headers http.Header) (string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return s.Put(name, data, headers)
}

func (s *testStore) Get(id string) ([]byte, http.Header, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.objects[id]
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

func (s *testStore) GetSignedURL(id string, expires time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return id, nil
}

func (s *testStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, id)
	return nil
}
