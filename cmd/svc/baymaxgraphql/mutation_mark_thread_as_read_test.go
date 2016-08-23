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

	g.ra.Expect(mock.NewExpectation(g.ra.Threads, &threading.ThreadsRequest{
		ViewerEntityID: "e_12345",
		ThreadIDs:      []string{"t_1", "t_2"},
	}).WithReturns(&threading.ThreadsResponse{
		Threads: []*threading.Thread{
			{
				ID:   "t_1",
				Type: threading.THREAD_TYPE_EXTERNAL,
			},
			{
				ID:   "t_2",
				Type: threading.THREAD_TYPE_TEAM,
			},
		},
	}, nil))

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
        threads {
        	id
        }
      }
    }`, nil)
	responseEquals(t, `{
    "data": {
      "markThreadsAsRead": {
        "clientMutationId": "a1b2c3",
        "success": true,
        "threads": [{
        	"id": "t_1"
        },
        {
        	"id": "t_2"
		}]
      }
    }}`, res)
}
