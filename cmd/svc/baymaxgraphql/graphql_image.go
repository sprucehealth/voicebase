package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
)

// ImageArguments represents the standardized image argument set for graphql
type ImageArguments struct {
	Crop   bool `json:"crop"`
	Width  int  `json:"width"`
	Height int  `json:"height"`
}

// ParseImageArguments parses the image arguments out of requests params
func ParseImageArguments(args map[string]interface{}) *ImageArguments {
	imgArgs := &ImageArguments{}
	if args != nil {
		if crop, ok := args["crop"]; ok {
			if crop, ok := crop.(bool); ok {
				imgArgs.Crop = crop
			}
		}
		if width, ok := args["width"]; ok {
			if width, ok := width.(int); ok {
				imgArgs.Width = width
			}
		}
		if height, ok := args["height"]; ok {
			if height, ok := height.(int); ok {
				imgArgs.Height = height
			}
		}
	}
	return imgArgs
}

var baseImageArguments = graphql.FieldConfigArgument{
	"crop":   &graphql.ArgumentConfig{Type: graphql.Boolean},
	"width":  &graphql.ArgumentConfig{Type: graphql.Int},
	"height": &graphql.ArgumentConfig{Type: graphql.Int},
}

// NewImageArguments returns the standardized image arguments
func NewImageArguments(configMap graphql.FieldConfigArgument) graphql.FieldConfigArgument {
	if configMap == nil {
		configMap = graphql.FieldConfigArgument{}
	}
	for fieldName, argConfig := range baseImageArguments {
		configMap[fieldName] = argConfig
	}
	return configMap
}

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
