package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"golang.org/x/net/context"
)

func TestVerifyPhoneNumberForAccountCreationMutation_Invite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	// Number differs

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
					PhoneNumber: "+16305551212",
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14155551212",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}))

	res := g.query(ctx, `
		mutation _ {
			verifyPhoneNumberForAccountCreation(input: {
				clientMutationId: "a1b2c3",
				phoneNumber: "+14155551212"
			}) {
				clientMutationId
				success
				errorCode
				errorMessage
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"verifyPhoneNumberForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVITE_PHONE_MISMATCH",
			"errorMessage": "The phone number must match the one that was in your invite.",
			"success": false
		}
	}
}`, string(b))

	// Number matches

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
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14155551212",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateVerificationCode, auth.VerificationCodeType_PHONE, "+14155551212").WithReturns(
		&auth.CreateVerificationCodeResponse{
			VerificationCode: &auth.VerificationCode{
				Code:  "123456",
				Token: "TheToken",
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:          "Your Spruce verification code is 123456",
				ToPhoneNumber: "+14155551212",
			},
		},
	}).WithReturns(nil))

	res = g.query(ctx, `
		mutation _ {
			verifyPhoneNumberForAccountCreation(input: {
				clientMutationId: "a1b2c3",
				phoneNumber: "+14155551212"
			}) {
				clientMutationId
				success
				token
				message
			}
		}`, nil)
	b, err = json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"verifyPhoneNumberForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"message": "A verification code has been sent to (415) 555-1212",
			"success": true,
			"token": "TheToken"
		}
	}
}`, string(b))
}

func TestVerifyPhoneNumberForAccountCreationMutation_SprucePhoneNumber(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	g.ra.Expect(mock.NewExpectation(g.ra.EntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14155551212",
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(
		[]*directory.Entity{
			{
				Contacts: []*directory.Contact{
					{
						Provisioned: true,
						Value:       "+14155551212",
					},
				},
			},
		}, nil))

	res := g.query(ctx, `
		mutation _ {
			verifyPhoneNumberForAccountCreation(input: {
				clientMutationId: "a1b2c3",
				phoneNumber: "+14155551212"
			}) {
				clientMutationId
				errorCode
				errorMessage
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"verifyPhoneNumberForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVALID_PHONE_NUMBER",
			"errorMessage": "Please use a non-Spruce number to create an account with.",
			"success": false
		}
	}
}`, string(b))
}

func TestVerifyEmailCodeEntityInfo_Invite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	// Check the verification code
	g.ra.Expect(mock.NewExpectation(g.ra.CheckVerificationCode, "token", "123456").WithReturns(
		&auth.CheckVerificationCodeResponse{
			Value: "email@email.com",
		}, nil))

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

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "parkedEntityID",
		},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			Info: &directory.EntityInfo{
				FirstName: "bat",
				LastName:  "man",
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			checkVerificationCode(input: {
				token: "token",
				code: "123456"
			}) {
				success
				errorCode
				errorMessage
				verifiedEntityInfo {
      				firstName
      				lastName
      				email
    			}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"checkVerificationCode": {
			"errorCode": null,
			"errorMessage": null,
			"success": true,
			"verifiedEntityInfo": {
				"email": "email@email.com",
				"firstName": "bat",
				"lastName": "man"
			}
		}
	}
}`, string(b))
}
