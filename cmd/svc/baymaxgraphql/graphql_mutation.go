package main

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

// createSavedThreadQuery

const (
	createSavedThreadQueryResultSuccess             = "SUCCESS"
	createSavedThreadQueryResultInvalidOrganization = "INVALID_ORGANIZATION"
	createSavedThreadQueryResultInvalidQuery        = "INVALID_QUERY"
	createSavedThreadQueryResultNotAllowed          = "NOT_ALLOWED"
)

var createSavedThreadQueryResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CreatedSavedThreadQueryResult",
		Description: "Result of createSavedThreadQuery",
		Values: graphql.EnumValueConfigMap{
			createSavedThreadQueryResultSuccess: &graphql.EnumValueConfig{
				Value:       createSavedThreadQueryResultSuccess,
				Description: "Success",
			},
			createSavedThreadQueryResultInvalidQuery: &graphql.EnumValueConfig{
				Value:       createSavedThreadQueryResultInvalidQuery,
				Description: "The provided query is invalid",
			},
			createSavedThreadQueryResultInvalidOrganization: &graphql.EnumValueConfig{
				Value:       createSavedThreadQueryResultInvalidOrganization,
				Description: "The provided organization ID is invalid",
			},
			createSavedThreadQueryResultNotAllowed: &graphql.EnumValueConfig{
				Value:       createSavedThreadQueryResultNotAllowed,
				Description: "The account does not have access to the organization",
			},
		},
	},
)

var createSavedThreadQueryInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateSavedThreadQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			// TODO: query
		},
	},
)

type createSavedThreadQueryOutput struct {
	ClientMutationID   string            `json:"clientMutationId,omitempty"`
	Result             string            `json:"result"`
	SavedThreadQueryID string            `json:"savedThreadQueryID,omitempty"`
	SavedThreadQuery   *savedThreadQuery `json:"savedThreadQuery,omitempty"`
}

var createSavedThreadQueryOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateSavedThreadQueryPayload",
		Fields: graphql.Fields{
			"clientMutationId":   newClientmutationIDOutputField(),
			"savedThreadQueryID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"savedThreadQuery":   &graphql.Field{Type: graphql.NewNonNull(savedThreadQueryType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createSavedThreadQueryOutput)
			return ok
		},
	},
)

/// registerDeviceForPush

type registerDeviceForPushOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
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

var registerDeviceForPushOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RegisterDeviceForPushPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
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

var markThreadAsReadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MarkThreadAsReadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*markThreadAsReadOutput)
			return ok
		},
	},
)

/// sendTestNotification

type sendTestNotificationOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
}

var sendTestNotificationInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SendTestNotificationInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"message":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var sendTestNotificationOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SendTestNotificationPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*sendTestNotificationOutput)
			return ok
		},
	},
)

// deleteThread

type deleteThreadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
}

var deleteThreadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeleteThreadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

var deleteThreadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeleteThreadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deleteThreadOutput)
			return ok
		},
	},
)

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"createAccount": createAccountMutation,
		"createSavedThreadQuery": &graphql.Field{
			Type: graphql.NewNonNull(createSavedThreadQueryOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createSavedThreadQueryInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := accountFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				orgID := input["organizationID"].(string)
				// TODO: validate organization exists

				ent, err := svc.entityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return &createSavedThreadQueryOutput{
						ClientMutationID: mutationID,
						Result:           createSavedThreadQueryResultNotAllowed,
					}, nil
				}

				res, err := svc.threading.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
					OrganizationID: orgID,
					EntityID:       ent.ID,
					// TODO: query
				})
				if err != nil {
					switch grpc.Code(err) {
					case codes.InvalidArgument:
						return nil, err
					}
					return nil, internalError(err)
				}
				sq, err := transformSavedQueryToResponse(res.SavedQuery)
				if err != nil {
					return nil, internalError(err)
				}
				return &createSavedThreadQueryOutput{
					ClientMutationID:   mutationID,
					Result:             createSavedThreadQueryResultSuccess,
					SavedThreadQueryID: res.SavedQuery.ID,
					SavedThreadQuery:   sq,
				}, nil
			},
		},
		"postMessage": postMessageMutation,
		"callEntity":  callEntityMutation,
		"registerDeviceForPush": &graphql.Field{
			Type: graphql.NewNonNull(registerDeviceForPushOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(registerDeviceForPushInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := accountFromContext(ctx)
				sh := spruceHeadersFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}
				golog.Infof("Registering Device For Push: Account:%s Device:%+v", acc.ID, sh)
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
				}, nil
			},
		},
		"markThreadAsRead": &graphql.Field{
			Type: graphql.NewNonNull(markThreadAsReadOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(markThreadAsReadInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := accountFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				threadID, _ := input["threadID"].(string)
				orgID, _ := input["organizationID"].(string)
				ent, err := svc.entityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return nil, errors.New("mark thread as read failed")
				}
				// TODO: Authorize

				_, err = svc.threading.MarkThreadAsRead(ctx, &threading.MarkThreadAsReadRequest{
					ThreadID: threadID,
					EntityID: ent.ID,
				})
				if err != nil {
					return nil, internalError(err)
				}

				return &markThreadAsReadOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
		"sendTestNotification": &graphql.Field{
			Type: graphql.NewNonNull(sendTestNotificationOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(sendTestNotificationInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := accountFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				threadID, _ := input["threadID"].(string)
				orgID, _ := input["organizationID"].(string)
				message, _ := input["message"].(string)
				ent, err := svc.entityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return nil, errors.New("send test notification failed")
				} else if ent == nil {
					return nil, fmt.Errorf("entity not found for token and orgID %s", orgID)
				}

				if err := svc.notification.SendNotification(&notification.Notification{
					ShortMessage:     message,
					ThreadID:         threadID,
					OrganizationID:   orgID,
					EntitiesToNotify: []string{ent.ID},
				}); err != nil {
					return nil, internalError(err)
				}

				return &sendTestNotificationOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
		"addContactInfos":                     addContactInfosMutation,
		"associateAttribution":                associateAttributionMutation,
		"authenticate":                        authenticateMutation,
		"authenticateWithCode":                authenticateWithCodeMutation,
		"checkPasswordResetToken":             checkPasswordResetTokenMutation,
		"checkVerificationCode":               checkVerificationCodeMutation,
		"createThread":                        createThreadMutation,
		"deleteContactInfos":                  deleteContactInfosMutation,
		"deleteThread":                        deleteThreadMutation,
		"inviteColleagues":                    inviteColleaguesMutation,
		"modifySetting":                       modifySettingMutation,
		"passwordReset":                       passwordResetMutation,
		"provisionEmail":                      provisionEmailMutation,
		"provisionPhoneNumber":                provisionPhoneNumberMutation,
		"requestPasswordReset":                requestPasswordResetMutation,
		"unauthenticate":                      unauthenticateMutation,
		"updateContactInfos":                  updateContactInfosMutation,
		"updateEntity":                        updateEntityMutation,
		"verifyPhoneNumber":                   verifyPhoneNumberMutation,
		"verifyPhoneNumberForAccountCreation": verifyPhoneNumberForAccountCreationMutation,
		"verifyPhoneNumberForPasswordReset":   verifyPhoneNumberForPasswordResetMutation,
	},
})
