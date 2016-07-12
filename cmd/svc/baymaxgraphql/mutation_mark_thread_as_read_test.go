package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestMarkThreadsAsReadMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	organizationID := "e_org"
	ctx = gqlctx.WithAccount(ctx, acc)

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "e_12345",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: organizationID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.MarkThreadsAsRead, &threading.MarkThreadsAsReadRequest{
		EntityID: "e_12345",
		Seen:     true,
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID:             "t_1",
				LastMessageTimestamp: uint64(12345),
			},
			{
				ThreadID:             "t_2",
				LastMessageTimestamp: uint64(12345),
			},
		},
	}))

	res := g.query(ctx, `
    mutation _ {
      markThreadsAsRead(input: {
        clientMutationId: "a1b2c3",
        seen: true,
        organizationID: "e_org",
        threadWatermarks: [
          {
            threadID: "t_1",
            lastMessageTimestamp: 12345,
          },
          {
            threadID: "t_2",
            lastMessageTimestamp: 12345,
          },
        ]
      }) {
        clientMutationId
        success
      }
    }`, nil)
	responseEquals(t, `{
    "data": {
      "markThreadsAsRead": {
        "clientMutationId": "a1b2c3",
        "success": true
      }
    }}`, res)
}
