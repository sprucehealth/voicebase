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
	"google.golang.org/grpc/codes"
)

func TestVerifyEmailForAccountCreationMutation_Invite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "DevID",
	})

	// No invite

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.AttributionData, &invite.AttributionDataRequest{
		DeviceID: "DevID",
	}).WithReturns((*invite.AttributionDataResponse)(nil), grpcErrorf(codes.NotFound, "Not Found")))

	res := g.query(ctx, `
		mutation _ {
			verifyEmailForAccountCreation(input: {
				clientMutationId: "a1b2c3",
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
		"verifyEmailForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVITE_REQUIRED",
			"errorMessage": "An invite is required to perform email verification with this device.",
			"success": false
		}
	}
}`, string(b))

	// Invite exists

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
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "parkedEntityID",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "someone@example.com",
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateVerificationCode, auth.VerificationCodeType_EMAIL, "someone@example.com").WithReturns(
		&auth.CreateVerificationCodeResponse{
			VerificationCode: &auth.VerificationCode{
				Code:  "123456",
				Token: "TheToken",
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Your Email Verification Code",
				FromName:         "Spruce Support",
				FromEmailAddress: "support@sprucehealth.com",
				Body:             "During sign up, please enter this code when prompted: 123456\nIf you have any troubles, we're here to help - simply reply to this email!\n\nThanks,\nThe Team at Spruce",
				ToEmailAddress:   "someone@example.com",
			},
		},
	}).WithReturns(nil))

	res = g.query(ctx, `
		mutation _ {
			verifyEmailForAccountCreation(input: {
				clientMutationId: "a1b2c3",
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
		"verifyEmailForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"message": "A verification code has been sent to the invited email.",
			"success": true,
			"token": "TheToken"
		}
	}
}`, string(b))
}
