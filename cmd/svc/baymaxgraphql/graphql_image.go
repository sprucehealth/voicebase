package main

import "github.com/graphql-go/graphql"

var imageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Image",
		Fields: graphql.Fields{
			"url":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"width":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"height": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*image)
			return ok
		},
	},
)
