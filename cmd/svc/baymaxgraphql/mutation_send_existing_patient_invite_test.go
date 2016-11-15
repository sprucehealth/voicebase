package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
)

func TestSendExistingPatientInvite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	acc = &auth.Account{
		Type: auth.AccountType_PROVIDER,
		ID:   "acc_2",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	patientEntityID := "patientEntityID"

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: patientEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID: patientEntityID,
				Info: &directory.EntityInfo{
					FirstName: "firstName",
					LastName:  "lastName",
				},
				Contacts: []*directory.Contact{
					{
						ID:          "phoneContactID1",
						ContactType: directory.ContactType_PHONE,
						Value:       "+12222222222",
					},
					{
						ID:          "emailContactID1",
						ContactType: directory.ContactType_EMAIL,
						Value:       "test@example.com",
					},
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "e_creator",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "e_org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	gomock.InOrder(
		// Send the patient invite
		g.inviteC.EXPECT().InvitePatients(ctx, &invite.InvitePatientsRequest{
			InviterEntityID:      "e_creator",
			OrganizationEntityID: "e_org",
			Patients: []*invite.Patient{
				{
					FirstName:      "firstName",
					PhoneNumber:    "+12222222222",
					ParkedEntityID: patientEntityID,
				},
			},
		}),
	)

	res := g.query(ctx, `
		mutation _ {
			sendExistingPatientInvite(input: {
				clientMutationId: "a1b2c3",
				organizationID: "e_org",
				entityID: "patientEntityID",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"sendExistingPatientInvite": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}
