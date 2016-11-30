package gql

import "github.com/sprucehealth/graphql"

// NewQueryRoot returns the root query object
func NewQueryRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"me":           meField,
				"entity":       entityField,
				"account":      accountField,
				"practiceLink": practiceLinkField,
			},
		})
}
