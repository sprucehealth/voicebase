package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"golang.org/x/net/context"
)

func TestMedicationSearchQuery(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.SearchMedications, &care.SearchMedicationsRequest{Query: "Omep"}).WithReturns(
		&care.SearchMedicationsResponse{
			Medications: []*care.Medication{
				{
					ID:    "Omeprazole (oral - delayed release capsule)",
					Name:  "Omeprazole",
					Route: "oral",
					Form:  "delayed release capsule",
					Strengths: []*care.MedicationStrength{
						{Strength: "10 mg", DispenseUnit: "Capsule(s)", OTC: false},
						{Strength: "20 mg", DispenseUnit: "Capsule(s)", OTC: true},
						{Strength: "40 mg", DispenseUnit: "Capsule(s)", OTC: false},
					},
				},
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			medicationSearch(name: "Omep") {
				id
				name
				route
				form
				dosages {
					dosage
					dispenseType
					otc
				}
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"medicationSearch": [{
			"id": "Omeprazole (oral - delayed release capsule)",
			"name": "Omeprazole",
			"route": "oral",
			"form": "delayed release capsule",
			"dosages": [
				{"dosage": "10 mg", "dispenseType": "Capsule(s)", "otc": false},
				{"dosage": "20 mg", "dispenseType": "Capsule(s)", "otc": true},
				{"dosage": "40 mg", "dispenseType": "Capsule(s)", "otc": false}
			]
		}]
	}
}`, res)
}
