package main

import "github.com/sprucehealth/graphql"

func newClientMutationIDInputField() *graphql.InputObjectFieldConfig {
	return &graphql.InputObjectFieldConfig{
		Description: "This field is for Relay compatibility and should not be used otherwise.",
		Type:        graphql.String,
	}
}

func newClientMutationIDOutputField() *graphql.Field {
	return &graphql.Field{
		Description: "This field is for Relay compatibility and should not be used otherwise.",
		Type:        graphql.String,
	}
}

func newUUIDInputField() *graphql.InputObjectFieldConfig {
	return &graphql.InputObjectFieldConfig{
		Description: "This field, if provided, makes the mutation idempotent.",
		Type:        graphql.String,
	}
}

var contactInfoInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ContactInfoInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"id":    &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"type":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(contactEnumType)},
			"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"label": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var serializedContactInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SerializedContactInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"platform": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(platformEnumType)},
			"contact":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var entityInfoInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "EntityInfoInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"firstName":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"middleInitial":      &graphql.InputObjectFieldConfig{Type: graphql.String},
			"lastName":           &graphql.InputObjectFieldConfig{Type: graphql.String},
			"groupName":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"shortTitle":         &graphql.InputObjectFieldConfig{Type: graphql.String},
			"longTitle":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"note":               &graphql.InputObjectFieldConfig{Type: graphql.String},
			"contactInfos":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(contactInfoInputType))},
			"serializedContacts": &graphql.InputObjectFieldConfig{Type: graphql.NewList(serializedContactInputType)},
		},
	},
)

/// registerDeviceForPush

type registerDeviceForPushOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var registerDeviceForPushInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "RegisterDeviceForPushInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"deviceToken":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var registerDeviceForPushErrorCodeEnum = graphql.String

var registerDeviceForPushOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RegisterDeviceForPushPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: registerDeviceForPushErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*registerDeviceForPushOutput)
			return ok
		},
	},
)

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"addContactInfos":                     addContactInfosMutation,
		"associateAttribution":                associateAttributionMutation,
		"associateInvite":                     associateInviteMutation,
		"authenticate":                        authenticateMutation,
		"authenticateWithCode":                authenticateWithCodeMutation,
		"callEntity":                          callEntityMutation,
		"checkPasswordResetToken":             checkPasswordResetTokenMutation,
		"checkVerificationCode":               checkVerificationCodeMutation,
		"createAccount":                       createAccountMutation,
		"createCarePlan":                      createCarePlanMutation,
		"createPatientAccount":                createPatientAccountMutation,
		"createProfile":                       createProfileMutation,
		"createProviderAccount":               createProviderAccountMutation,
		"createTeamThread":                    createTeamThreadMutation,
		"createThread":                        createThreadMutation,
		"createVideoCall":                     createVideoCallMutation,
		"deleteContactInfos":                  deleteContactInfosMutation,
		"deleteThread":                        deleteThreadMutation,
		"inviteColleagues":                    inviteColleaguesMutation,
		"invitePatients":                      invitePatientsMutation,
		"leaveThread":                         leaveThreadMutation,
		"markThreadAsRead":                    markThreadAsReadMutation,
		"markThreadsAsRead":                   markThreadsAsReadMutation,
		"modifySetting":                       modifySettingMutation,
		"passwordReset":                       passwordResetMutation,
		"postEvent":                           postEventMutation,
		"postMessage":                         postMessageMutation,
		"provisionEmail":                      provisionEmailMutation,
		"provisionPhoneNumber":                provisionPhoneNumberMutation,
		"registerDeviceForPush":               registerDeviceForPushMutation,
		"requestPasswordReset":                requestPasswordResetMutation,
		"submitVisit":                         submitVisitMutation,
		"submitVisitAnswers":                  submitVisitAnswersMutation,
		"triageVisit":                         triageVisitMutation,
		"unauthenticate":                      unauthenticateMutation,
		"updateCall":                          updateCallMutation,
		"updateContactInfos":                  updateContactInfosMutation,
		"updateEntity":                        updateEntityMutation,
		"updateProfile":                       updateProfileMutation,
		"updateThread":                        updateThreadMutation,
		"verifyEmail":                         verifyEmailMutation,
		"verifyEmailForAccountCreation":       verifyEmailForAccountCreationMutation,
		"verifyPhoneNumber":                   verifyPhoneNumberMutation,
		"verifyPhoneNumberForAccountCreation": verifyPhoneNumberForAccountCreationMutation,
		"verifyPhoneNumberForPasswordReset":   verifyPhoneNumberForPasswordResetMutation,
	},
})
