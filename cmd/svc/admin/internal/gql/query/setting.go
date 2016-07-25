package query

import "github.com/sprucehealth/graphql"

// newContactType returns a type object representing an entity contact
func newSettingType() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Setting",
			Fields: graphql.Fields{
				"type":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"subkey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"value":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			},
		})
}
