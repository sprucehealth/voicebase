package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &mockDAL{}

type mockDAL struct {
	*mock.Expector
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

func (dl *mockDAL) InsertEntityContacts(models []*dal.EntityContact) error {
	rets := dl.Expector.Record(models)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) Entity(id dal.EntityID) (*dal.Entity, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Entity), mock.SafeError(rets[1])
}

func (dl *mockDAL) Entities(ids []dal.EntityID, statuses []dal.EntityStatus) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(ids, statuses)
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

func (dl *mockDAL) InsertExternalEntityIDs(models []*dal.ExternalEntityID) error {
	rets := dl.Expector.Record(models)
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

func (dl *mockDAL) EntityMembers(id dal.EntityID, statuses []dal.EntityStatus) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(id, statuses)
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

func (dl *mockDAL) ExternalEntityIDsForEntities(entityID []dal.EntityID) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
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

func (dl *mockDAL) DeleteEntityContactsForEntityID(id dal.EntityID) (int64, error) {
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

func (dl *mockDAL) EntityDomain(id *dal.EntityID, domain *string) (dal.EntityID, string, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return dal.EmptyEntityID(), "", nil
	}

	return rets[0].(dal.EntityID), rets[1].(string), mock.SafeError(rets[2])
}

func (dl *mockDAL) InsertEntityDomain(id dal.EntityID, domain string) error {
	rets := dl.Expector.Record(id, domain)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (dl *mockDAL) UpsertSerializedClientEntityContact(model *dal.SerializedClientEntityContact) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) SerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform) (*dal.SerializedClientEntityContact, error) {
	rets := dl.Expector.Record(entityID, platform)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.SerializedClientEntityContact), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateSerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform, update *dal.SerializedClientEntityContactUpdate) (int64, error) {
	rets := dl.Expector.Record(entityID, platform, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteSerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform) (int64, error) {
	rets := dl.Expector.Record(entityID, platform)
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
