package gql

import (
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/auth"
	"github.com/sprucehealth/graphql"
)

// meField is a graphql field for Querying a Me object
var meField = &graphql.Field{
	Type:    graphql.NewNonNull(meType),
	Resolve: func(p graphql.ResolveParams) (interface{}, error) { return models.Me{}, nil },
}

// newMeType is an instance of the Me graphql type
// TODO: Have this done by a `maker` might be overkill. But idea here is to test out a new pattern to improve testability
var meType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Me",
		Fields: graphql.Fields{
			"username": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.String),
				Resolve: meResolve,
			},
			// For single query purposes allow entities to be lookedup inside a `me` call
			"entity":  entityField,
			"account": accountField,
		},
	},
)

func meResolve(p graphql.ResolveParams) (interface{}, error) {
	return auth.UID(p.Context), nil
}
