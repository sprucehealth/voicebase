package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
)

var providerAccountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProviderAccount",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
			accountInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type": &graphql.Field{
				Type: graphql.NewNonNull(accountTypeEnum),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return string(models.AccountTypeProvider), nil
				},
			},
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					a := p.Source.(*models.ProviderAccount)
					return accountOrganizations(p, a)
				},
			},
		},
	},
)
