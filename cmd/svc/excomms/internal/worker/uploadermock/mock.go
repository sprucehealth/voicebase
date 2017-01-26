package uploadermock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type Uploader struct {
	*mock.Expector
}

func New(t *testing.T) *Uploader {
	return &Uploader{&mock.Expector{T: t}}
}

func (u *Uploader) Upload(contentType, url string) (*models.Media, error) {
	rets := u.Record(contentType, url)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.Media), mock.SafeError(rets[1])
}
