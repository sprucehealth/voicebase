package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
	"strings"
)

var medicationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Medication",
		Fields: graphql.Fields{
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"route":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"form":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"dosages": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(medicationDosageType))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Medication)
			return ok
		},
	},
)

var medicationDosageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MedicationDosage",
		Fields: graphql.Fields{
			"dosage":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"dispenseType": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"otc":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.MedicationDosage)
			return ok
		},
	},
)

var stubMedications = []*models.Medication{
	{ID: "Benzoyl Peroxide Topical (topical - lotion)", Name: "Benzoyl Peroxide Topical", Route: "topical", Form: "lotion", Dosages: []*models.MedicationDosage{
		{Dosage: "10%", DispenseType: "Tube(s)", OTC: true},
		{Dosage: "3%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "4%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "4.25%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "5%", DispenseType: "Tube(s)", OTC: true},
		{Dosage: "6%", DispenseType: "Tube(s)", OTC: true},
		{Dosage: "7%", DispenseType: "Tube(s)", OTC: true},
		{Dosage: "9%", DispenseType: "Tube(s)", OTC: false},
	}},
	{ID: "Omeprazole (oral - delayed release capsule)", Name: "Omeprazole", Route: "oral", Form: "delayed release capsule", Dosages: []*models.MedicationDosage{
		{Dosage: "10 mg", DispenseType: "Capsule(s)", OTC: false},
		{Dosage: "20 mg", DispenseType: "Capsule(s)", OTC: true},
		{Dosage: "40 mg", DispenseType: "Capsule(s)", OTC: false},
	}},
	{ID: "Tretinoin Topical (topical - cream)", Name: "Tretinoin Topical", Route: "topical", Form: "cream", Dosages: []*models.MedicationDosage{
		{Dosage: "0.025%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.05%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.1%", DispenseType: "Tube(s)", OTC: false},
	}},
	{ID: "Tretinoin Topical (topical - gel)", Name: "Tretinoin Topical", Route: "topical", Form: "gel", Dosages: []*models.MedicationDosage{
		{Dosage: "0.01%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.025%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.04%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.05%", DispenseType: "Tube(s)", OTC: false},
		{Dosage: "0.1%", DispenseType: "Tube(s)", OTC: false},
	}},
}

var medicationSearchQuery = &graphql.Field{
	Type: graphql.NewNonNull(graphql.NewList(medicationType)),
	Args: graphql.FieldConfigArgument{
		"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		acc := gqlctx.Account(p.Context)

		_ = ctx
		_ = acc

		nameQuery := p.Args["name"].(string)

		// TODO: actually implement this against DoseSpot API

		var meds []*models.Medication
		nameQuery = strings.ToLower(nameQuery)
		for _, m := range stubMedications {
			if strings.Contains(strings.ToLower(m.Name), nameQuery) {
				meds = append(meds, m)
			}
		}
		return meds, nil
	}),
}
