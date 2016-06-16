package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"golang.org/x/net/context"
)

func TestSubmitVisitAnswers(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "entity_12345"
	visitID := "visit_12345"
	answersJSON := `{"10":{"type:":"q_type_free_text","text":"hello"}}`

	g.ra.Expect(mock.NewExpectation(g.ra.Visit, &care.GetVisitRequest{
		ID: visitID,
	}).WithReturns(&care.GetVisitResponse{
		Visit: &care.Visit{
			ID:       visitID,
			EntityID: entityID,
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateVisitAnswers, &care.CreateVisitAnswersRequest{
		VisitID:       visitID,
		ActorEntityID: entityID,
		AnswersJSON:   answersJSON,
	}))

	res := g.query(ctx, `
		mutation _ ($visitID: ID!, $answersJSON: String!) {
		submitVisitAnswers(input: {
			clientMutationId: "a1b2c3",
			visitID: $visitID,
			answersJSON: $answersJSON,
			}) {
				clientMutationId
				success
			}
		}`, map[string]interface{}{
		"organizationId": entityID,
		"visitID":        visitID,
		"answersJSON":    answersJSON,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"submitVisitAnswers": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}
