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
	"github.com/sprucehealth/backend/svc/invite"
	"golang.org/x/net/context"
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
		Token: "token",
	}).WithReturns(&invite.LookupInviteResponse{
		Type:   invite.LookupInviteResponse_COLLEAGUE,
		Values: []*invite.AttributionValue{{Key: "foo", Value: "bar"}},
	}, nil))

	g.inviteC.Expect(mock.NewExpectation(g.inviteC.SetAttributionData, &invite.SetAttributionDataRequest{
		DeviceID: "deviceID",
		Values:   []*invite.AttributionValue{{Key: "foo", Value: "bar"}},
	}).WithReturns(&invite.SetAttributionDataResponse{}, nil))

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
		Token: "token",
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
