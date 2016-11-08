package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
)

func TestTextInviteLink_OrganizationCode(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)

	g.svc.inviteAPIDomain = "invite.test.com"
	g.svc.serviceNumber = phone.Number("+11234567890")

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvite, &invite.LookupInviteRequest{
		InviteToken: "token",
	}).WithReturns(&invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_ORGANIZATION_CODE,
		Invite: &invite.LookupInviteResponse_Organization{
			Organization: &invite.OrganizationInvite{
				OrganizationEntityID: "orgID",
				Token:                "token",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "orgID",
		},
	}).WithReturns([]*directory.Entity{
		{
			Info: &directory.EntityInfo{
				DisplayName: "test org",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            "Download the Spruce app now and connect with test org: https://invite.test.com/token [code: token]",
				FromPhoneNumber: "+11234567890",
				ToPhoneNumber:   "+17348465522",
			},
		},
	}))

	res := g.query(ctx, `
		mutation _ {
			textInviteLink(input: {
				clientMutationId: "a1b2c3",
				token: "token",
        phoneNumber: "734 846 5522"
			}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"textInviteLink": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

}

func TestTextInviteLink_PatientInvite(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)

	g.svc.inviteAPIDomain = "invite.test.com"
	g.svc.serviceNumber = phone.Number("+11234567890")

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvite, &invite.LookupInviteRequest{
		InviteToken: "token",
	}).WithReturns(&invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_PATIENT,
		Invite: &invite.LookupInviteResponse_Patient{
			Patient: &invite.PatientInvite{
				OrganizationEntityID: "orgID",
				InviterEntityID:      "entityID",
				Patient: &invite.Patient{
					FirstName:      "PatientFirstName",
					PhoneNumber:    "+17348465523",
					ParkedEntityID: "patientEntityID",
				},
			},
		},
	}, nil))

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.InvitePatients, &invite.InvitePatientsRequest{
		OrganizationEntityID: "orgID",
		InviterEntityID:      "entityID",
		Patients: []*invite.Patient{
			{
				FirstName:      "PatientFirstName",
				PhoneNumber:    "+17348465523",
				ParkedEntityID: "patientEntityID",
			},
		},
	}))

	res := g.query(ctx, `
		mutation _ {
			textInviteLink(input: {
				clientMutationId: "a1b2c3",
				token: "token",
        phoneNumber: "734 846 5522"
			}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"textInviteLink": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

}
