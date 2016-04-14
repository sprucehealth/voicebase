package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

// TODO: mraines: Add more complex subqueries to test
func TestCreatePatientAccountMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)

	ctx = gqlctx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.AttributionData, &invite.AttributionDataRequest{
		DeviceID: "DevID",
	}).WithReturns(&invite.AttributionDataResponse{
		Values: []*invite.AttributionValue{
			{Key: "invite_token", Value: "InviteToken"},
		},
	}, nil))
	g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvite, &invite.LookupInviteRequest{
		Token: "InviteToken",
	}).WithReturns(&invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_PATIENT,
		Invite: &invite.LookupInviteResponse_Patient{
			Patient: &invite.PatientInvite{
				Patient: &invite.Patient{
					ParkedEntityID: "parkedEntityID",
				},
				OrganizationEntityID: "e_org_inv",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entity, "parkedEntityID", []directory.EntityInformation{directory.EntityInformation_CONTACTS}, int64(0)).WithReturns(&directory.Entity{
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
			},
		},
	}, nil))

	// Assert that our email was verified
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "emailToken").WithReturns("someone@somewhere.com", nil))

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someoneElse@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
		Type:        auth.AccountType_PATIENT,
	}).WithReturns(&auth.CreateAccountResponse{
		Account: &auth.Account{
			ID:   "a_1",
			Type: auth.AccountType_PATIENT,
		},
		Token: &auth.AuthToken{
			Value:               "token",
			ExpirationEpoch:     123123123,
			ClientEncryptionKey: "supersecretkey",
		},
	}, nil))

	// Associate the parked account entity
	g.ra.Expect(mock.NewExpectation(g.ra.UnauthorizedCreateExternalIDs, &directory.CreateExternalIDsRequest{
		EntityID:    "parkedEntityID",
		ExternalIDs: []string{"a_1"},
	}).WithReturns(&directory.Entity{
		ID: "e_int",
		Info: &directory.EntityInfo{
			DisplayName: "first last",
		},
	}, nil))

	// Update the parked account entity
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateEntity, &directory.UpdateEntityRequest{
		EntityID:         "parkedEntityID",
		UpdateEntityInfo: true,
		EntityInfo: &directory.EntityInfo{
			FirstName: "first",
			LastName:  "last",
			Gender:    directory.EntityInfo_MALE,
			DOB: &directory.Date{
				Month: 7,
				Day:   25,
				Year:  1986,
			},
		},
		UpdateAccountID: true,
		AccountID:       "a_1",
	}).WithReturns(&directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "first last",
			FirstName:   "first",
			LastName:    "last",
			Gender:      directory.EntityInfo_MALE,
			DOB: &directory.Date{
				Month: 7,
				Day:   25,
				Year:  1986,
			},
		},
	}, nil))

	// Update any threads we find with the new display name
	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, "parkedEntityID", true).WithReturns([]*threading.Thread{
		{ID: "threadID"},
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:    "threadID",
		SystemTitle: "first last",
	}).WithReturns(&threading.UpdateThreadResponse{}, nil))

	// Query the acount entity
	g.ra.Expect(mock.NewExpectation(g.ra.PatientEntity, &models.PatientAccount{
		ID: "a_1",
	}).WithReturns(&directory.Entity{
		Info: &directory.EntityInfo{
			FirstName: "first",
			LastName:  "last",
			DOB: &directory.Date{
				Month: 7,
				Day:   25,
				Year:  1986,
			},
			Gender: directory.EntityInfo_MALE,
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createPatientAccount(input: {
				clientMutationId: "a1b2c3",
				email: "someoneElse@somewhere.com",
				password: "password",
				firstName: "first",
				lastName: "last",
				gender: MALE,
				dob: {
					month: 7,
					day: 25,
					year: 1986,	
				},
				emailVerificationToken: "emailToken",
			}) {
				clientMutationId
				token
				clientEncryptionKey
				account {
					id
					... on PatientAccount {
						entity {
							firstName
							lastName
							dob {
								month
								day
								year
							}
							gender
						}
					}
				}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createPatientAccount": {
			"account": {
				"entity": {
					"dob": {
						"day": 25,
						"month": 7,
						"year": 1986
					},
					"firstName": "first",
					"gender": "MALE",
					"lastName": "last"
				},
				"id": "a_1"
			},
			"clientEncryptionKey": "supersecretkey",
			"clientMutationId": "a1b2c3",
			"token": "token"
		}
	}
}`, string(b))
}
