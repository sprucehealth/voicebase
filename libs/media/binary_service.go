package media

import (
	"io"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
)

// BinaryMeta is it's media metadata
type BinaryMeta struct {
	ID       string
	Name     string
	MimeType string
	Size     uint64
	// The following fields are only available for some media types
	Width    int
	Height   int
	Duration time.Duration
}

// BinaryService implements a media storage service.
type BinaryService struct {
	*mediaStorage
	maxSizeBytes int
}

// NewBinaryService returns a new initialized media service.
func NewBinaryService(store, storeCache storage.Store, maxSizeBytes int) *BinaryService {
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
		ID:       id,
		MimeType: contentType,
		Size:     uint64(size),
	}
	return meta, errors.Trace(err)
}

// GetMeta returns the metadata associated with a media entry
func (s *BinaryService) GetMeta(id string) (*BinaryMeta, error) {
	h, err := s.store.GetHeader(id)
	if err != nil {
		return nil, errors.Wrapf(err, "mediaID=%q", id)
	}
	width, _ := strconv.Atoi(h.Get(widthHeader))
	height, _ := strconv.Atoi(h.Get(heightHeader))
	size, _ := strconv.ParseUint(h.Get(contentLengthHeader), 10, 64)
	durationNS, _ := strconv.ParseInt(h.Get(durationHeader), 10, 64)
	return &BinaryMeta{
		Name:     h.Get(originalNameHeader),
		MimeType: h.Get(mimeTypeHeader),
		Width:    width,
		Height:   height,
		Size:     size,
		Duration: time.Duration(durationNS),
	}, nil
}

// Copy a stored binary file
func (s *BinaryService) Copy(dstID, srcID string) error {
	if err := s.store.Copy(dstID, srcID); err != nil {
		if errors.Cause(err) == storage.ErrNoObject {
			return errors.Wrapf(ErrNotFound, "mediaID=%q", srcID)
		}
		return errors.Trace(err)
	}
	return nil

}
