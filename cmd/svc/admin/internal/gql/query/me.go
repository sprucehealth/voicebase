package query

import (
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/handlers/auth"
	"github.com/sprucehealth/graphql"
)

// newMeField returns a graphql field for Querying a Me object
func newMeField() *graphql.Field {
	return &graphql.Field{
		Type:    graphql.NewNonNull(newMeType()),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) { return models.Me{}, nil },
	}
}

// newMeType returns an instance of the Me graphql type
// TODO: Have this done by a `maker` might be overkill. But idea here is to test out a new pattern to improve testability
func newMeType() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Me",
			Fields: graphql.Fields{
				"username": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return auth.UID(p.Context), nil
					},
				},
				// For single query purposes allow entities to be lookedup inside a `me` call
				"entity": newEntityField(),
			},
		},
	)
}
