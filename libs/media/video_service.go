package media

import (
	"io"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/videoutil"
)

// VideoMeta is it's media metadata
type VideoMeta struct {
	MimeType string
	Duration time.Duration
	Size     uint64
	URL      string
}

// VideoService implements a media storage service.
type VideoService struct {
	*mediaStorage
	maxSizeBytes int
}

// NewVideoService returns a new initialized media service.
func NewVideoService(store, storeCache storage.DeterministicStore, maxSizeBytes int) *VideoService {
	return &VideoService{
		mediaStorage: &mediaStorage{store: store, storeCache: storeCache},
		maxSizeBytes: maxSizeBytes,
	}
}

// PutReader stores an binary segment and returns the binary metadata
func (s *VideoService) PutReader(id string, r io.ReadSeeker, contentType string) (*VideoMeta, error) {
	// TODO: Figure out how we can avoid reading the whole stream before beginning the upload
	// TODO: Build out support for duration detection
	// TODO: Detect empty video files
	// Figure out the size of the data
	size, err := SeekerSize(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Figure out the duration of the video
	duration, err := videoutil.Duration(r, contentType)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Trace(err)
	}

	url, err := s.store.PutReader(id, r, size, contentType, map[string]string{
		durationHeader: strconv.FormatInt(duration.Nanoseconds(), 10),
	})
	meta := &VideoMeta{
		MimeType: contentType,
		Duration: duration,
		Size:     uint64(size),
		URL:      url,
	}
	return meta, errors.Trace(err)
}
