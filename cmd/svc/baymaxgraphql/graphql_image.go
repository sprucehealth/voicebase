package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
)

var imageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Image",
		Fields: graphql.Fields{
			"url":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"width":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"height": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Image)
			return ok
		},
	},
)
