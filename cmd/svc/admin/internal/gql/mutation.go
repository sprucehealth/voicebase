package gql

import "github.com/sprucehealth/graphql"

// NewMutationRoot returns the root mutation object
func NewMutationRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"disconnectVendorAccount": newDisconnectVendorAccountField(),
				"modifySetting":           newModifySettingField(),
			},
		})
}
