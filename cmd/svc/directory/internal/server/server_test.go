package server

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	mock_dal "github.com/sprucehealth/backend/cmd/svc/directory/internal/dal/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestLookupEntitiesByEntityID(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}), []*dal.Entity{
		&dal.Entity{
			ID:   eID1,
			Name: "entity1",
			Type: dal.EntityTypeExternal,
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
	test.Equals(t, "entity1", resp.Entities[0].Name)
	test.Equals(t, directory.EntityType_EXTERNAL, resp.Entities[0].Type)
	mock.FinishAll(dl)
}

func TestLookupEntitiesByExternalID(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
			ID:   eID1,
			Name: "entity1",
			Type: dal.EntityTypeInternal,
		},
		&dal.Entity{
			ID:   eID2,
			Name: "entity2",
			Type: dal.EntityTypeInternal,
		},
	}, nil))
	resp, err := s.LookupEntities(context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType:  directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{ExternalID: externalID},
	})
	test.OK(t, err)

	test.Equals(t, 2, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
	test.Equals(t, eID2.String(), resp.Entities[1].ID)
	test.Equals(t, "entity2", resp.Entities[1].Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[1].Type)
	mock.FinishAll(dl)
}

func TestLookupEntitiesNoResults(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
	mock.FinishAll(dl)
}

func TestLookupEntitiesByContact(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
			ID:   eID1,
			Name: "entity1",
			Type: dal.EntityTypeInternal,
		},
		&dal.Entity{
			ID:   eID2,
			Name: "entity2",
			Type: dal.EntityTypeInternal,
		},
	}, nil))
	resp, err := s.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue:         contactValue,
		RequestedInformation: &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.Equals(t, 2, len(resp.Entities))
	test.Equals(t, eID1.String(), resp.Entities[0].ID)
	test.Equals(t, "entity1", resp.Entities[0].Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[0].Type)
	test.Equals(t, eID2.String(), resp.Entities[1].ID)
	test.Equals(t, "entity2", resp.Entities[1].Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entities[1].Type)
	mock.FinishAll(dl)
}

func TestLookupEntitiesByContactNoResults(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
		Name:   name,
		Type:   dal.EntityTypeInternal,
		Status: dal.EntityStatusActive,
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
		ID:   eID1,
		Name: name,
		Type: dal.EntityTypeInternal,
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		Name:                      name,
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, eID1.String(), resp.Entity.ID)
	test.Equals(t, name, resp.Entity.Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entity.Type)
	mock.FinishAll(dl)
}

func TestCreateEntityInitialEntityNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
		Name:                      name,
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.NotFound, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCreateEntityEmptyContact(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
		Name:                      name,
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCreateEntityInvalidEmail(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
		Name:                      name,
		Type:                      eType,
		ExternalID:                externalID,
		InitialMembershipEntityID: eID2.String(),
		Contacts:                  contacts,
		RequestedInformation:      &directory.RequestedInformation{},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCreateEntitySparse(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	name := "batman"
	eType := directory.EntityType_INTERNAL
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.InsertEntity, &dal.Entity{
		Name:   name,
		Type:   dal.EntityTypeInternal,
		Status: dal.EntityStatusActive,
	}), eID1, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:   eID1,
		Name: name,
		Type: dal.EntityTypeInternal,
	}, nil))
	resp, err := s.CreateEntity(context.Background(), &directory.CreateEntityRequest{
		Name: name,
		Type: eType,
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, eID1.String(), resp.Entity.ID)
	test.Equals(t, name, resp.Entity.Name)
	test.Equals(t, directory.EntityType_INTERNAL, resp.Entity.Type)
	mock.FinishAll(dl)
}

func TestCreateMembership(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
		ID:   eID1,
		Name: "newmember",
		Type: dal.EntityTypeInternal,
	}, nil))
	resp, err := s.CreateMembership(context.Background(), &directory.CreateMembershipRequest{
		EntityID:       eID1.String(),
		TargetEntityID: eID2.String(),
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Entity)
	test.Equals(t, "newmember", resp.Entity.Name)
	test.Equals(t, eID1.String(), resp.Entity.ID)
}

func TestCreateMembershipEntityNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
	mock.FinishAll(dl)
}

func TestCreateMembershipTargetEntityNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
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
	mock.FinishAll(dl)
}

func TestCreateContact(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	eID1, err := dal.NewEntityID()
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{}, nil))
	dl.Expect(mock.NewExpectation(dl.InsertEntityContact, &dal.EntityContact{
		EntityID: eID1,
		Type:     dal.EntityContactTypePhone,
		Value:    "+12345678910",
	}))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entity, eID1), &dal.Entity{
		ID:   eID1,
		Name: "batman",
		Type: dal.EntityTypeInternal,
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
	test.Equals(t, "batman", resp.Entity.Name)
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
	s := New(dl)
	eID1, err := dal.NewEntityID()
	test.OK(t, err)
	eID2, err := dal.NewEntityID()
	test.OK(t, err)
	eID3, err := dal.NewEntityID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID1}), []*dal.Entity{
		&dal.Entity{
			ID:   eID1,
			Name: "entity1",
			Type: dal.EntityTypeExternal,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.EntityMemberships, eID1), []*dal.EntityMembership{
		&dal.EntityMembership{
			TargetEntityID: eID2,
		},
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Entities, []dal.EntityID{eID2}), []*dal.Entity{
		&dal.Entity{
			ID:   eID2,
			Name: "entity2",
			Type: dal.EntityTypeExternal,
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
	test.Equals(t, "entity1", resp.Entities[0].Name)
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
