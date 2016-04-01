package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
)

var patientAccountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PatientAccount",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
			accountInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					a := p.Source.(*models.PatientAccount)
					return accountOrganizations(p, a)
				},
			},
		},
	},
)
