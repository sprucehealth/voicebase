package query

import "github.com/sprucehealth/graphql"

// NewRoot returns the root query object
func NewRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"me":     newMeField(),
				"entity": newEntityField(),
			},
		})
}
