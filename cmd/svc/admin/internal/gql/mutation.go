package gql

import "github.com/sprucehealth/graphql"

// NewMutationRoot returns the root mutation object
func NewMutationRoot() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"createDefaultSavedThreadQueryTemplates": createDefaultSavedThreadQueryTemplatesField,
				"createOrganizationLink":                 createOrganizationLinkField,
				"createSavedThreadQuery":                 createSavedThreadQueryField,
				"createTriggeredMessage":                 createTriggeredMessageField,
				"deleteSavedThreadQuery":                 deleteSavedThreadQueryField,
				"disableAccount":                         disableAccountField,
				"modifyAccountContact":                   modifyAccountContactField,
				"modifyPracticeLink":                     modifyPracticeLinkField,
				"modifySetting":                          modifySettingField,
				"modifyTriggeredMessage":                 modifyTriggeredMessageField,
				"provisionNumber":                        provisionNumberField,
				"updateSavedThreadQuery":                 updateSavedThreadQueryField,
				"updateVendorAccount":                    updateVendorAccountField,
				"updateSyncConfiguration":                updateSyncConfigurationField,
			},
		})
}
