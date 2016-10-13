package media

import (
	"io"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
)

// BinaryMeta is it's media metadata
type BinaryMeta struct {
	MimeType string
	Size     uint64
	URL      string
}

// BinaryService implements a media storage service.
type BinaryService struct {
	*mediaStorage
	maxSizeBytes int
}

// NewBinaryService returns a new initialized media service.
func NewBinaryService(store, storeCache storage.DeterministicStore, maxSizeBytes int) *BinaryService {
	return &BinaryService{
		mediaStorage: &mediaStorage{store: store, storeCache: storeCache},
		maxSizeBytes: maxSizeBytes,
	}
}

// PutReader stores an binary segment and returns the binary metadata
func (s *BinaryService) PutReader(id string, r io.ReadSeeker, contentType string) (*BinaryMeta, error) {
	// Figure out the size of the data
	size, err := SeekerSize(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	url, err := s.store.PutReader(id, r, size, contentType, nil)
	meta := &BinaryMeta{
		MimeType: contentType,
		Size:     uint64(size),
		URL:      url,
	}
	return meta, errors.Trace(err)
}

// Copy a stored binary file
func (s *BinaryService) Copy(dstID, srcID string) (string, error) {
	if err := s.store.Copy(s.store.IDFromName(dstID), s.store.IDFromName(srcID)); err != nil {
		if errors.Cause(err) == storage.ErrNoObject {
			return "", ErrNotFound
		}
		return "", errors.Trace(err)
	}
	return s.store.IDFromName(dstID), nil

}
