package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/invite/clientdata"
	"github.com/sprucehealth/backend/svc/media"
	"google.golang.org/grpc/codes"
)

func TestAssociateInviteMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvite, &invite.LookupInviteRequest{
		LookupKeyType: invite.LookupInviteRequest_TOKEN,
		LookupKeyOneof: &invite.LookupInviteRequest_Token{
			Token: "token",
		},
	}).WithReturns(&invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_COLLEAGUE,
		Invite: &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				OrganizationEntityID: "orgID",
				InviterEntityID:      "inviterID",
				Colleague: &invite.Colleague{
					FirstName: "colleagueFirstName",
				},
			},
		},
		Values: []*invite.AttributionValue{{Key: "foo", Value: "bar"}},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "orgID",
		},
	}).WithReturns([]*directory.Entity{{ID: "orgID", ImageMediaID: "mediaID", Info: &directory.EntityInfo{DisplayName: "displayName"}}}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "inviterID",
		},
	}).WithReturns([]*directory.Entity{{ID: "inviterID", Info: &directory.EntityInfo{DisplayName: "inviterDisplayName"}}}, nil))

	cData, err := clientdata.ColleagueInviteClientJSON(
		&directory.Entity{ID: "orgID", Info: &directory.EntityInfo{DisplayName: "displayName"}},
		&directory.Entity{ID: "inviterID", Info: &directory.EntityInfo{DisplayName: "inviterDisplayName"}},
		"colleagueFirstName", "", "")
	test.OK(t, err)
	g.inviteC.Expect(mock.NewExpectation(g.inviteC.SetAttributionData, &invite.SetAttributionDataRequest{
		DeviceID: "deviceID",
		Values:   []*invite.AttributionValue{{Key: "foo", Value: "bar"}, {Key: "client_data", Value: cData}, {Key: "invite_type", Value: "COLLEAGUE"}},
	}).WithReturns(&invite.SetAttributionDataResponse{}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.MediaInfo, "mediaID").WithReturns(&media.MediaInfo{MIME: &media.MIME{Type: "image", Subtype: "png"}}, nil))

	res := g.query(ctx, `
		mutation _ {
			associateInvite(input: {
				clientMutationId: "a1b2c3",
				token: "token",
			}) {
				clientMutationId
				success
				inviteType
				values {
					key
					value
				}
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	bCData, err := json.MarshalIndent(cData, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"associateInvite": {
			"clientMutationId": "a1b2c3",
			"inviteType": "COLLEAGUE",
			"success": true,
			"values": [
				{
					"key": "foo",
					"value": "bar"
				},
				{
					"key": "client_data",
					"value": `+string(bCData)+`
				},
				{
					"key": "invite_type",
					"value": "COLLEAGUE"
				}
			]
		}
	}
}`, string(b))
}

func TestAssociateInviteMutation_NotFound(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	sh := &device.SpruceHeaders{DeviceID: "deviceID"}
	ctx = devicectx.WithSpruceHeaders(ctx, sh)

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvite, &invite.LookupInviteRequest{
		LookupKeyType: invite.LookupInviteRequest_TOKEN,
		LookupKeyOneof: &invite.LookupInviteRequest_Token{
			Token: "token",
		},
	}).WithReturns(&invite.LookupInviteResponse{}, grpcErrorf(codes.NotFound, "not found")))

	res := g.query(ctx, `
		mutation _ {
			associateInvite(input: {
				clientMutationId: "a1b2c3",
				token: "token",
			}) {
				clientMutationId
				success
				errorCode
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"associateInvite": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVALID_INVITE",
			"success": false
		}
	}
}`, string(b))
}
