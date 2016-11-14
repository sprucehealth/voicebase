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

	"github.com/sprucehealth/backend/libs/errors"
)

const fsMetaSuffix = ".meta"

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

func (s *local) pathForID(id string) string {
	if strings.HasPrefix(id, "/") {
		id = id[1:]
	}
	return filepath.Join(s.path, id)
}

func (s *local) Put(id string, data []byte, contentType string, meta map[string]string) (string, error) {
	return s.PutReader(id, bytes.NewReader(data), int64(len(data)), contentType, meta)
}

func (s *local) PutReader(id string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	// TODO: support contentType & meta
	fullPath := s.pathForID(id)
	if !strings.HasPrefix(fullPath, s.path) {
		return "", fmt.Errorf("storage.Local: invalid id %q", id)
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
		os.Remove(fullPath)
		return "", err
	}
	f, err = os.Create(fullPath + fsMetaSuffix)
	defer f.Close()
	if meta == nil {
		meta = map[string]string{}
	}
	meta["Content-Length"] = strconv.FormatInt(size, 10)
	meta["Content-Type"] = contentType
	if err := json.NewEncoder(f).Encode(meta); err != nil {
		os.Remove(fullPath)
		os.Remove(fullPath + fsMetaSuffix)
		return "", err
	}
	return id, f.Sync()
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
	return localHeader(s.pathForID(id))
}

func localHeader(path string) (http.Header, error) {
	f, err := os.Open(path + fsMetaSuffix)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(ErrNoObject, "path=%q", path)
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
	path := s.pathForID(id)
	h, err := localHeader(path)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, h, nil
}

func (s *local) Delete(id string) error {
	path := s.pathForID(id)
	os.Remove(path + fsMetaSuffix)
	return os.Remove(path)
}

func (s *local) ExpiringURL(id string, expiration time.Duration) (string, error) {
	return s.pathForID(id), nil
}

func (s *local) Copy(dstID, srcID string) (err error) {
	dstPath := s.pathForID(dstID)
	srcPath := s.pathForID(srcID)
	defer func() {
		if err != nil {
			os.Remove(dstPath)
			os.Remove(dstPath + fsMetaSuffix)
		}
	}()
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	srcMeta, err := os.Open(srcPath + fsMetaSuffix)
	if err != nil {
		return err
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	dstMeta, err := os.Create(dstPath + fsMetaSuffix)
	if err != nil {
		return err
	}
	defer dstMeta.Close()
	if _, err := io.Copy(dstMeta, srcMeta); err != nil {
		return err
	}
	return dstMeta.Sync()
}
