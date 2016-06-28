package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

func TestCallQuery(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.IPCall, "ipc_123").WithReturns(
		&excomms.IPCall{
			ID:    "ipc_123",
			Token: "token",
			Participants: []*excomms.IPCallParticipant{
				{
					EntityID:  "entity_1",
					AccountID: "account_1",
					Identity:  "identity_1",
					State:     excomms.IPCallState_ACCEPTED,
					Role:      excomms.IPCallParticipantRole_CALLER,
				},
				{
					EntityID:  "entity_2",
					AccountID: "account_2",
					Identity:  "identity_2",
					State:     excomms.IPCallState_PENDING,
					Role:      excomms.IPCallParticipantRole_RECIPIENT,
				},
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			call(id: "ipc_123") {
				id
				accessToken
				role
				caller {
					state
					twilioIdentity
				}
				recipients {
					state
					twilioIdentity
				}
				allowVideo
				videoEnabledByDefault
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"call": {
			"accessToken": "token",
			"allowVideo": true,
			"caller": {
				"state": "ACCEPTED",
				"twilioIdentity": "identity_1"
			},
			"id": "ipc_123",
			"recipients": [{
				"state": "PENDING",
				"twilioIdentity": "identity_2"
			}],
			"role": "CALLER",
			"videoEnabledByDefault": true
		}
	}
}`, res)
}