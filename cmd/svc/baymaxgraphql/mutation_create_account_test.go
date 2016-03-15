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

func TestCreateAccountMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *models.Account
	ctx = gqlctx.WithAccount(ctx, acc)

	// Verify phone number token
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "validToken").WithReturns("+14155551212", nil))

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
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
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			GroupName:   "org",
			DisplayName: "org",
		},
		Type: directory.EntityType_ORGANIZATION,
	}).WithReturns(&directory.Entity{
		ID: "e_org",
	}, nil))

	// Create internal entity
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
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
	}).WithReturns(&directory.Entity{
		ID: "e_int",
	}, nil))

	// Create saved query
	g.ra.Expect(mock.NewExpectation(g.ra.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		OrganizationID: "e_org",
		EntityID:       "e_int",
		Query:          nil,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_1",
		},
	}, nil))

	// Create linked support threads
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: supportThreadTitle,
			GroupName:   supportThreadTitle,
		},
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: "e_org",
	}).WithReturns(&directory.Entity{
		ID: "e_sys_1",
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: supportThreadTitle + " (org)",
			GroupName:   supportThreadTitle + " (org)",
		},
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: "spruce_org",
	}).WithReturns(&directory.Entity{
		ID: "e_sys_2",
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CreateLinkedThreads, &threading.CreateLinkedThreadsRequest{
		Organization1ID:      "e_org",
		Organization2ID:      "spruce_org",
		PrimaryEntity1ID:     "e_sys_1",
		PrimaryEntity2ID:     "e_sys_2",
		PrependSenderThread1: false,
		PrependSenderThread2: true,
		Summary:              supportThreadTitle + ": " + teamSpruceInitialText[:128],
		Text:                 teamSpruceInitialText,
		Type:                 threading.ThreadType_SUPPORT,
		SystemTitle1:         supportThreadTitle,
		SystemTitle2:         supportThreadTitle + " (org)",
	}).WithReturns(&threading.CreateLinkedThreadsResponse{}, nil))

	// Create onboarding thread
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			DisplayName: onboardingThreadTitle,
			GroupName:   onboardingThreadTitle,
		},
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: "e_org",
	}).WithReturns(&directory.Entity{
		ID: "e_sys_3",
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.CreateOnboardingThread, &threading.CreateOnboardingThreadRequest{
		OrganizationID:  "e_org",
		PrimaryEntityID: "e_sys_3",
		SystemTitle:     onboardingThreadTitle,
	}).WithReturns(&threading.CreateOnboardingThreadResponse{}, nil))

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
	var acc *models.Account
	ctx = gqlctx.WithAccount(ctx, acc)

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
	var acc *models.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = gqlctx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
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
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "validToken").WithReturns("+14155551212", nil))

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
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
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
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
	}).WithReturns(&directory.Entity{
		ID: "e_int",
	}, nil))

	// Create saved query
	g.ra.Expect(mock.NewExpectation(g.ra.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		OrganizationID: "e_org_inv",
		EntityID:       "e_int",
		Query:          nil,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_1",
		},
	}, nil))

	// Analytics looks up the organization to get the name for invites
	g.ra.Expect(mock.NewExpectation(g.ra.Entity, "e_org_inv", []directory.EntityInformation(nil), int64(0)).WithReturns(
		&directory.Entity{
			Info: &directory.EntityInfo{
				DisplayName: "The Org",
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
