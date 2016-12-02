package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &MockDAL{}

type MockDAL struct {
	*mock.Expector
}

// New returns an initialized instance of MockDAL
func New(t *testing.T) *MockDAL {
	return &MockDAL{&mock.Expector{T: t}}
}

func (dl *MockDAL) InsertMedia(model *dal.Media) (dal.MediaID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.MediaID(""), nil
	}
	return rets[0].(dal.MediaID), mock.SafeError(rets[1])
}

func (dl *MockDAL) Media(id dal.MediaID) (*dal.Media, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Media), mock.SafeError(rets[1])
}

func (dl *MockDAL) Medias(ids []dal.MediaID) ([]*dal.Media, error) {
	rets := dl.Record(ids)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Media), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateMedia(id dal.MediaID, update *dal.MediaUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteMedia(id dal.MediaID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) Transact(trans func(dal dal.DAL) error) (err error) {
	if err := trans(dl); err != nil {
		return errors.Trace(err)
	}
	return nil
}
