package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
)

var visitAutocompleteSearchResultType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VisitAutocompleteSearchResult",
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.ID},
			"subtitle": &graphql.Field{Type: graphql.String},
			"title":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.VisitAutocompleteSearchResult)
			return ok
		},
	},
)

const (
	visitAutocompleteSearchPatientDrug    = "PATIENT_DRUG"
	visitAutocompleteSearchPatientAllergy = "PATIENT_ALLERGY"
)

var visitAutocompleteSearchQuery = &graphql.Field{
	Type: graphql.NewNonNull(graphql.NewList(visitAutocompleteSearchResultType)),
	Args: graphql.FieldConfigArgument{
		"query":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		"source":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		"visitID":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
		"questionID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Patient(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)

		source := p.Args["source"].(string)
		query := p.Args["query"].(string)

		var results []string
		switch source {
		case visitAutocompleteSearchPatientAllergy:
			res, err := ram.SearchAllergyMedications(ctx, &care.SearchAllergyMedicationsRequest{
				Query: query,
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			results = res.Results
		case visitAutocompleteSearchPatientDrug:
			res, err := ram.SearchSelfReportedMedications(ctx, &care.SearchSelfReportedMedicationsRequest{
				Query: query,
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			results = res.Results
		}

		resultItems := make([]*models.VisitAutocompleteSearchResult, len(results))
		for i, result := range results {
			resultItems[i] = &models.VisitAutocompleteSearchResult{
				Title: result,
			}
		}

		return resultItems, nil
	}),
}
