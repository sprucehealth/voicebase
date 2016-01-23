package main

import "github.com/graphql-go/graphql"

var subdomainType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Subdomain",
		Fields: graphql.Fields{
			"available": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*subdomain)
			return ok
		},
	},
)
