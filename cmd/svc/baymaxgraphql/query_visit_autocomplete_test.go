package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"golang.org/x/net/context"
)

func TestVisitAutocompleteSearchQuery_SelfReported(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.SearchSelfReportedMedications, &care.SearchSelfReportedMedicationsRequest{Query: "Advil"}).WithReturns(
		&care.SearchSelfReportedMedicationsResponse{
			Results: []string{
				"Advil 1",
				"Advil 2",
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			visitAutocompleteSearch(query: "Advil", source:PATIENT_DRUG,  visitID: "visit_1", questionID: "question_1") {
        title
        subtitle
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"visitAutocompleteSearch": [{
			"title": "Advil 1",
			"subtitle": ""
		},
    {
			"title": "Advil 2",
			"subtitle": ""
		}]
	}
}`, res)
}

func TestVisitAutocompleteSearchQuery_AllergyMedications(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.SearchAllergyMedications, &care.SearchAllergyMedicationsRequest{Query: "Advil"}).WithReturns(
		&care.SearchAllergyMedicationsResponse{
			Results: []string{
				"Advil 1",
				"Advil 2",
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			visitAutocompleteSearch(query: "Advil", source:PATIENT_ALLERGY,  visitID: "visit_1", questionID: "question_1") {
        title
        subtitle
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"visitAutocompleteSearch": [{
			"title": "Advil 1",
			"subtitle": ""
		},
    {
			"title": "Advil 2",
			"subtitle": ""
		}]
	}
}`, res)
}
