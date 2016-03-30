package main

import (
	"github.com/sprucehealth/backend/svc/directory"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

func TestLeaveThreadMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	organizationID := "e_org"
	ctx = gqlctx.WithAccount(ctx, acc)

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, "t_1", "").WithReturns(&threading.Thread{
		ID:             "t_1",
		Type:           threading.ThreadType_TEAM,
		OrganizationID: organizationID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.EntityForAccountID, organizationID, "a_1").WithReturns(&directory.Entity{
		ID: "eor",
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:              "t_1",
		RemoveMemberEntityIDs: []string{"eor"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:          "t_1",
			Type:        threading.ThreadType_TEAM,
			SystemTitle: "Person1, Person2",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			leaveThread(input: {
				clientMutationId: "a1b2c3",
				threadID: "t_1",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"leaveThread": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)
}
