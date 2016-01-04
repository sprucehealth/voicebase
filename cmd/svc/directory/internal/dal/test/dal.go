package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDAL struct {
	*mock.Expector
}

// NewDAL returns an initialized instance of mockDAL. This returns the interface for a build time check that this mock always matches.
func NewDAL() dal.DAL {
	return &mockDAL{}
}

// NewMockDAL returns an initialized instance of mockDAL
func NewMockDAL(t *testing.T) *mockDAL {
	return &mockDAL{&mock.Expector{T: t}}
}

func (dl *mockDAL) InsertEntity(model *dal.Entity) (dal.EntityID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EntityID{}, nil
	}
	return rets[0].(dal.EntityID), mock.SafeError(rets[1])
}

func (dl *mockDAL) Entity(id dal.EntityID) (*dal.Entity, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Entity), mock.SafeError(rets[1])
}

func (dl *mockDAL) Entities(ids []dal.EntityID) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(ids)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Entity), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateEntity(id dal.EntityID, update *dal.EntityUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteEntity(id dal.EntityID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertExternalEntityID(model *dal.ExternalEntityID) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *mockDAL) ExternalEntityIDs(externalID string) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(externalID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
}

func (dl *mockDAL) ExternalEntityIDsForEntity(entityID dal.EntityID) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertEntityMembership(model *dal.EntityMembership) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *mockDAL) EntityMemberships(id dal.EntityID) ([]*dal.EntityMembership, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityMembership), mock.SafeError(rets[1])
}

func (dl *mockDAL) EntityMembers(id dal.EntityID) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Entity), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertEntityContact(model *dal.EntityContact) (dal.EntityContactID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EntityContactID{}, nil
	}
	return rets[0].(dal.EntityContactID), mock.SafeError(rets[1])
}

func (dl *mockDAL) EntityContact(id dal.EntityContactID) (*dal.EntityContact, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *mockDAL) EntityContacts(id dal.EntityID) ([]*dal.EntityContact, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *mockDAL) EntityContactsForValue(value string) ([]*dal.EntityContact, error) {
	rets := dl.Expector.Record(value)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateEntityContact(id dal.EntityContactID, update *dal.EntityContactUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteEntityContact(id dal.EntityContactID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertEvent(model *dal.Event) (dal.EventID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EventID{}, nil
	}
	return rets[0].(dal.EventID), mock.SafeError(rets[1])
}

func (dl *mockDAL) Event(id dal.EventID) (*dal.Event, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Event), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateEvent(id dal.EventID, update *dal.EventUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteEvent(id dal.EventID) (int64, error) {
	rets := dl.Expector.Record(id)
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
