package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	mock_dal "github.com/sprucehealth/backend/cmd/svc/directory/internal/dal/test"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events/eventsmock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestLookupEntitiesByEntityID(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eSrc := &directory.EntitySource{Type: directory.EntitySource_PRACTICE_CODE, Data: "123456"}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
			Source:      directory.FlattenEntitySource(eSrc),
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key:                  &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
	test.Equals(t, eSrc, resp.Entities[0].Source)
}

func TestLookupEntitiesByDisplayName(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eSrc := &directory.EntitySource{Type: directory.EntitySource_PRACTICE_CODE, Data: "123456"}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.SearchEntities, &dal.EntitySearch{
		DisplayName: "DisplayName",
		Statuses:    ([]dal.EntityStatus)(nil),
		Types:       []dal.EntityType{},
	}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
			Source:      directory.FlattenEntitySource(eSrc),
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key:                  &directory.LookupEntitiesRequest_DisplayName{DisplayName: "DisplayName"},
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
	test.Equals(t, eSrc, resp.Entities[0].Source)
}

func TestLookupEntitiesByEntityIDNonZeroDepth(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	eID2, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeOrganization,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.EntityMembers, eID1, []dal.EntityStatus(nil), []dal.EntityType{}).WithReturns([]*dal.Entity{
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.EntityContacts, eID1).WithReturns([]*dal.EntityContact{
		{
			EntityID: eID1,
			Type:     dal.EntityContactTypePhone,
			Value:    "+1734846552",
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS}, resp.Entities[0].IncludedInformation)
	test.Equals(t, ([]directory.EntityInformation)(nil), resp.Entities[0].Members[0].IncludedInformation)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_ORGANIZATION, resp.Entities[0].Type)
}

func TestLookupEntitiesByBatchEntityID(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
		},
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key:                  &directory.LookupEntitiesRequest_BatchEntityID{BatchEntityID: &directory.IDList{IDs: []string{eID1.String(), eID2.String()}}},
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.Equals(t, 2, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
	test.Equals(t, eID2.String(), resp.Entities[1].ID)
	test.Equals(t, "entity2", resp.Entities[1].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[1].Type)
}

func TestLookupEntitiesByExternalID(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	externalID := "account:12345678"
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDs, externalID), []*dal.ExternalEntityID{
		{
			EntityID: eID1,
		},
		{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{ExternalID: externalID},
	})
	test.OK(t, err)

	test.Equals(t, 2, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
	test.Equals(t, eID2.String(), resp.Entities[1].ID)
	test.Equals(t, "entity2", resp.Entities[1].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[1].Type)
}

func TestLookupEntitiesByExternalID_MemberOfEntity(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	externalID := "account:12345678"
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	orgID1, err := dal.NewEntityID()
	test.OK(t, err)
	orgID2, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDs, externalID), []*dal.ExternalEntityID{
		{
			EntityID: eID1,
		},
		{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID1), []*dal.EntityMembership{
		{
			TargetEntityID: orgID1,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{orgID1}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          orgID1,
			DisplayName: orgID1.String(),
			Type:        dal.EntityTypeOrganization,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID2), []*dal.EntityMembership{
		{
			TargetEntityID: orgID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{orgID2}, ([]dal.EntityStatus)(nil), []dal.EntityType{}), []*dal.Entity{
		{
			ID:          orgID2,
			DisplayName: orgID2.String(),
			Type:        dal.EntityTypeOrganization,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key:            &directory.LookupEntitiesRequest_ExternalID{ExternalID: externalID},
		MemberOfEntity: orgID1.String(),
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
}

func TestLookupEntitiesNoResults(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	defer dl.Finish()
	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}, []dal.EntityStatus{dal.EntityStatusActive}, []dal.EntityType{}), []*dal.Entity{}, nil))
	_, err = s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key:      &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestLookupEntitiesByContact(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	contactValue := " 1234567@gmail.com "
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContactsForValue, strings.TrimSpace(contactValue)), []*dal.EntityContact{
		{
			EntityID: eID1,
		},
		{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}, []dal.EntityStatus{dal.EntityStatusActive, dal.EntityStatusDeleted}, []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	resp, err := s.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue:         contactValue,
		RequestedInformation: &directory.RequestedInformation{},
		Statuses: []directory.EntityStatus{
			directory.EntityStatus_ACTIVE,
			directory.EntityStatus_DELETED,
		},
	})
	test.OK(t, err)

	test.Equals(t, 2, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
	test.Equals(t, eID2.String(), resp.Entities[1].ID)
	test.Equals(t, "entity2", resp.Entities[1].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[1].Type)
	mock.FinishAll(dl)
}

func TestLookupEntitiesByContact_MemberOfEntity(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	contactValue := " 1234567@gmail.com "
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	orgID1, err := dal.NewEntityID()
	test.OK(t, err)
	orgID2, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContactsForValue, strings.TrimSpace(contactValue)), []*dal.EntityContact{
		{
			EntityID: eID1,
		},
		{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}, []dal.EntityStatus{dal.EntityStatusActive, dal.EntityStatusDeleted}, []dal.EntityType{}), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID1), []*dal.EntityMembership{
		{
			TargetEntityID: orgID1,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{orgID1}, []dal.EntityStatus{dal.EntityStatusActive, dal.EntityStatusDeleted}, []dal.EntityType{}), []*dal.Entity{
		{
			ID:          orgID1,
			DisplayName: orgID1.String(),
			Type:        dal.EntityTypeOrganization,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID2), []*dal.EntityMembership{
		{
			TargetEntityID: orgID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{orgID2}, []dal.EntityStatus{dal.EntityStatusActive, dal.EntityStatusDeleted}, []dal.EntityType{}), []*dal.Entity{
		{
			ID:          orgID2,
			DisplayName: orgID2.String(),
			Type:        dal.EntityTypeOrganization,
			Status:      dal.EntityStatusActive,
		},
	}, nil))

	resp, err := s.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue:         contactValue,
		RequestedInformation: &directory.RequestedInformation{},
		Statuses: []directory.EntityStatus{
			directory.EntityStatus_ACTIVE,
			directory.EntityStatus_DELETED,
		},
		MemberOfEntity: orgID1.String(),
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
}

func TestLookupEntitiesByContactNoResults(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	contactValue := " 1234567@gmail.com "
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContactsForValue, strings.TrimSpace(contactValue)), []*dal.EntityContact{}, nil))
	_, err := s.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue:         contactValue,
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateEntityFull(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+1234567890",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "bat@cave.com",
			Provisioned: true,
		},
	}
	male := dal.EntityGenderMale
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), &dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.InsertEntity, &dal.Entity{
		DisplayName: name,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
		DOB:         &encoding.Date{Month: 7, Day: 25, Year: 1986},
		Gender:      &male,
	}), eID1, nil))
	dl.Expect(mock.NewExpectation(dl.InsertExternalEntityID, &dal.ExternalEntityID{
		EntityID:   eID1,
		ExternalID: externalID,
	}))
	dl.Expect(mock.NewExpectation(dl.InsertEntityMembership, &dal.EntityMembership{
		EntityID:       eID1,
		TargetEntityID: eID2,
		Status:         dal.EntityMembershipStatusActive,
	}))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContact, &dal.EntityContact{
		EntityID:    eID1,
		Type:        dal.EntityContactTypePhone,
		Value:       "+1234567890",
		Provisioned: false,
	}))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContact, &dal.EntityContact{
		EntityID:    eID1,
		Type:        dal.EntityContactTypeEmail,
		Value:       "bat@cave.com",
		Provisioned: true,
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: name,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
			DOB: &directory.Date{
				Year:  1986,
				Month: 7,
				Day:   25,
			},
			Gender: directory.EntityInfo_MALE,
		},
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, eID1.String(), resp.Entity.ID)
	test.Equals(t, name, resp.Entity.Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entity.Type)
}

func TestCreateEntityInitialEntityNotFound(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12345678910",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "bat@cave.com",
			Provisioned: true,
		},
	}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), (*dal.Entity)(nil), dal.ErrNotFound))
	_, err = s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
		},
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateEntityEmptyContact(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "",
		},
	}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), &dal.Entity{}, nil))
	_, err = s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
		},
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
}

func TestCreateEntityInvalidEmail(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "notavalidemail",
		},
	}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), &dal.Entity{}, nil))
	_, err = s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
		},
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
}

func TestCreateEntitySparse(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	firstName := "Batman"
	eType := directory.EntityType_INTERNAL
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.InsertEntity, &dal.Entity{
		DisplayName: name,
		FirstName:   firstName,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}), eID1, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: name,
		FirstName:   firstName,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
			FirstName:   firstName,
		},
		Type: eType,
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, eID1.String(), resp.Entity.ID)
	test.Equals(t, name, resp.Entity.Info.DisplayName)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entity.Type)
}

func TestCreateMembership(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), &dal.Entity{}, nil))
	dl.Expect(mock.NewExpectation(dl.InsertEntityMembership, &dal.EntityMembership{
		EntityID:       eID1,
		TargetEntityID: eID2,
		Status:         dal.EntityMembershipStatusActive,
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "newmember",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))
	resp, err := s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "newmember", resp.Entity.Info.DisplayName)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestCreateMembershipEntityNotFound(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), (*dal.Entity)(nil), dal.ErrNotFound))
	_, err = s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateMembershipTargetEntityNotFound(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), (*dal.Entity)(nil), dal.ErrNotFound))
	_, err = s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateContact(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContact, &dal.EntityContact{
		EntityID: eID1,
		Type:     dal.EntityContactTypePhone,
		Value:    "+12345678910",
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))
	resp, err := s.CreateContact(context.Background(), &directory.CreateContactRequest{
		EntityID: eID1.String(),
		Contact: &directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12345678910",
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestCreateContactEntityNotFound(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), (*dal.Entity)(nil), dal.ErrNotFound))
	_, err = s.CreateContact(context.Background(), &directory.CreateContactRequest{
		EntityID: eID1.String(),
		Contact: &directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12345678910",
		},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCreateContactInvalidEmail(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	_, err = s.CreateContact(context.Background(), &directory.CreateContactRequest{
		EntityID: eID1.String(),
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "notavalidemail",
		},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCreateEntityDomain(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.UpsertEntityDomain, eID1, "domain"))
	_, err = s.CreateEntityDomain(context.Background(), &directory.CreateEntityDomainRequest{
		EntityID: eID1.String(),
		Domain:   "domain",
	})
	test.OK(t, err)
	mock.FinishAll(dl)
}

func TestUpdateEntityDomain(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.EntityDomain, &eID1, (*string)(nil), []interface{}{dal.ForUpdate}).WithReturns(eID1, "oldDomain", nil))
	dl.Expect(mock.NewExpectation(dl.EntityDomain, (*dal.EntityID)(nil), ptr.String("newDomain")).WithReturns(eID1, "", errors.Trace(dal.ErrNotFound)))

	dl.Expect(mock.NewExpectation(dl.UpsertEntityDomain, eID1, "newdomain"))
	_, err = s.UpdateEntityDomain(context.Background(), &directory.UpdateEntityDomainRequest{
		EntityID: eID1.String(),
		Domain:   "newDomain",
	})
	test.OK(t, err)
	mock.FinishAll(dl)
}

func TestLookupEntityDomain(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.EntityDomain, &eID1, ptr.String("")).WithReturns(eID1, "hello", nil))
	res, err := s.LookupEntityDomain(context.Background(), &directory.LookupEntityDomainRequest{
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.Equals(t, eID1.String(), res.EntityID)
	test.Equals(t, "hello", res.Domain)
	mock.FinishAll(dl)
}

func TestLookupEntitiesAdditionalInformationGraphCrawl(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	eID3, err := dal.NewEntityID()
	test.OK(t, err)
	statuses := []dal.EntityStatus{dal.EntityStatusActive}
	rootTypes := []dal.EntityType{dal.EntityTypeOrganization}
	childTypes := []dal.EntityType{dal.EntityTypeInternal}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}, statuses, rootTypes), []*dal.Entity{
		{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID1), []*dal.EntityMembership{
		{
			TargetEntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID2}, statuses, childTypes), []*dal.Entity{
		{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeExternal,
			Status:      dal.EntityStatusActive,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID2), []*dal.EntityMembership{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{}, statuses, childTypes), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID2, statuses, childTypes), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID2}), []*dal.ExternalEntityID{
		{ExternalID: "external2"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID2), []*dal.EntityContact{
		{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678912",
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID1, statuses, childTypes), []*dal.Entity{
		{
			ID:     eID3,
			Type:   dal.EntityTypeInternal,
			Status: dal.EntityStatusActive,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID3), []*dal.EntityMembership{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{}, statuses, childTypes), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID3, statuses, childTypes), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID3}), []*dal.ExternalEntityID{
		{ExternalID: "external3"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID3), []*dal.EntityContact{
		{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678913",
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID1}), []*dal.ExternalEntityID{
		{ExternalID: "external1"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID1), []*dal.EntityContact{
		{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678911",
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 2,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
		ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
	test.Equals(t, 1, len(resp.Entities[0].Contacts))
	test.Equals(t, "+12345678911", resp.Entities[0].Contacts[0].Value)
	test.Equals(t, 1, len(resp.Entities[0].ExternalIDs))
	test.Equals(t, "external1", resp.Entities[0].ExternalIDs[0])
	test.Equals(t, 1, len(resp.Entities[0].Memberships))
	test.Equals(t, eID2.String(), resp.Entities[0].Memberships[0].ID)
	test.Equals(t, 1, len(resp.Entities[0].Memberships[0].Contacts))
	test.Equals(t, "+12345678912", resp.Entities[0].Memberships[0].Contacts[0].Value)
	test.Equals(t, 1, len(resp.Entities[0].Memberships[0].ExternalIDs))
	test.Equals(t, "external2", resp.Entities[0].Memberships[0].ExternalIDs[0])
	test.Equals(t, eID3.String(), resp.Entities[0].Members[0].ID)
	test.Equals(t, 1, len(resp.Entities[0].Members[0].Contacts))
	test.Equals(t, "+12345678913", resp.Entities[0].Members[0].Contacts[0].Value)
	test.Equals(t, 1, len(resp.Entities[0].Members[0].ExternalIDs))
	test.Equals(t, "external3", resp.Entities[0].Members[0].ExternalIDs[0])
	mock.FinishAll(dl)
}

func TestCreateContacts(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{
		{
			EntityID: eID1,
			Type:     dal.EntityContactTypePhone,
			Value:    "+12345678910",
		},
		{
			EntityID: eID1,
			Type:     dal.EntityContactTypeEmail,
			Value:    "test@email.com",
		},
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))
	resp, err := s.CreateContacts(context.Background(), &directory.CreateContactsRequest{
		EntityID: eID1.String(),
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12345678910",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "test@email.com",
			},
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestUpdateContacts(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eCID1, err := dal.NewEntityContactID()
	test.OK(t, err)
	eCID2, err := dal.NewEntityContactID()
	test.OK(t, err)
	phoneType := dal.EntityContactTypePhone
	emailType := dal.EntityContactTypeEmail
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type:   dal.EntityTypeInternal,
		Status: dal.EntityStatusActive,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateEntityContact, eCID1, &dal.EntityContactUpdate{
		Type:     &phoneType,
		Value:    ptr.String("+12345678910"),
		Label:    ptr.String(""),
		Verified: ptr.Bool(false),
	}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.UpdateEntityContact, eCID2, &dal.EntityContactUpdate{
		Type:     &emailType,
		Value:    ptr.String("test@email.com"),
		Label:    ptr.String("NewLabel"),
		Verified: ptr.Bool(false),
	}).WithReturns(int64(1), nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))

	resp, err := s.UpdateContacts(context.Background(), &directory.UpdateContactsRequest{
		EntityID: eID1.String(),
		Contacts: []*directory.Contact{
			{
				ID:          eCID1.String(),
				ContactType: directory.ContactType_PHONE,
				Value:       "+12345678910",
			},
			{
				ID:          eCID2.String(),
				ContactType: directory.ContactType_EMAIL,
				Value:       "test@email.com",
				Label:       "NewLabel",
			},
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestDeleteContacts(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eCID1, err := dal.NewEntityContactID()
	test.OK(t, err)
	eCID2, err := dal.NewEntityContactID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContact, eCID1))
	dl.Expect(mock.NewExpectation(dl.DeleteEntityContact, eCID2))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))

	resp, err := s.DeleteContacts(context.Background(), &directory.DeleteContactsRequest{
		EntityID:         eID1.String(),
		EntityContactIDs: []string{eCID1.String(), eCID2.String()},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestUpdateEntity(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type: dal.EntityTypeInternal,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContactsForEntityID, eID1))

	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{
		FirstName:     ptr.String(""),
		LastName:      ptr.String(""),
		MiddleInitial: ptr.String(""),
		GroupName:     ptr.String(""),
		ShortTitle:    ptr.String(""),
		LongTitle:     ptr.String(""),
		AccountID:     ptr.String("account_id"),
		Note:          ptr.String("I am the knight"),
	}))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Note:        "I am the knight",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))

	gomock.InOrder(
		publisher.EXPECT().PublishAsync(&directory.EntityUpdatedEvent{
			EntityID: eID1.String(),
		}),
	)

	resp, err := s.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
		EntityID:         eID1.String(),
		UpdateEntityInfo: true,
		EntityInfo: &directory.EntityInfo{
			DisplayName: "batman",
			Note:        "I am the knight",
		},
		UpdateAccountID: true,
		AccountID:       "account_id",
		UpdateContacts:  true,
		Contacts:        nil,
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, "I am the knight", resp.Entity.Info.Note)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestUpdateEntityWithContacts(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type: dal.EntityTypeInternal,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContactsForEntityID, eID1))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{
		{
			EntityID:    eID1,
			Value:       "1",
			Provisioned: true,
			Label:       "Label1",
			Type:        dal.EntityContactTypeEmail,
		},
		{
			EntityID:    eID1,
			Value:       "2",
			Provisioned: false,
			Label:       "Label2",
			Type:        dal.EntityContactTypePhone,
		},
	}))

	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{
		DisplayName:   ptr.String("1"),
		FirstName:     ptr.String(""),
		LastName:      ptr.String(""),
		MiddleInitial: ptr.String(""),
		GroupName:     ptr.String(""),
		ShortTitle:    ptr.String(""),
		LongTitle:     ptr.String(""),
		AccountID:     nil,
		Note:          ptr.String("I am the knight"),
	}))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "1",
		Note:        "I am the knight",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))

	gomock.InOrder(
		publisher.EXPECT().PublishAsync(&directory.EntityUpdatedEvent{
			EntityID: eID1.String(),
		}),
	)

	resp, err := s.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
		EntityID:         eID1.String(),
		UpdateEntityInfo: true,
		EntityInfo: &directory.EntityInfo{
			DisplayName: "batman",
			Note:        "I am the knight",
		},
		UpdateContacts: true,
		Contacts: []*directory.Contact{
			{
				Value:       "1",
				Provisioned: true,
				Label:       "Label1",
				ContactType: directory.ContactType_EMAIL,
			},
			{
				Value:       "2",
				Provisioned: false,
				Label:       "Label2",
				ContactType: directory.ContactType_PHONE,
			},
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "1", resp.Entity.Info.DisplayName)
	test.Equals(t, "I am the knight", resp.Entity.Info.Note)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestUpdateEntityWithSerializedContacts(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type: dal.EntityTypeInternal,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContactsForEntityID, eID1))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{
		{
			EntityID:    eID1,
			Value:       "1",
			Provisioned: true,
			Label:       "Label1",
			Type:        dal.EntityContactTypeEmail,
		},
		{
			EntityID:    eID1,
			Value:       "2",
			Provisioned: false,
			Label:       "Label2",
			Type:        dal.EntityContactTypePhone,
		},
	}))

	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{
		DisplayName:   ptr.String("1"),
		FirstName:     ptr.String(""),
		LastName:      ptr.String(""),
		MiddleInitial: ptr.String(""),
		GroupName:     ptr.String(""),
		ShortTitle:    ptr.String(""),
		LongTitle:     ptr.String(""),
		AccountID:     ptr.String("abc"),
		Note:          ptr.String("I am the knight"),
	}))

	dl.Expect(mock.NewExpectation(dl.UpsertSerializedClientEntityContact,
		&dal.SerializedClientEntityContact{
			EntityID:                eID1,
			Platform:                dal.SerializedClientEntityContactPlatformIOS,
			SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
		}))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "1",
		Note:        "I am the knight",
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}, nil))

	gomock.InOrder(
		publisher.EXPECT().PublishAsync(&directory.EntityUpdatedEvent{
			EntityID: eID1.String(),
		}),
	)

	resp, err := s.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
		EntityID:         eID1.String(),
		UpdateEntityInfo: true,
		EntityInfo: &directory.EntityInfo{
			DisplayName: "batman",
			Note:        "I am the knight",
		},
		UpdateContacts: true,
		Contacts: []*directory.Contact{
			{
				Value:       "1",
				Provisioned: true,
				Label:       "Label1",
				ContactType: directory.ContactType_EMAIL,
			},
			{
				Value:       "2",
				Provisioned: false,
				Label:       "Label2",
				ContactType: directory.ContactType_PHONE,
			},
		},
		UpdateSerializedEntityContacts: true,
		SerializedEntityContacts: []*directory.SerializedClientEntityContact{
			{
				EntityID:                eID1.String(),
				Platform:                directory.Platform_IOS,
				SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
			},
		},
		UpdateAccountID: true,
		AccountID:       "abc",
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "1", resp.Entity.Info.DisplayName)
	test.Equals(t, "I am the knight", resp.Entity.Info.Note)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestSerializedEntityContact(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	platform := dal.SerializedClientEntityContactPlatformIOS

	dl.Expect(mock.NewExpectation(dl.SerializedClientEntityContact, eID1, platform).WithReturns(&dal.SerializedClientEntityContact{
		EntityID:                eID1,
		SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
		Platform:                platform,
	}, nil))

	resp, err := s.SerializedEntityContact(context.Background(), &directory.SerializedEntityContactRequest{
		EntityID: eID1.String(),
		Platform: directory.Platform_IOS,
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp)
	test.Equals(t, &directory.SerializedClientEntityContact{
		EntityID:                eID1.String(),
		Platform:                directory.Platform_IOS,
		SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
	}, resp.SerializedEntityContact)
}

func TestCreateEHRLink(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	name := "drchrono"
	url := "https://drchrono.com"

	dl.Expect(mock.NewExpectation(dl.InsertExternalLinkForEntity, eID1, name, url))

	resp, err := s.CreateExternalLink(context.Background(), &directory.CreateExternalLinkRequest{
		Name:     "drchrono",
		URL:      "https://drchrono.com",
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp)
}

func TestDeleteEHRLink(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	name := "drchrono"

	dl.Expect(mock.NewExpectation(dl.DeleteExternalLinkForEntity, eID1, name))

	resp, err := s.DeleteExternalLink(context.Background(), &directory.DeleteExternalLinkRequest{
		Name:     "drchrono",
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp)
}

func TestEHRLinksForEntity(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	name := "drchrono"
	url := "https://drchrono.com"

	dl.Expect(mock.NewExpectation(dl.ExternalLinksForEntity, eID1).WithReturns([]*dal.ExternalLink{
		{
			Name: name,
			URL:  url,
		},
	}, nil))

	resp, err := s.LookupExternalLinksForEntity(context.Background(), &directory.LookupExternalLinksForEntityRequest{
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp)
	test.Equals(t, &directory.LookupExternalLinksforEntityResponse{
		Links: []*directory.LookupExternalLinksforEntityResponse_ExternalLink{
			{
				Name: name,
				URL:  url,
			},
		},
	}, resp)
}

func TestSerializedEntityContactNotFound(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	platform := dal.SerializedClientEntityContactPlatformIOS

	dl.Expect(mock.NewExpectation(dl.SerializedClientEntityContact, eID1, platform).WithReturns((*dal.SerializedClientEntityContact)(nil), dal.ErrNotFound))

	resp, err := s.SerializedEntityContact(context.Background(), &directory.SerializedEntityContactRequest{
		EntityID: eID1.String(),
		Platform: directory.Platform_IOS,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestDeleteEntity(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	deleted := dal.EntityStatusDeleted
	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{Status: &deleted}).WithReturns(int64(1), nil))

	resp, err := s.DeleteEntity(context.Background(), &directory.DeleteEntityRequest{
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp)
}

func TestProfile(t *testing.T) {
	entityID, err := dal.NewEntityID()
	test.OK(t, err)
	profileID, err := dal.NewEntityProfileID()
	test.OK(t, err)
	clk := clock.NewManaged(time.Now())
	cases := map[string]struct {
		request          *directory.ProfileRequest
		dal              *mock_dal.MockDAL
		expectedResponse *directory.ProfileResponse
		expectedError    error
	}{
		"LookupEntityID-BadID": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_EntityID{
					EntityID: "notAnEntityID",
				},
			},
			dal:              nil,
			expectedResponse: nil,
			expectedError: func() error {
				_, err := dal.ParseEntityID("notAnEntityID")
				return err
			}(),
		},
		"LookupEntityID-NotFound": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_EntityID{
					EntityID: entityID.String(),
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, entityID).WithReturns((*dal.EntityProfile)(nil), dal.ErrNotFound))
				return mDAL
			}(),
			expectedResponse: nil,
			expectedError:    grpc.Errorf(codes.NotFound, "Profile for entity id %s not found", entityID),
		},
		"LookupEntityID-Found": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_EntityID{
					EntityID: entityID.String(),
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, entityID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.ProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
			},
			expectedError: nil,
		},
		"LookupProfileID-BadID": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_ProfileID{
					ProfileID: "notAProfileID",
				},
			},
			dal:              nil,
			expectedResponse: nil,
			expectedError: func() error {
				_, err := dal.ParseEntityProfileID("notAProfileID")
				return err
			}(),
		},
		"LookupProfileID-NotFound": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_ProfileID{
					ProfileID: profileID.String(),
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, profileID).WithReturns((*dal.EntityProfile)(nil), dal.ErrNotFound))
				return mDAL
			}(),
			expectedResponse: nil,
			expectedError:    grpc.Errorf(codes.NotFound, "Profile for profile id %s not found", profileID),
		},
		"LookupProfileID-Found": {
			request: &directory.ProfileRequest{
				Key: &directory.ProfileRequest_ProfileID{
					ProfileID: profileID.String(),
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.ProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
			},
			expectedError: nil,
		},
	}

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	for cn, c := range cases {
		svr := New(c.dal, publisher, metrics.NewRegistry())
		resp, err := svr.Profile(context.Background(), c.request)
		test.EqualsCase(t, cn, c.expectedResponse, resp)
		test.EqualsCase(t, cn, errors.Cause(c.expectedError), errors.Cause(err))
		if c.dal != nil {
			c.dal.Finish()
		}
	}
}

func TestUpdateProfile(t *testing.T) {
	entityID, err := dal.NewEntityID()
	test.OK(t, err)
	newEntityID, err := dal.NewEntityID()
	test.OK(t, err)
	profileID, err := dal.NewEntityProfileID()
	test.OK(t, err)
	clk := clock.NewManaged(time.Now())
	entR, err := transformEntityToResponse(&dal.Entity{
		ID:       entityID,
		Created:  clk.Now(),
		Modified: clk.Now(),
		Type:     dal.EntityTypeInternal,
		Status:   dal.EntityStatusActive,
	})
	test.OK(t, err)
	cases := map[string]struct {
		request          *directory.UpdateProfileRequest
		dal              *mock_dal.MockDAL
		expectedResponse *directory.UpdateProfileResponse
		expectedError    error
	}{
		"ProfileID-NoProfile": {
			request:          &directory.UpdateProfileRequest{},
			dal:              nil,
			expectedResponse: nil,
			expectedError:    grpc.Errorf(codes.InvalidArgument, "Profile required"),
		},
		"ProfileID-BadID": {
			request: &directory.UpdateProfileRequest{
				ProfileID: "notAProfileID",
				Profile:   &directory.Profile{},
			},
			dal:              nil,
			expectedResponse: nil,
			expectedError: func() error {
				_, err := dal.ParseEntityProfileID("notAProfileID")
				return grpc.Errorf(codes.InvalidArgument, err.Error())
			}(),
		},
		"EntityID-BadID": {
			request: &directory.UpdateProfileRequest{
				Profile: &directory.Profile{
					EntityID: "notAnEntityID",
				},
			},
			dal:              nil,
			expectedResponse: nil,
			expectedError: func() error {
				_, err := dal.ParseEntityID("notAnEntityID")
				return grpc.Errorf(codes.InvalidArgument, err.Error())
			}(),
		},
		"ProfileID-NotFound": {
			request: &directory.UpdateProfileRequest{
				ProfileID: profileID.String(),
				Profile:   &directory.Profile{},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns((*dal.EntityProfile)(nil), dal.ErrNotFound))
				return mDAL
			}(),
			expectedResponse: nil,
			expectedError:    grpc.Errorf(codes.NotFound, "Profile id %s not found", profileID),
		},
		"ProfileID-ChangeOwningEntity": {
			request: &directory.UpdateProfileRequest{
				ProfileID: profileID.String(),
				Profile: &directory.Profile{
					EntityID: newEntityID.String(),
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					EntityID: entityID,
				}, nil))
				return mDAL
			}(),
			expectedResponse: nil,
			expectedError:    grpc.Errorf(codes.PermissionDenied, "The owning entity of a profile cannot be changed - Is: %s, Request Provided: %s", entityID, newEntityID),
		},
		"ProfileID-Success": {
			request: &directory.UpdateProfileRequest{
				ProfileID: profileID.String(),
				Profile: &directory.Profile{
					DisplayName: "New display name",
					Sections:    []*directory.ProfileSection{},
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					EntityID: entityID,
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpsertEntityProfile, &dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{Sections: []*directory.ProfileSection{}},
				}).WithReturns(profileID, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpdateEntity, entityID, &dal.EntityUpdate{
					CustomDisplayName: ptr.String("New display name"),
					FirstName:         ptr.String(""),
					LastName:          ptr.String(""),
					HasProfile:        ptr.Bool(true),
				}))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.Entity, entityID).WithReturns(&dal.Entity{
					ID:       entityID,
					Created:  clk.Now(),
					Modified: clk.Now(),
					Type:     dal.EntityTypeInternal,
					Status:   dal.EntityStatusActive,
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.UpdateProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
				Entity: entR,
			},
			expectedError: nil,
		},
		"EntityID-NewSuccess": {
			request: &directory.UpdateProfileRequest{
				Profile: &directory.Profile{
					DisplayName: "New display name",
					FirstName:   "FirstName",
					LastName:    "LastName",
					EntityID:    entityID.String(),
					Sections:    []*directory.ProfileSection{},
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, entityID).WithReturns((*dal.EntityProfile)(nil), dal.ErrNotFound))
				mDAL.Expect(mock.NewExpectation(mDAL.UpsertEntityProfile, &dal.EntityProfile{
					ID:       dal.EmptyEntityProfileID(),
					EntityID: entityID,
					Sections: &directory.ProfileSections{Sections: []*directory.ProfileSection{}},
				}).WithReturns(profileID, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpdateEntity, entityID, &dal.EntityUpdate{
					CustomDisplayName: ptr.String("New display name"),
					FirstName:         ptr.String("FirstName"),
					LastName:          ptr.String("LastName"),
					HasProfile:        ptr.Bool(true),
				}))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.Entity, entityID).WithReturns(&dal.Entity{
					ID:       entityID,
					Created:  clk.Now(),
					Modified: clk.Now(),
					Type:     dal.EntityTypeInternal,
					Status:   dal.EntityStatusActive,
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.UpdateProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
				Entity: entR,
			},
			expectedError: nil,
		},
		"EntityID-ExistingSuccess": {
			request: &directory.UpdateProfileRequest{
				Profile: &directory.Profile{
					DisplayName: "New display name",
					EntityID:    entityID.String(),
					Sections:    []*directory.ProfileSection{},
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, entityID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					EntityID: entityID,
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpsertEntityProfile, &dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{Sections: []*directory.ProfileSection{}},
				}).WithReturns(profileID, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpdateEntity, entityID, &dal.EntityUpdate{
					CustomDisplayName: ptr.String("New display name"),
					HasProfile:        ptr.Bool(true),
					FirstName:         ptr.String(""),
					LastName:          ptr.String(""),
				}))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.Entity, entityID).WithReturns(&dal.Entity{
					ID:       entityID,
					Created:  clk.Now(),
					Modified: clk.Now(),
					Type:     dal.EntityTypeInternal,
					Status:   dal.EntityStatusActive,
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.UpdateProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
				Entity: entR,
			},
			expectedError: nil,
		},
		"EntityID-ImageMediaID": {
			request: &directory.UpdateProfileRequest{
				ImageMediaID: "imageMediaID",
				Profile: &directory.Profile{
					DisplayName: "New display name",
					EntityID:    entityID.String(),
					Sections:    []*directory.ProfileSection{},
				},
			},
			dal: func() *mock_dal.MockDAL {
				mDAL := mock_dal.NewMockDAL(t)
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfileForEntity, entityID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					EntityID: entityID,
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpsertEntityProfile, &dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{Sections: []*directory.ProfileSection{}},
				}).WithReturns(profileID, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.UpdateEntity, entityID, &dal.EntityUpdate{
					CustomDisplayName: ptr.String("New display name"),
					HasProfile:        ptr.Bool(true),
					FirstName:         ptr.String(""),
					LastName:          ptr.String(""),
					ImageMediaID:      ptr.String("imageMediaID"),
				}))
				mDAL.Expect(mock.NewExpectation(mDAL.EntityProfile, profileID).WithReturns(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}, nil))
				mDAL.Expect(mock.NewExpectation(mDAL.Entity, entityID).WithReturns(&dal.Entity{
					ID:       entityID,
					Created:  clk.Now(),
					Modified: clk.Now(),
					Type:     dal.EntityTypeInternal,
					Status:   dal.EntityStatusActive,
				}, nil))
				return mDAL
			}(),
			expectedResponse: &directory.UpdateProfileResponse{
				Profile: transformEntityProfileToResponse(&dal.EntityProfile{
					ID:       profileID,
					EntityID: entityID,
					Sections: &directory.ProfileSections{},
					Modified: clk.Now(),
				}),
				Entity: entR,
			},
			expectedError: nil,
		},
	}

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	for cn, c := range cases {
		svr := New(c.dal, publisher, metrics.NewRegistry())
		resp, err := svr.UpdateProfile(context.Background(), c.request)
		test.EqualsCase(t, cn, c.expectedResponse, resp)
		test.EqualsCase(t, cn, c.expectedError, err)
		if c.dal != nil {
			c.dal.Finish()
		}
	}
}

func TestContact(t *testing.T) {
	t.Parallel()
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	ctrl := gomock.NewController(t)
	publisher := eventsmock.NewMockPublisher(ctrl)
	defer ctrl.Finish()

	s := New(dl, publisher, metrics.NewRegistry())
	ecID1, err := dal.NewEntityContactID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContact, ecID1), &dal.EntityContact{ID: ecID1}, nil))

	resp, err := s.Contact(context.Background(), &directory.ContactRequest{
		ContactID: ecID1.String(),
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Contact)
	test.Equals(t, transformEntityContactToResponse(&dal.EntityContact{ID: ecID1}), resp.Contact)
}
