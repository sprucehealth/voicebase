package media

import (
	"io"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/audioutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
)

// AudioMeta is it's media metadata
type AudioMeta struct {
	MimeType string
	Duration time.Duration
	Size     uint64
	URL      string
}

// AudioService implements a media storage service.
type AudioService struct {
	*mediaStorage
	maxSizeBytes int
}

// NewAudioService returns a new initialized media service.
func NewAudioService(store, storeCache storage.DeterministicStore, maxSizeBytes int) *AudioService {
	return &AudioService{
		mediaStorage: &mediaStorage{store: store, storeCache: storeCache},
		maxSizeBytes: maxSizeBytes,
	}
}

// PutReader stores an audio segment and returns the audio metadata
func (s *AudioService) PutReader(id string, r io.ReadSeeker, contentType string) (*AudioMeta, error) {
	// TODO: Figure out how we can avoid reading the whole stream before beginning the upload
	// TODO: Build out support for more robust duration detection
	// TODO: Detect empty audio segments or white noise
	// Figure out the size of the data
	size, err := SeekerSize(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Figure out the duration of the audio
	duration, err := audioutil.Duration(r, contentType)
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
	meta := &AudioMeta{
		MimeType: contentType,
		Duration: duration,
		Size:     uint64(size),
		URL:      url,
	}
	return meta, errors.Trace(err)
}

// Copy a stored audio file
func (s *AudioService) Copy(dstID, srcID string) (string, error) {
	if err := s.store.Copy(s.store.IDFromName(dstID), s.store.IDFromName(srcID)); err != nil {
		if errors.Cause(err) == storage.ErrNoObject {
			return "", errors.Wrapf(ErrNotFound, "mediaID=%q", srcID)
		}
		return "", errors.Trace(err)
	}
	return s.store.IDFromName(dstID), nil
}
