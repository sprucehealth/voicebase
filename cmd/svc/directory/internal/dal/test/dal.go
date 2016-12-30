package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &MockDAL{}

type MockDAL struct {
	*mock.Expector
}

// NewMockDAL returns an initialized instance of mockDAL
func NewMockDAL(t *testing.T) *MockDAL {
	return &MockDAL{&mock.Expector{T: t}}
}

func (dl *MockDAL) InsertEntity(model *dal.Entity) (dal.EntityID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EntityID{}, nil
	}
	return rets[0].(dal.EntityID), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertEntityContacts(models []*dal.EntityContact) error {
	rets := dl.Expector.Record(models)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) Entity(id dal.EntityID) (*dal.Entity, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Entity), mock.SafeError(rets[1])
}

func (dl *MockDAL) Entities(ids []dal.EntityID, statuses []dal.EntityStatus, types []dal.EntityType) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(ids, statuses, types)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Entity), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateEntity(id dal.EntityID, update *dal.EntityUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteEntity(id dal.EntityID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertExternalEntityID(model *dal.ExternalEntityID) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertExternalEntityIDs(models []*dal.ExternalEntityID) error {
	rets := dl.Expector.Record(models)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *MockDAL) ExternalEntityIDs(externalID string) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(externalID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
}

func (dl *MockDAL) ExternalEntityIDsForEntity(entityID dal.EntityID) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertEntityMembership(model *dal.EntityMembership) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityMemberships(id dal.EntityID) ([]*dal.EntityMembership, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityMembership), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityMembers(id dal.EntityID, statuses []dal.EntityStatus, types []dal.EntityType) ([]*dal.Entity, error) {
	rets := dl.Expector.Record(id, statuses, types)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.Entity), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertEntityContact(model *dal.EntityContact) (dal.EntityContactID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EntityContactID{}, nil
	}
	return rets[0].(dal.EntityContactID), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityContact(id dal.EntityContactID) (*dal.EntityContact, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityContacts(id dal.EntityID) ([]*dal.EntityContact, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityContactsForValue(value string) ([]*dal.EntityContact, error) {
	rets := dl.Expector.Record(value)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.EntityContact), mock.SafeError(rets[1])
}

func (dl *MockDAL) ExternalEntityIDsForEntities(entityID []dal.EntityID) ([]*dal.ExternalEntityID, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.ExternalEntityID), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateEntityContact(id dal.EntityContactID, update *dal.EntityContactUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteEntityContact(id dal.EntityContactID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteEntityContactsForEntityID(id dal.EntityID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertEvent(model *dal.Event) (dal.EventID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EventID{}, nil
	}
	return rets[0].(dal.EventID), mock.SafeError(rets[1])
}

func (dl *MockDAL) Event(id dal.EventID) (*dal.Event, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Event), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateEvent(id dal.EventID, update *dal.EventUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteEvent(id dal.EventID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityDomain(id *dal.EntityID, domain *string, opts ...dal.QueryOption) (dal.EntityID, string, error) {
	var rets []interface{}
	if len(opts) == 0 {
		rets = dl.Expector.Record(id, domain)
	} else {
		rets = dl.Expector.Record(id, domain, optsToInterfaces(opts))
	}
	if len(rets) == 0 {
		return dal.EmptyEntityID(), "", nil
	}

	return rets[0].(dal.EntityID), rets[1].(string), mock.SafeError(rets[2])
}

func (dl *MockDAL) UpsertEntityDomain(id dal.EntityID, domain string) error {
	rets := dl.Expector.Record(id, domain)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (dl *MockDAL) UpsertSerializedClientEntityContact(model *dal.SerializedClientEntityContact) error {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) SerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform) (*dal.SerializedClientEntityContact, error) {
	rets := dl.Expector.Record(entityID, platform)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.SerializedClientEntityContact), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateSerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform, update *dal.SerializedClientEntityContactUpdate) (int64, error) {
	rets := dl.Expector.Record(entityID, platform, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteSerializedClientEntityContact(entityID dal.EntityID, platform dal.SerializedClientEntityContactPlatform) (int64, error) {
	rets := dl.Expector.Record(entityID, platform)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityProfile(id dal.EntityProfileID) (*dal.EntityProfile, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.EntityProfile), mock.SafeError(rets[1])
}

func (dl *MockDAL) EntityProfileForEntity(id dal.EntityID) (*dal.EntityProfile, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.EntityProfile), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpsertEntityProfile(model *dal.EntityProfile) (dal.EntityProfileID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return dal.EmptyEntityProfileID(), nil
	}
	return rets[0].(dal.EntityProfileID), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteEntityProfile(id dal.EntityProfileID) (int64, error) {
	rets := dl.Expector.Record(id)
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

func (dl *MockDAL) InsertExternalLinkForEntity(entityID dal.EntityID, name, url string) error {
	rets := dl.Record(entityID, name, url)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) DeleteExternalLinkForEntity(entityID dal.EntityID, name string) error {
	rets := dl.Record(entityID, name)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) ExternalLinksForEntity(entityID dal.EntityID) ([]*dal.ExternalLink, error) {
	rets := dl.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*dal.ExternalLink), mock.SafeError(rets[1])
}

func (dl *MockDAL) SearchEntities(entitySearch *dal.EntitySearch, opts ...dal.QueryOption) ([]*dal.Entity, error) {
	rets := dl.Record(entitySearch)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*dal.Entity), mock.SafeError(rets[1])
}

func optsToInterfaces(opts []dal.QueryOption) []interface{} {
	ifs := make([]interface{}, len(opts))
	for i, o := range opts {
		ifs[i] = o
	}
	return ifs
}
