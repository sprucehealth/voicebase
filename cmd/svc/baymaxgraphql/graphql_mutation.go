package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/graphql"
)

func newClientMutationIDInputField() *graphql.InputObjectFieldConfig {
	return &graphql.InputObjectFieldConfig{
		Description: "This field is for Relay compatibility and should not be used otherwise.",
		Type:        graphql.String,
	}
}

func newClientmutationIDOutputField() *graphql.Field {
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
			"clientMutationId": newClientmutationIDOutputField(),
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

/// markThreadAsRead

type markThreadAsReadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var markThreadAsReadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MarkThreadAsReadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var markThreadAsReadErrorCodeEnum = graphql.String

var markThreadAsReadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MarkThreadAsReadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: markThreadAsReadErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*markThreadAsReadOutput)
			return ok
		},
	},
)

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"callEntity":            callEntityMutation,
		"createAccount":         createAccountMutation,
		"createProviderAccount": createProviderAccountMutation,
		"createPatientAccount":  createPatientAccountMutation,
		"postMessage":           postMessageMutation,
		"registerDeviceForPush": &graphql.Field{
			Type: graphql.NewNonNull(registerDeviceForPushOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(registerDeviceForPushInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)
				sh := gqlctx.SpruceHeaders(ctx)
				if acc == nil {
					return nil, errors.ErrNotAuthenticated(ctx)
				}
				golog.Debugf("Registering Device For Push: Account:%s Device:%+v", acc.ID, sh)
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				deviceToken, _ := input["deviceToken"].(string)
				if err := svc.notification.RegisterDeviceForPush(&notification.DeviceRegistrationInfo{
					ExternalGroupID: acc.ID,
					DeviceToken:     deviceToken,
					Platform:        sh.Platform.String(),
					PlatformVersion: sh.PlatformVersion,
					AppVersion:      sh.AppVersion.String(),
					Device:          sh.Device,
					DeviceModel:     sh.DeviceModel,
					DeviceID:        sh.DeviceID,
				}); err != nil {
					golog.Errorf(err.Error())
					return nil, errors.New("device registration failed")
				}

				return &registerDeviceForPushOutput{
					ClientMutationID: mutationID,
					Success:          true,
				}, nil
			},
		},
		"markThreadAsRead": &graphql.Field{
			Type: graphql.NewNonNull(markThreadAsReadOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(markThreadAsReadInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)
				if acc == nil {
					return nil, errors.ErrNotAuthenticated(ctx)
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				threadID, _ := input["threadID"].(string)
				orgID, _ := input["organizationID"].(string)
				ent, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				if ent == nil || ent.Type != directory.EntityType_INTERNAL {
					return nil, errors.New("not authorized")
				}

				if err = ram.MarkThreadAsRead(ctx, threadID, ent.ID); err != nil {
					return nil, err
				}

				return &markThreadAsReadOutput{
					ClientMutationID: mutationID,
					Success:          true,
				}, nil
			},
		},
		"addContactInfos":                     addContactInfosMutation,
		"associateAttribution":                associateAttributionMutation,
		"associateInvite":                     associateInviteMutation,
		"authenticate":                        authenticateMutation,
		"authenticateWithCode":                authenticateWithCodeMutation,
		"checkPasswordResetToken":             checkPasswordResetTokenMutation,
		"checkVerificationCode":               checkVerificationCodeMutation,
		"createTeamThread":                    createTeamThreadMutation,
		"createThread":                        createThreadMutation,
		"deleteContactInfos":                  deleteContactInfosMutation,
		"deleteThread":                        deleteThreadMutation,
		"inviteColleagues":                    inviteColleaguesMutation,
		"leaveThread":                         leaveThreadMutation,
		"modifySetting":                       modifySettingMutation,
		"passwordReset":                       passwordResetMutation,
		"postEvent":                           postEventMutation,
		"provisionEmail":                      provisionEmailMutation,
		"provisionPhoneNumber":                provisionPhoneNumberMutation,
		"requestPasswordReset":                requestPasswordResetMutation,
		"unauthenticate":                      unauthenticateMutation,
		"updateContactInfos":                  updateContactInfosMutation,
		"updateEntity":                        updateEntityMutation,
		"updateThread":                        updateThreadMutation,
		"verifyPhoneNumber":                   verifyPhoneNumberMutation,
		"verifyPhoneNumberForAccountCreation": verifyPhoneNumberForAccountCreationMutation,
		"verifyPhoneNumberForPasswordReset":   verifyPhoneNumberForPasswordResetMutation,
	},
})
