package gql

import "github.com/sprucehealth/graphql"

const (
	genderMale    = "MALE"
	genderFemale  = "FEMALE"
	genderOther   = "OTHER"
	genderUnknown = "UNKNOWN"
)

// genderEnumType represents the possible enum values mapped to genders
var genderEnumType = graphql.NewEnum(graphql.EnumConfig{
	Name: "Gender",
	Values: graphql.EnumValueConfigMap{
		genderUnknown: &graphql.EnumValueConfig{
			Value: genderUnknown,
		},
		genderMale: &graphql.EnumValueConfig{
			Value: genderMale,
		},
		genderFemale: &graphql.EnumValueConfig{
			Value: genderFemale,
		},
		genderOther: &graphql.EnumValueConfig{
			Value: genderOther,
		},
	},
})
