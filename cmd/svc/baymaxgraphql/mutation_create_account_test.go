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
	"github.com/sprucehealth/backend/svc/threading"
)

func TestCreateAccountMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
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
		Type:        auth.AccountType_PROVIDER,
		Duration:    auth.TokenDuration_SHORT,
	}).WithReturns(&auth.CreateAccountResponse{
		Account: &auth.Account{
			ID:   "a_1",
			Type: auth.AccountType_PROVIDER,
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
		Type:      directory.EntityType_ORGANIZATION,
		AccountID: "a_1",
	}).WithReturns(&directory.Entity{
		ID: "e_org",
		Info: &directory.EntityInfo{
			DisplayName: "org",
		},
	}, nil))

	// Create internal entity
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			FirstName: "first",
			LastName:  "last",
		},
		Type:                      directory.EntityType_INTERNAL,
		ExternalID:                "a_1",
		InitialMembershipEntityID: "e_org",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
				Provisioned: false,
				Verified:    true,
			},
		},
		AccountID: "a_1",
	}).WithReturns(&directory.Entity{
		ID: "e_int",
		Info: &directory.EntityInfo{
			DisplayName: "first last",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SavedQueryTemplates, "e_org").WithReturns([]*threading.SavedQuery{
		{
			ShortTitle:           "Patient",
			LongTitle:            "All Patient Conversations",
			Description:          "Any new activity in a patient conversation",
			Ordinal:              2000,
			Query:                &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
			Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
			NotificationsEnabled: true,
		},
	}, nil))

	// Create saved query

	g.ra.Expect(mock.NewExpectation(g.ra.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
		EntityID:             "e_int",
		ShortTitle:           "Patient",
		LongTitle:            "All Patient Conversations",
		Description:          "Any new activity in a patient conversation",
		Ordinal:              2000,
		Query:                &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
		NotificationsEnabled: true,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_2",
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
		Type:                 threading.THREAD_TYPE_SUPPORT,
		SystemTitle1:         supportThreadTitle,
		SystemTitle2:         supportThreadTitle + " (org)",
	}).WithReturns(&threading.CreateLinkedThreadsResponse{Thread1: &threading.Thread{}, Thread2: &threading.Thread{}}, nil))

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
	var acc *auth.Account
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

	ctx := initGraphQLContext()

	gomock.InOrder(
		// Get attribution data
		g.inviteC.EXPECT().AttributionData(ctx, &invite.AttributionDataRequest{
			DeviceID: "DevID",
		}).Return(&invite.AttributionDataResponse{
			Values: []*invite.AttributionValue{
				{Key: "invite_token", Value: "InviteToken"},
			},
		}, nil),

		// Get the invite for the token
		g.inviteC.EXPECT().LookupInvite(ctx, &invite.LookupInviteRequest{
			InviteToken: "InviteToken",
		}).Return(&invite.LookupInviteResponse{
			Type: invite.LOOKUP_INVITE_RESPONSE_COLLEAGUE,
			Invite: &invite.LookupInviteResponse_Colleague{
				Colleague: &invite.ColleagueInvite{
					Colleague: &invite.Colleague{
						Email:       "someone@example.com",
						PhoneNumber: "+14155551212",
					},
					OrganizationEntityID: "e_org_inv",
				},
			},
		}, nil),
	)

	// Verify phone number token
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "validToken").WithReturns("+14155551212", nil))

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someone@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
		Type:        auth.AccountType_PROVIDER,
		Duration:    auth.TokenDuration_SHORT,
		DeviceID:    "DevID",
		Platform:    auth.Platform_ANDROID,
	}).WithReturns(&auth.CreateAccountResponse{
		Account: &auth.Account{
			ID:   "a_1",
			Type: auth.AccountType_PROVIDER,
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
			FirstName: "first",
			LastName:  "last",
		},
		Type:                      directory.EntityType_INTERNAL,
		ExternalID:                "a_1",
		InitialMembershipEntityID: "e_org_inv",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
				Provisioned: false,
				Verified:    true,
			},
		},
		AccountID: "a_1",
	}).WithReturns(&directory.Entity{
		ID: "e_int",
		Info: &directory.EntityInfo{
			DisplayName: "first last",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SavedQueryTemplates, "e_org_inv").WithReturns([]*threading.SavedQuery{
		{
			ShortTitle:           "Patient",
			LongTitle:            "All Patient Conversations",
			Description:          "Any new activity in a patient conversation",
			Ordinal:              2000,
			Query:                &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
			NotificationsEnabled: true,
			Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
		},
	}, nil))

	// Create saved query

	g.ra.Expect(mock.NewExpectation(g.ra.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
		EntityID:             "e_int",
		ShortTitle:           "Patient",
		LongTitle:            "All Patient Conversations",
		Description:          "Any new activity in a patient conversation",
		Ordinal:              2000,
		Query:                &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
		NotificationsEnabled: true,
	}).WithReturns(&threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID: "sq_2",
		},
	}, nil))

	gomock.InOrder(
		// Clean up our invite
		g.inviteC.EXPECT().MarkInviteConsumed(ctx, &invite.MarkInviteConsumedRequest{
			Token: "InviteToken",
		}).Return(&invite.MarkInviteConsumedResponse{}, nil),
	)

	// Analytics looks up the organization to get the name for invites
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "e_org_inv",
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				Info: &directory.EntityInfo{
					DisplayName: "The Org",
				},
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
