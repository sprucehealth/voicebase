package gql

import "github.com/sprucehealth/graphql"

// NewMutationRoot returns the root mutation object
func NewMutationRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"createOrganizationLink": createOrganizationLinkField,
				"createTriggeredMessage": createTriggeredMessageField,
				"disableAccount":         disableAccountField,
				"modifyAccountContact":   modifyAccountContactField,
				"modifyPracticeLink":     modifyPracticeLinkField,
				"modifySetting":          modifySettingField,
				"modifyTriggeredMessage": modifyTriggeredMessageField,
				"provisionNumber":        provisionNumberField,
				"updateVendorAccount":    updateVendorAccountField,
			},
		})
}
