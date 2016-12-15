package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestInvitePatients_RequirePhoneAndEmail(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)
	orgID := "orgID"
	parkedEntityID := "patient_entity_id"
	entID := "entID"

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	patientEntity := &directory.Entity{
		ID:   parkedEntityID,
		Type: directory.EntityType_PATIENT,
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "test@example.com",
			},
		},
		Info: &directory.EntityInfo{
			FirstName:   "firstName",
			LastName:    "lastName",
			DisplayName: "firstName lastName",
		},
	}

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		Type: directory.EntityType_PATIENT,
		InitialMembershipEntityID: orgID,
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "test@example.com",
			},
		},
		EntityInfo: &directory.EntityInfo{
			FirstName: "firstName",
			LastName:  "lastName",
		},
	}).WithReturns(patientEntity, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  orgID,
		PrimaryEntityID: parkedEntityID,
		MemberEntityIDs: []string{orgID, parkedEntityID},
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		Summary:         "firstName lastName",
		SystemTitle:     "firstName lastName",
		Origin:          threading.THREAD_ORIGIN_PATIENT_INVITE,
	}).WithReturns(&threading.Thread{
		ID:              "thread_id",
		PrimaryEntityID: parkedEntityID,
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
	}, nil))

	// for empty state
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: parkedEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		patientEntity,
	}, nil))

	// to determine delete
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: parkedEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		patientEntity,
	}, nil))

	g.inviteC.EXPECT().InvitePatients(ctx, &invite.InvitePatientsRequest{
		OrganizationEntityID: orgID,
		InviterEntityID:      entID,
		Patients: []*invite.Patient{
			{
				FirstName:      "firstName",
				PhoneNumber:    "+11234567890",
				Email:          "test@example.com",
				ParkedEntityID: parkedEntityID,
			},
		},
	})

	res := g.query(ctx, `
		mutation _ {
			invitePatients(input: {
				clientMutationId: "a1b2c3",
				organizationID: "orgID",
				patients: [{
					firstName: "firstName",
					lastName: "lastName",
					email: "test@example.com",
					phoneNumber: "(123) 456-7890"
					}]
			}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"invitePatients": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}

func TestInvitePatients_Phone(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)
	orgID := "orgID"
	parkedEntityID := "patient_entity_id"
	entID := "entID"

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
				},
			},
		},
	}, nil))

	patientEntity := &directory.Entity{
		ID:   parkedEntityID,
		Type: directory.EntityType_PATIENT,
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
			},
		},
		Info: &directory.EntityInfo{
			FirstName:   "firstName",
			LastName:    "lastName",
			DisplayName: "firstName lastName",
		},
	}

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		Type: directory.EntityType_PATIENT,
		InitialMembershipEntityID: orgID,
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
			},
		},
		EntityInfo: &directory.EntityInfo{
			FirstName: "firstName",
			LastName:  "lastName",
		},
	}).WithReturns(patientEntity, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  orgID,
		PrimaryEntityID: parkedEntityID,
		MemberEntityIDs: []string{orgID, parkedEntityID},
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		Summary:         "firstName lastName",
		SystemTitle:     "firstName lastName",
		Origin:          threading.THREAD_ORIGIN_PATIENT_INVITE,
	}).WithReturns(&threading.Thread{
		ID:              "thread_id",
		PrimaryEntityID: parkedEntityID,
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
	}, nil))

	// for empty state
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: parkedEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		patientEntity,
	}, nil))

	// to determine delete
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: parkedEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		patientEntity,
	}, nil))

	g.inviteC.EXPECT().InvitePatients(ctx, &invite.InvitePatientsRequest{
		OrganizationEntityID: orgID,
		InviterEntityID:      entID,
		Patients: []*invite.Patient{
			{
				FirstName:      "firstName",
				PhoneNumber:    "+11234567890",
				ParkedEntityID: parkedEntityID,
			},
		},
	})

	res := g.query(ctx, `
		mutation _ {
			invitePatients(input: {
				clientMutationId: "a1b2c3",
				organizationID: "orgID",
				patients: [{
					firstName: "firstName",
					lastName: "lastName",
					phoneNumber: "(123) 456-7890"
					}]
			}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"invitePatients": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}
