package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
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

var medicationSearchQuery = &graphql.Field{
	Type: graphql.NewNonNull(graphql.NewList(medicationType)),
	Args: graphql.FieldConfigArgument{
		"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)

		nameQuery := p.Args["name"].(string)

		res, err := ram.SearchMedications(ctx, &care.SearchMedicationsRequest{Query: nameQuery})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		meds := make([]*models.Medication, len(res.Medications))
		for i, m := range res.Medications {
			dosages := make([]*models.MedicationDosage, len(m.Strengths))
			for j, st := range m.Strengths {
				dosages[j] = &models.MedicationDosage{
					Dosage:       st.Strength,
					DispenseType: st.DispenseUnit,
					OTC:          st.OTC,
				}
			}
			meds[i] = &models.Medication{
				ID:      m.ID,
				Name:    m.Name,
				Route:   m.Route,
				Form:    m.Form,
				Dosages: dosages,
			}
		}
		return meds, nil
	}),
}
