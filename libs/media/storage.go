package media

import (
	"io"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/storage"
)

// mediaStore represents all the common storage related operations for media services
type mediaStorage struct {
	store      storage.Store
	storeCache storage.Store
}

// GetReader returns a reader for the requested media id
func (s *mediaStorage) GetReader(id string) (io.ReadCloser, error) {
	rc, _, err := s.store.GetReader(id)
	if errors.Cause(err) == storage.ErrNoObject {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return rc, nil
}

// ExpiringURL returns an expiring url from the unerlying store
func (s *mediaStorage) ExpiringURL(id string, exp time.Duration) (string, error) {
	return s.store.ExpiringURL(id, exp)
}
