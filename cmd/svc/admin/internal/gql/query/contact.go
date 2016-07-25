package query

import "github.com/sprucehealth/graphql"

const (
	contactEnumPhone = "PHONE"
	contactEnumEmail = "EMAIL"
)

// contactEnumType represents the possible enum values mapped to contact types
var contactEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "ContactType",
		Values: graphql.EnumValueConfigMap{
			contactEnumPhone: &graphql.EnumValueConfig{
				Value: contactEnumPhone,
			},
			contactEnumEmail: &graphql.EnumValueConfig{
				Value: contactEnumEmail,
			},
		},
	},
)

// newContactType returns a type object representing an entity contact
func newContactType() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "ContactInfo",
			Fields: graphql.Fields{
				"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"type":        &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
				"value":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"provisioned": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
				"label":       &graphql.Field{Type: graphql.String},
			},
		})
}
