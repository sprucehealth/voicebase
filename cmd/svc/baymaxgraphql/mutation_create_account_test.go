package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestCreateAccountMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *account
	ctx = ctxWithAccount(ctx, acc)

	// Verify phone number token
	g.authC.Expect(mock.NewExpectation(g.authC.VerifiedValue, &auth.VerifiedValueRequest{
		Token: "validToken",
	}).WithReturns(&auth.VerifiedValueResponse{
		Value: "+14155551212",
	}, nil))

	// Create account
	g.authC.Expect(mock.NewExpectation(g.authC.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someone@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
	}).WithReturns(&auth.CreateAccountResponse{
		Account: &auth.Account{
			ID: "a_1",
		},
		Token: &auth.AuthToken{
			Value:               "token",
			ExpirationEpoch:     123123123,
			ClientEncryptionKey: "supersecretkey",
		},
	}, nil))

	// Create organization
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			GroupName:   "org",
			DisplayName: "org",
		},
		Type: directory.EntityType_ORGANIZATION,
	}).WithReturns(&directory.CreateEntityResponse{
		Entity: &directory.Entity{
			ID: "e_org",
		},
	}, nil))

	// Create internal entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			FirstName:   "first",
			LastName:    "last",
			DisplayName: "first last",
		},
		Type:                      directory.EntityType_INTERNAL,
		ExternalID:                "a_1",
		InitialMembershipEntityID: "e_org",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
				Provisioned: false,
			},
		},
	}).WithReturns(&directory.CreateEntityResponse{
		Entity: &directory.Entity{
			ID: "e_int",
		},
	}, nil))

	// Create saved query
	g.thC.Expect(mock.NewExpectation(g.thC.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		OrganizationID: "e_org",
		EntityID:       "e_int",
		Query:          nil,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_1",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createAccount(input: {
				clientMutationId: "a1b2c3",
				email: "someone@somewhere.com",
				password: "password",
				phoneNumber: "415-555-1212",
				firstName: "first",
				lastName: "last",
				organizationName: "org",
				phoneVerificationToken: "validToken",
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
		"createAccount": {
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

func TestCreateAccountMutation_InvalidName(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *account
	ctx = ctxWithAccount(ctx, acc)

	res := g.query(ctx, `
		mutation _ ($firstName: String!) {
			createAccount(input: {
				clientMutationId: "a1b2c3",
				email: "someone@somewhere.com",
				password: "password",
				phoneNumber: "415-555-1212",
				firstName: $firstName,
				lastName: "last",
				organizationName: "org",
				phoneVerificationToken: "validToken",
			}) {
				clientMutationId
				success
				errorCode
			}
		}`,
		map[string]interface{}{
			"firstName": "first😎",
		})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createAccount": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVALID_FIRST_NAME",
			"success": false
		}
	}
}`, string(b))
}

func TestCreateAccountMutation_InviteColleague(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *account
	ctx = ctxWithAccount(ctx, acc)
	ctx = ctxWithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	// Fetch invite info
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
		Type: invite.LookupInviteResponse_COLLEAGUE,
		Invite: &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				Colleague: &invite.Colleague{
					Email:       "someone@example.com",
					PhoneNumber: "+14155551212",
				},
				OrganizationEntityID: "e_org_inv",
			},
		},
	}, nil))

	// Verify phone number token
	g.authC.Expect(mock.NewExpectation(g.authC.VerifiedValue, &auth.VerifiedValueRequest{
		Token: "validToken",
	}).WithReturns(&auth.VerifiedValueResponse{
		Value: "+14155551212",
	}, nil))

	// Create account
	g.authC.Expect(mock.NewExpectation(g.authC.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someone@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
	}).WithReturns(&auth.CreateAccountResponse{
		Account: &auth.Account{
			ID: "a_1",
		},
		Token: &auth.AuthToken{
			Value:               "token",
			ExpirationEpoch:     123123123,
			ClientEncryptionKey: "supersecretkey",
		},
	}, nil))

	// Create internal entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			FirstName:   "first",
			LastName:    "last",
			DisplayName: "first last",
		},
		Type:                      directory.EntityType_INTERNAL,
		ExternalID:                "a_1",
		InitialMembershipEntityID: "e_org_inv",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
				Provisioned: false,
			},
		},
	}).WithReturns(&directory.CreateEntityResponse{
		Entity: &directory.Entity{
			ID: "e_int",
		},
	}, nil))

	// Create saved query
	g.thC.Expect(mock.NewExpectation(g.thC.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		OrganizationID: "e_org_inv",
		EntityID:       "e_int",
		Query:          nil,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_1",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			createAccount(input: {
				clientMutationId: "a1b2c3",
				email: "someone@somewhere.com",
				password: "password",
				phoneNumber: "415-555-1212",
				firstName: "first",
				lastName: "last",
				organizationName: "org",
				phoneVerificationToken: "validToken",
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
		"createAccount": {
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
