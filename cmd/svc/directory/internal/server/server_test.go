package server

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	mock_dal "github.com/sprucehealth/backend/cmd/svc/directory/internal/dal/test"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestLookupEntitiesByEntityID(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}), []*dal.Entity{
		&dal.Entity{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType:        directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof:       &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.Equals(t, 1, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Info.DisplayName)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
}

func TestLookupEntitiesByExternalID(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	externalID := "account:12345678"
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDs, externalID), []*dal.ExternalEntityID{
		&dal.ExternalEntityID{
			EntityID: eID1,
		},
		&dal.ExternalEntityID{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}), []*dal.Entity{
		&dal.Entity{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
		},
		&dal.Entity{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType:  directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{ExternalID: externalID},
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

func TestLookupEntitiesNoResults(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}), []*dal.Entity{}, nil))
	_, err = s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType:  directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestLookupEntitiesByContact(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	contactValue := " 1234567@gmail.com "
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContactsForValue, strings.TrimSpace(contactValue)), []*dal.EntityContact{
		&dal.EntityContact{
			EntityID: eID1,
		},
		&dal.EntityContact{
			EntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1, eID2}), []*dal.Entity{
		&dal.Entity{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeInternal,
		},
		&dal.Entity{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeInternal,
		},
	}, nil))
	resp, err := s.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue:         contactValue,
		RequestedInformation: &directory.RequestedInformation{},
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

func TestLookupEntitiesByContactNoResults(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		&directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "batphone", // This should break when phone validation is enabled
		},
		&directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "bat@cave.com", // This should break when phone validation is enabled
			Provisioned: true,
		},
	}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), &dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.InsertEntity, &dal.Entity{
		DisplayName: name,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
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
		Value:       "batphone",
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
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		&directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12345678910",
		},
		&directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "bat@cave.com",
			Provisioned: true,
		},
	}
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), (*dal.Entity)(nil), api.ErrNotFound("not found")))
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		&directory.Contact{
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	externalID := "brucewayne"
	contacts := []*directory.Contact{
		&directory.Contact{
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.InsertEntity, &dal.Entity{
		DisplayName: name,
		Type:        dal.EntityTypeInternal,
		Status:      dal.EntityStatusActive,
	}), eID1, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: name,
		Type:        dal.EntityTypeInternal,
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), (*dal.Entity)(nil), api.ErrNotFound("not found")))
	_, err = s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateMembershipTargetEntityNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID2), (*dal.Entity)(nil), api.ErrNotFound("not found")))
	_, err = s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCreateContact(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
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
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), (*dal.Entity)(nil), api.ErrNotFound("not found")))
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
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
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.InsertEntityDomain, eID1, "domain"))
	_, err = s.CreateEntityDomain(context.Background(), &directory.CreateEntityDomainRequest{
		EntityID: eID1.String(),
		Domain:   "domain",
	})
	test.OK(t, err)
	mock.FinishAll(dl)
}

func TestLookupEntityDomain(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.EntityDomain, &eID1).WithReturns(eID1, "hello", nil))
	res, err := s.LookupEntityDomain(context.Background(), &directory.LookupEntityDomainRequest{
		EntityID: eID1.String(),
	})
	test.OK(t, err)
	test.Equals(t, eID1.String(), res.EntityID)
	test.Equals(t, "hello", res.Domain)
	mock.FinishAll(dl)
}

func TestLookupEntitiesAdditionalInformationGraphCrawl(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	eID3, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}), []*dal.Entity{
		&dal.Entity{
			ID:          eID1,
			DisplayName: "entity1",
			Type:        dal.EntityTypeExternal,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID1), []*dal.EntityMembership{
		&dal.EntityMembership{
			TargetEntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID2}), []*dal.Entity{
		&dal.Entity{
			ID:          eID2,
			DisplayName: "entity2",
			Type:        dal.EntityTypeExternal,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID2), []*dal.EntityMembership{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{}), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID2), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID2}), []*dal.ExternalEntityID{
		&dal.ExternalEntityID{ExternalID: "external2"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID2), []*dal.EntityContact{
		&dal.EntityContact{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678912",
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID1), []*dal.Entity{
		&dal.Entity{
			ID:   eID3,
			Type: dal.EntityTypeInternal,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID3), []*dal.EntityMembership{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{}), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMembers, eID3), []*dal.Entity{}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID3}), []*dal.ExternalEntityID{
		&dal.ExternalEntityID{ExternalID: "external3"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID3), []*dal.EntityContact{
		&dal.EntityContact{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678913",
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.ExternalEntityIDsForEntities, []dal.EntityID{eID1}), []*dal.ExternalEntityID{
		&dal.ExternalEntityID{ExternalID: "external1"},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityContacts, eID1), []*dal.EntityContact{
		&dal.EntityContact{
			Type:  dal.EntityContactTypePhone,
			Value: "+12345678911",
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType:  directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{EntityID: eID1.String()},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 2,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{
		&dal.EntityContact{
			EntityID: eID1,
			Type:     dal.EntityContactTypePhone,
			Value:    "+12345678910",
		},
		&dal.EntityContact{
			EntityID: eID1,
			Type:     dal.EntityContactTypeEmail,
			Value:    "test@email.com",
		},
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
	}, nil))
	resp, err := s.CreateContacts(context.Background(), &directory.CreateContactsRequest{
		EntityID: eID1.String(),
		Contacts: []*directory.Contact{
			&directory.Contact{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12345678910",
			},
			&directory.Contact{
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
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eCID1, err := dal.NewEntityContactID()
	test.OK(t, err)
	eCID2, err := dal.NewEntityContactID()
	test.OK(t, err)
	phoneType := dal.EntityContactTypePhone
	emailType := dal.EntityContactTypeEmail
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateEntityContact, eCID1, &dal.EntityContactUpdate{
		Type:  &phoneType,
		Value: ptr.String("+12345678910"),
		Label: ptr.String(""),
	}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.UpdateEntityContact, eCID2, &dal.EntityContactUpdate{
		Type:  &emailType,
		Value: ptr.String("test@email.com"),
		Label: ptr.String("NewLabel"),
	}).WithReturns(int64(1), nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Type:        dal.EntityTypeInternal,
	}, nil))

	resp, err := s.UpdateContacts(context.Background(), &directory.UpdateContactsRequest{
		EntityID: eID1.String(),
		Contacts: []*directory.Contact{
			&directory.Contact{
				ID:          eCID1.String(),
				ContactType: directory.ContactType_PHONE,
				Value:       "+12345678910",
			},
			&directory.Contact{
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
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
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type: dal.EntityTypeInternal,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{
		DisplayName:   ptr.String("batman"),
		FirstName:     ptr.String(""),
		LastName:      ptr.String(""),
		MiddleInitial: ptr.String(""),
		GroupName:     ptr.String(""),
		ShortTitle:    ptr.String(""),
		LongTitle:     ptr.String(""),
		Note:          ptr.String("I am the knight"),
	}))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContactsForEntityID, eID1))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{}))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Note:        "I am the knight",
		Type:        dal.EntityTypeInternal,
	}, nil))

	resp, err := s.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
		EntityID: eID1.String(),
		EntityInfo: &directory.EntityInfo{
			DisplayName: "batman",
			Note:        "I am the knight",
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, "I am the knight", resp.Entity.Info.Note)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestUpdateEntityWithContacts(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		Type: dal.EntityTypeInternal,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateEntity, eID1, &dal.EntityUpdate{
		DisplayName:   ptr.String("batman"),
		FirstName:     ptr.String(""),
		LastName:      ptr.String(""),
		MiddleInitial: ptr.String(""),
		GroupName:     ptr.String(""),
		ShortTitle:    ptr.String(""),
		LongTitle:     ptr.String(""),
		Note:          ptr.String("I am the knight"),
	}))

	dl.Expect(mock.NewExpectation(dl.DeleteEntityContactsForEntityID, eID1))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContacts, []*dal.EntityContact{
		&dal.EntityContact{
			EntityID:    eID1,
			Value:       "1",
			Provisioned: true,
			Label:       "Label1",
			Type:        dal.EntityContactTypeEmail,
		},
		&dal.EntityContact{
			EntityID:    eID1,
			Value:       "2",
			Provisioned: false,
			Label:       "Label2",
			Type:        dal.EntityContactTypePhone,
		},
	}))

	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:          eID1,
		DisplayName: "batman",
		Note:        "I am the knight",
		Type:        dal.EntityTypeInternal,
	}, nil))

	resp, err := s.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
		EntityID: eID1.String(),
		EntityInfo: &directory.EntityInfo{
			DisplayName: "batman",
			Note:        "I am the knight",
		},
		Contacts: []*directory.Contact{
			&directory.Contact{
				Value:       "1",
				Provisioned: true,
				Label:       "Label1",
				ContactType: directory.ContactType_EMAIL,
			},
			&directory.Contact{
				Value:       "2",
				Provisioned: false,
				Label:       "Label2",
				ContactType: directory.ContactType_PHONE,
			},
		},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "batman", resp.Entity.Info.DisplayName)
	test.Equals(t, "I am the knight", resp.Entity.Info.Note)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}
