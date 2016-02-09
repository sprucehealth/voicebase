package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/excomms"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestVerifyPhoneNumberForAccountCreationMutation_Invite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *account
	ctx = ctxWithAccount(ctx, acc)
	ctx = ctxWithSpruceHeaders(ctx, &apiservice.SpruceHeaders{
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

	res := g.query(ctx, `
		mutation _ {
			verifyPhoneNumberForAccountCreation(input: {
				clientMutationId: "a1b2c3",
				phoneNumber: "+14155551212"
			}) {
				clientMutationId
				result
				message
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"verifyPhoneNumberForAccountCreation": {
			"clientMutationId": "a1b2c3",
			"message": "The phone number did not match.",
			"result": "INVITE_PHONE_MISMATCH"
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

	g.authC.Expect(mock.NewExpectation(g.authC.CreateVerificationCode, &auth.CreateVerificationCodeRequest{
		Type:          auth.VerificationCodeType_PHONE,
		ValueToVerify: "+14155551212",
	}).WithReturns(&auth.CreateVerificationCodeResponse{
		VerificationCode: &auth.VerificationCode{
			Code:  "123456",
			Token: "TheToken",
		},
	}, nil))

	g.exC.Expect(mock.NewExpectation(g.exC.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:          "Your Spruce verification code is 123456",
				ToPhoneNumber: "+14155551212",
			},
		},
	}).WithReturns(&excomms.SendMessageResponse{}, nil))

	res = g.query(ctx, `
		mutation _ {
			verifyPhoneNumberForAccountCreation(input: {
				clientMutationId: "a1b2c3",
				phoneNumber: "+14155551212"
			}) {
				clientMutationId
				result
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
			"result": "SUCCESS",
			"token": "TheToken"
		}
	}
}`, string(b))
}
