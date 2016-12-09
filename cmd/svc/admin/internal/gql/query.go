package gql

import "github.com/sprucehealth/graphql"

// NewQueryRoot returns the root query object
func NewQueryRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"account":          accountField,
				"entity":           entityField,
				"contact":          contactField,
				"me":               meField,
				"practiceLink":     practiceLinkField,
				"triggeredMessage": triggeredMessageField,
			},
		})
}
