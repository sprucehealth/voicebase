package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

func TestPostEventMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		mutation _ {
			postEvent(input: {
				clientMutationId: "a1b2c3",
				eventName: "someEvent",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"postEvent": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)
}
