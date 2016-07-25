package query

import "github.com/sprucehealth/graphql"

// newDateType returns a type object representing a date in time
func newDateType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Date",
		Fields: graphql.Fields{
			"month": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"day":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"year":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})
}
