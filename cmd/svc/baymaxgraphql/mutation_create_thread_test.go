package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

func init() {
	conc.Testing = true
}

func TestCreateThreadMutation_NoExistingThreads(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	// organizationID := "e_org"
	ctx = gqlctx.WithAccount(ctx, acc)

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "e_creator",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "e_org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "someone@example.com",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
			Depth: 1,
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(([]*directory.Entity)(nil), grpcErrorf(codes.NotFound, "No entities found")))

	entityInfo := &directory.EntityInfo{
		FirstName:     "firstName",
		MiddleInitial: "middleInitial",
		LastName:      "lastName",
		GroupName:     "groupName",
		ShortTitle:    "shortTitle",
		LongTitle:     "longTitle",
		Note:          "note",
	}
	contacts := []*directory.Contact{
		{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
		{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
	}
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		Type: directory.EntityType_EXTERNAL,
		InitialMembershipEntityID: "e_org",
		Contacts:                  contacts,
		EntityInfo:                entityInfo,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}).WithReturns(&directory.Entity{
		ID:       "e_patient",
		Type:     directory.EntityType_EXTERNAL,
		Contacts: contacts,
		Info: &directory.EntityInfo{
			DisplayName: "firstName middleInitial. lastName, shortTitle",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		UUID:            "zztop",
		OrganizationID:  "e_org",
		MemberEntityIDs: []string{"e_org"},
		FromEntityID:    "e_creator",
		PrimaryEntityID: "e_patient",
		Summary:         "New conversation",
		SystemTitle:     "firstName middleInitial. lastName, shortTitle",
		Type:            threading.THREAD_TYPE_EXTERNAL,
	}).WithReturns(&threading.Thread{
		ID:              "t_1",
		PrimaryEntityID: "e_patient",
		Type:            threading.THREAD_TYPE_EXTERNAL,
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createThread(input: {
				clientMutationId: "a1b2c3",
				uuid: "zztop",
				organizationID: "e_org",
				entityInfo: {
					firstName:     "firstName",
					middleInitial: "middleInitial",
					lastName:      "lastName",
					groupName:     "groupName",
					shortTitle:    "shortTitle",
					longTitle:     "longTitle",
					note:          "note",
					contactInfos:  [
						{type: PHONE, value: "+14155555555", label: "Phone"},
						{type: EMAIL, value: "someone@example.com", label: "Email"},
						{type: PHONE, value: "1", label: "Invalid"}
					],
				},
				createForContactInfo: {type: EMAIL, value: "someone@example.com"},
			}) {
				clientMutationId
				success
				thread {
					id
					allowInternalMessages
					isDeletable
				}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createThread": {
			"clientMutationId": "a1b2c3",
			"success": true,
			"thread": {
				"allowInternalMessages": true,
				"id": "t_1",
				"isDeletable": true
			}
		}
	}
}`, string(b))
}

func TestCreateThreadMutation_DifferentOrg(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "e_creator",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "e_org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "someone@example.com",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			Depth:             1,
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	},
	).WithReturns([]*directory.Entity{
		{
			ID:   "e_existing_1",
			Type: directory.EntityType_EXTERNAL,
			Contacts: []*directory.Contact{
				{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
				{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
			},
			Info: &directory.EntityInfo{
				FirstName:     "differentName",
				MiddleInitial: "middleInitial",
				DisplayName:   "differentName middleInitial. lastName, shortTitle",
				LastName:      "lastName",
				GroupName:     "groupName",
				ShortTitle:    "shortTitle",
				LongTitle:     "longTitle",
				Note:          "note",
			},
			Memberships: []*directory.Entity{
				{ID: "differentOrg"},
			},
		},
	}, nil))

	// The rest should behave like a create because the found entity doesn't match the org

	entityInfo := &directory.EntityInfo{
		FirstName:     "firstName",
		MiddleInitial: "middleInitial",
		LastName:      "lastName",
		GroupName:     "groupName",
		ShortTitle:    "shortTitle",
		LongTitle:     "longTitle",
		Note:          "note",
	}
	contacts := []*directory.Contact{
		{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
		{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
	}
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		Type: directory.EntityType_EXTERNAL,
		InitialMembershipEntityID: "e_org",
		Contacts:                  contacts,
		EntityInfo:                entityInfo,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}).WithReturns(&directory.Entity{
		ID:       "e_patient",
		Type:     directory.EntityType_EXTERNAL,
		Contacts: contacts,
		Info: &directory.EntityInfo{
			DisplayName: "firstName middleInitial. lastName, shortTitle",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		UUID:            "zztop",
		OrganizationID:  "e_org",
		MemberEntityIDs: []string{"e_org"},
		FromEntityID:    "e_creator",
		SystemTitle:     "firstName middleInitial. lastName, shortTitle",
		PrimaryEntityID: "e_patient",
		Summary:         "New conversation",
		Type:            threading.THREAD_TYPE_EXTERNAL,
	}).WithReturns(&threading.Thread{
		ID:              "t_1",
		Type:            threading.THREAD_TYPE_EXTERNAL,
		PrimaryEntityID: "e_patient",
		SystemTitle:     "firstName middleInitial. lastName, shortTitle",
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createThread(input: {
				clientMutationId: "a1b2c3",
				uuid: "zztop",
				organizationID: "e_org",
				entityInfo: {
					firstName:     "firstName",
					middleInitial: "middleInitial",
					lastName:      "lastName",
					groupName:     "groupName",
					shortTitle:    "shortTitle",
					longTitle:     "longTitle",
					note:          "note",
					contactInfos:  [
						{type: PHONE, value: "+14155555555", label: "Phone"},
						{type: EMAIL, value: "someone@example.com", label: "Email"},
					],
				},
				createForContactInfo: {type: EMAIL, value: "someone@example.com"},
			}) {
				clientMutationId
				success
				thread {
					id
					title
					allowInternalMessages
					isDeletable
				}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createThread": {
			"clientMutationId": "a1b2c3",
			"success": true,
			"thread": {
				"allowInternalMessages": true,
				"id": "t_1",
				"isDeletable": true,
				"title": "firstName middleInitial. lastName, shortTitle"
			}
		}
	}
}`, string(b))
}

func TestCreateThreadMutation_ExistingThreads_DifferentName(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "e_creator",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "e_org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "someone@example.com",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			Depth:             1,
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	},
	).WithReturns([]*directory.Entity{
		{
			ID:   "e_existing_1",
			Type: directory.EntityType_EXTERNAL,
			Contacts: []*directory.Contact{
				{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
				{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
			},
			Info: &directory.EntityInfo{
				DisplayName: "(415) 555-5555",
			},
			Memberships: []*directory.Entity{
				{ID: "e_org"},
			},
		},
		{
			ID:   "e_existing_2",
			Type: directory.EntityType_EXTERNAL,
			Contacts: []*directory.Contact{
				{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
				{ContactType: directory.ContactType_PHONE, Value: "+16305555555", Label: "Phone"},
			},
			Info: &directory.EntityInfo{
				DisplayName: "someone@example.com",
			},
			Memberships: []*directory.Entity{
				{ID: "e_org"},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, "e_existing_1", true).WithReturns([]*threading.Thread{
		{ID: "t_1", OrganizationID: "e_org", PrimaryEntityID: "e_existing_1", SystemTitle: "(415) 555-5555", Type: threading.THREAD_TYPE_EXTERNAL},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, "e_existing_2", true).WithReturns([]*threading.Thread{
		{ID: "t_2", OrganizationID: "e_org", PrimaryEntityID: "e_existing_2", SystemTitle: "someone@example.com", Type: threading.THREAD_TYPE_EXTERNAL},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createThread(input: {
				clientMutationId: "a1b2c3",
				uuid: "zztop",
				organizationID: "e_org",
				entityInfo: {
					firstName:     "firstName",
					middleInitial: "middleInitial",
					lastName:      "lastName",
					groupName:     "groupName",
					shortTitle:    "shortTitle",
					longTitle:     "longTitle",
					note:          "note",
					contactInfos:  [
						{type: PHONE, value: "+14155555555", label: "Phone"},
						{type: EMAIL, value: "someone@example.com", label: "Email"},
					],
				},
				createForContactInfo: {type: EMAIL, value: "someone@example.com"},
			}) {
				clientMutationId
				success
				errorCode
				thread {
					allowInternalMessages
					isDeletable
					id
					title
				}
				existingThreads {
					allowInternalMessages
					isDeletable
					id
					title
				}
				nameDiffers
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createThread": {
			"clientMutationId": "a1b2c3",
			"errorCode": "EXISTING_THREAD",
			"existingThreads": [
				{
					"allowInternalMessages": true,
					"id": "t_1",
					"isDeletable": true,
					"title": "(415) 555-5555"
				},
				{
					"allowInternalMessages": true,
					"id": "t_2",
					"isDeletable": true,
					"title": "someone@example.com"
				}
			],
			"nameDiffers": true,
			"success": false,
			"thread": {
				"allowInternalMessages": true,
				"id": "t_1",
				"isDeletable": true,
				"title": "(415) 555-5555"
			}
		}
	}
}`, string(b))
}

func TestCreateThreadMutation_ExistingThreads_SameName(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "e_creator",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "e_org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "someone@example.com",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			Depth:             1,
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	},
	).WithReturns([]*directory.Entity{
		{
			ID:   "e_existing_1",
			Type: directory.EntityType_EXTERNAL,
			Contacts: []*directory.Contact{
				{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
				{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
			},
			Info: &directory.EntityInfo{
				FirstName:     "differentName",
				MiddleInitial: "middleInitial",
				LastName:      "lastName",
				GroupName:     "groupName",
				DisplayName:   "differentName middleInitial. lastName, shortTitle",
				ShortTitle:    "shortTitle",
				LongTitle:     "longTitle",
				Note:          "note",
			},
			Memberships: []*directory.Entity{
				{ID: "e_org"},
			},
		},
		{
			ID:   "e_existing_2",
			Type: directory.EntityType_EXTERNAL,
			Contacts: []*directory.Contact{
				{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
				{ContactType: directory.ContactType_PHONE, Value: "+16305555555", Label: "Phone"},
			},
			Info: &directory.EntityInfo{
				FirstName:     "firstName",
				MiddleInitial: "middleInitial",
				LastName:      "lastName",
				DisplayName:   "firstName middleInitial. lastName, shortTitle",
				GroupName:     "groupName",
				ShortTitle:    "shortTitle",
				LongTitle:     "longTitle",
				Note:          "note",
			},
			Memberships: []*directory.Entity{
				{ID: "e_org"},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, "e_existing_1", true).WithReturns([]*threading.Thread{
		{ID: "t_1", OrganizationID: "e_org", PrimaryEntityID: "e_existing_1", Type: threading.THREAD_TYPE_EXTERNAL},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, "e_existing_2", true).WithReturns([]*threading.Thread{
		{ID: "t_2", OrganizationID: "e_org", PrimaryEntityID: "e_existing_2", Type: threading.THREAD_TYPE_EXTERNAL},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createThread(input: {
				clientMutationId: "a1b2c3",
				uuid: "zztop",
				organizationID: "e_org",
				entityInfo: {
					firstName:     "firstName",
					middleInitial: "middleInitial",
					lastName:      "lastName",
					groupName:     "groupName",
					shortTitle:    "shortTitle",
					longTitle:     "longTitle",
					note:          "note",
					contactInfos:  [
						{type: PHONE, value: "+14155555555", label: "Phone"},
						{type: EMAIL, value: "someone@example.com", label: "Email"},
					],
				},
				createForContactInfo: {type: EMAIL, value: "someone@example.com"},
			}) {
				clientMutationId
				success
				errorCode
				thread {
					id
					allowInternalMessages
					isDeletable
				}
				existingThreads {
					id
					allowInternalMessages
					isDeletable
				}
				nameDiffers
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createThread": {
			"clientMutationId": "a1b2c3",
			"errorCode": "EXISTING_THREAD",
			"existingThreads": [
				{
					"allowInternalMessages": true,
					"id": "t_1",
					"isDeletable": true
				},
				{
					"allowInternalMessages": true,
					"id": "t_2",
					"isDeletable": true
				}
			],
			"nameDiffers": false,
			"success": false,
			"thread": {
				"allowInternalMessages": true,
				"id": "t_2",
				"isDeletable": true
			}
		}
	}
}`, string(b))
}
