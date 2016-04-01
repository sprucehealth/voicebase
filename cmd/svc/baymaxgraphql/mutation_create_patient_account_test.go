package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
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

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someone@somewhere.com",
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

	// Create account entity
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			FirstName:   "first",
			LastName:    "last",
			DisplayName: "first last",
			Gender:      directory.EntityInfo_MALE,
			DOB:         &directory.Date{Month: 7, Day: 25, Year: 1986},
		},
		Type:       directory.EntityType_PATIENT,
		ExternalID: "a_1",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
				Provisioned: false,
			},
		},
	}).WithReturns(&directory.Entity{
		ID: "e_int",
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createPatientAccount(input: {
				clientMutationId: "a1b2c3",
				email: "someone@somewhere.com",
				password: "password",
				phoneNumber: "415-555-1212",
				firstName: "first",
				lastName: "last",
				gender: MALE,
				dob: {
					month: 7,
					day: 25,
					year: 1986,	
				},
			}) {
				clientMutationId
				token
				clientEncryptionKey
				account {
					id
				}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createPatientAccount": {
			"account": {
				"id": "a_1"
			},
			"clientEncryptionKey": "supersecretkey",
			"clientMutationId": "a1b2c3",
			"token": "token"
		}
	}
}`, string(b))
}
