package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &mockDAL{}

type mockDAL struct {
	*mock.Expector
}

// New returns an initialized instance of mockDAL
func New(t *testing.T) *mockDAL {
	return &mockDAL{&mock.Expector{T: t}}
}

func (dl *mockDAL) InsertMedia(model *dal.Media) (dal.MediaID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.MediaID(""), nil
	}
	return rets[0].(dal.MediaID), mock.SafeError(rets[1])
}

func (dl *mockDAL) Media(id dal.MediaID) (*dal.Media, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Media), mock.SafeError(rets[1])
}

func (dl *mockDAL) Medias(ids []dal.MediaID) ([]*dal.Media, error) {
	rets := dl.Record(ids)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Media), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateMedia(id dal.MediaID, update *dal.MediaUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteMedia(id dal.MediaID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) Transact(trans func(dal dal.DAL) error) (err error) {
	if err := trans(dl); err != nil {
		return errors.Trace(err)
	}
	return nil
}
