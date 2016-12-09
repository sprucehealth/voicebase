package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
)

// TODO: mraines: Add more complex subqueries to test
func TestCreatePatientAccountMutation(t *testing.T) {
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
			Type: invite.LOOKUP_INVITE_RESPONSE_PATIENT,
			Invite: &invite.LookupInviteResponse_Patient{
				Patient: &invite.PatientInvite{
					Patient: &invite.Patient{
						ParkedEntityID: "parkedEntityID",
					},
					OrganizationEntityID: "e_org_inv",
				},
			},
		}, nil),
	)

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "parkedEntityID",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
			Depth: 0,
		},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_PHONE,
					Value:       "+14155551212",
				},
			},
		},
	}, nil))

	// Assert that our email was verified
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "emailToken").WithReturns("someone@somewhere.com", nil))

	account := &auth.Account{
		ID:   "a_1",
		Type: auth.AccountType_PATIENT,
	}
	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someoneElse@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
		Type:        auth.AccountType_PATIENT,
		Duration:    auth.TokenDuration_LONG,
		DeviceID:    "DevID",
		Platform:    auth.Platform_ANDROID,
	}).WithReturns(&auth.CreateAccountResponse{
		Account: account,
		Token: &auth.AuthToken{
			Value:               "token",
			ExpirationEpoch:     123123123,
			ClientEncryptionKey: "supersecretkey",
		},
	}, nil))
	gqlctx.InPlaceWithAccount(ctx, account)

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
		UpdateContacts:  true,
		Contacts: []*directory.Contact{
			{ContactType: directory.ContactType_EMAIL, Value: "someoneElse@somewhere.com"},
			{ContactType: directory.ContactType_PHONE, Value: "+14155551212", Verified: true},
		},
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
		{ID: "threadID", OrganizationID: "threadOrg"},
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID: "threadOrg",
		ThreadID:      "threadID",
		SystemTitle:   "first last",
	}).WithReturns(&threading.UpdateThreadResponse{}, nil))

	gomock.InOrder(
		// Clean up our invite
		g.inviteC.EXPECT().MarkInviteConsumed(ctx, &invite.MarkInviteConsumedRequest{
			Token: "InviteToken",
		}).Return(&invite.MarkInviteConsumedResponse{}, nil),
	)

	// Query the account entity
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "a_1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
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
				duration: LONG,
			}) {
				clientMutationId
				token
				clientEncryptionKey
				account {
					id
					type
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
				"id": "a_1",
				"type": "PATIENT"
			},
			"clientEncryptionKey": "supersecretkey",
			"clientMutationId": "a1b2c3",
			"token": "token"
		}
	}
}`, string(b))
}

func TestCreatePatientAccountMutation_PracticeLink(t *testing.T) {
	conc.Testing = true
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)

	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
		Platform: device.Android,
	})

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
			Type: invite.LOOKUP_INVITE_RESPONSE_ORGANIZATION_CODE,
			Invite: &invite.LookupInviteResponse_Organization{
				Organization: &invite.OrganizationInvite{
					OrganizationEntityID: "e_org_inv",
					Token:                "org_token",
					Tags:                 []string{"autotag1", "autotag2"},
				},
			},
		}, nil),
	)

	// Assert that phone was verified
	g.ra.Expect(mock.NewExpectation(g.ra.VerifiedValue, "phoneToken").WithReturns("+12222222222", nil))

	// Create account
	g.ra.Expect(mock.NewExpectation(g.ra.CreateAccount, &auth.CreateAccountRequest{
		FirstName:   "first",
		LastName:    "last",
		Email:       "someoneElse@somewhere.com",
		PhoneNumber: "+14155551212",
		Password:    "password",
		Type:        auth.AccountType_PATIENT,
		Duration:    auth.TokenDuration_LONG,
		DeviceID:    "DevID",
		Platform:    auth.Platform_ANDROID,
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

	// create new entity
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntity, &directory.CreateEntityRequest{
		Type: directory.EntityType_PATIENT,
		InitialMembershipEntityID: "e_org_inv",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "someoneElse@somewhere.com",
			},
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+14155551212",
			},
		},
		EntityInfo: &directory.EntityInfo{
			FirstName: "first",
			LastName:  "last",
		},
	}).WithReturns(&directory.Entity{
		ID: "parkedEntityID",
		Info: &directory.EntityInfo{
			FirstName:   "first",
			LastName:    "last",
			DisplayName: "first last",
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

	// Create the empty thread
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "e_org_inv",
		PrimaryEntityID: "parkedEntityID",
		MemberEntityIDs: []string{"e_org_inv", "parkedEntityID"},
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		Summary:         "first last",
		SystemTitle:     "first last",
		Origin:          threading.THREAD_ORIGIN_ORGANIZATION_CODE,
		Tags:            []string{"autotag1", "autotag2"},
	}).WithReturns(&threading.Thread{ID: "threadID"}, nil))

	// update entity
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
		UpdateContacts:  true,
		Contacts: []*directory.Contact{
			{ContactType: directory.ContactType_EMAIL, Value: "someoneElse@somewhere.com", Verified: false},
			{ContactType: directory.ContactType_PHONE, Value: "+14155551212", Verified: true},
		},
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
		{ID: "threadID", OrganizationID: "threadOrg"},
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID: "threadOrg",
		ThreadID:      "threadID",
		SystemTitle:   "first last",
	}).WithReturns(&threading.UpdateThreadResponse{}, nil))

	// Query the acount entity
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "a_1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
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
				phoneNumber: "+14155551212",
				dob: {
					month: 7,
					day: 25,
					year: 1986,
				},
				phoneVerificationToken: "phoneToken",
				duration: LONG,
			}) {
				clientMutationId
				token
				clientEncryptionKey
				account {
					id
					type
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
				"id": "a_1",
				"type": "PATIENT"
			},
			"clientEncryptionKey": "supersecretkey",
			"clientMutationId": "a1b2c3",
			"token": "token"
		}
	}
}`, string(b))
}
