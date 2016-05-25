package media

import (
	"io"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
)

// BinaryMeta is it's media metadata
type BinaryMeta struct {
	MimeType string
	Size     int
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

	_, err = s.store.PutReader(id, r, size, contentType, nil)
	meta := &BinaryMeta{
		MimeType: contentType,
		Size:     int(size),
	}
	return meta, errors.Trace(err)
}
