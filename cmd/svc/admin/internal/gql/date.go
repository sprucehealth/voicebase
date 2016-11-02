package gql

import "github.com/sprucehealth/graphql"

// dateType is a type object representing a date in time
var dateType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Date",
	Fields: graphql.Fields{
		"month": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"day":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"year":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})
