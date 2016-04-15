package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

func TestUpdateThreadMutation(t *testing.T) {
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

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:              "t_1",
		UserTitle:             "newTitle",
		AddMemberEntityIDs:    []string{"e1", "e2"},
		RemoveMemberEntityIDs: []string{"e3"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:          "t_1",
			Type:        threading.ThreadType_TEAM,
			UserTitle:   "newTitle",
			SystemTitle: "Person1, Person2",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			updateThread(input: {
				clientMutationId: "a1b2c3",
				threadID: "t_1",
				title: "newTitle",
				addMemberEntityIDs: ["e1", "e2"],
				removeMemberEntityIDs: ["e3"],
			}) {
				clientMutationId
				success
				thread {
					id
					allowInternalMessages
					allowDelete
					allowAddMembers
					allowRemoveMembers
					allowLeave
					allowUpdateTitle
					title
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"updateThread": {
				"clientMutationId": "a1b2c3",
				"success": true,
				"thread": {
					"allowAddMembers": true,
					"allowDelete": false,
					"allowInternalMessages": false,
					"allowLeave": true,
					"allowRemoveMembers": true,
					"allowUpdateTitle": true,
					"id": "t_1",
					"title": "newTitle"
				}
			}
		}}`, res)
}
