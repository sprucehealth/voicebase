package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// authenticate

const (
	authenticateResultSuccess         = "SUCCESS"
	authenticateResultInvalidEmail    = "INVALID_EMAIL"
	authenticateResultInvalidPassword = "INVALID_PASSWORD"
)

type authenticateOutput struct {
	ClientMutationID string   `json:"clientMutationId"`
	Result           string   `json:"result"`
	Token            string   `json:"token,omitempty"`
	Account          *account `json:"account,omitempty"`
}

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

var authenticateResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "AuthenticateResult",
		Description: "Result of authenticate mutation",
		Values: graphql.EnumValueConfigMap{
			authenticateResultSuccess: &graphql.EnumValueConfig{
				Value:       authenticateResultSuccess,
				Description: "Success",
			},
			authenticateResultInvalidEmail: &graphql.EnumValueConfig{
				Value:       authenticateResultInvalidEmail,
				Description: "Email not found",
			},
			authenticateResultInvalidPassword: &graphql.EnumValueConfig{
				Value:       authenticateResultInvalidPassword,
				Description: "Password doesn't match",
			},
		},
	},
)

var authenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var authenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AuthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(authenticateResultType)},
			"token":            &graphql.Field{Type: graphql.String},
			"account":          &graphql.Field{Type: accountType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*authenticateOutput)
			return ok
		},
	},
)

/// unauthenticate

type unauthenticateOutput struct {
	ClientMutationID string `json:"clientMutationId"`
}

var unauthenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UnauthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var unauthenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UnauthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*unauthenticateOutput)
			return ok
		},
	},
)

/// createAccount

type createAccountOutput struct {
	ClientMutationID string   `json:"clientMutationId"`
	Token            string   `json:"token,omitempty"`
	Account          *account `json:"account,omitempty"`
}

var createAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"phoneNumber":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"firstName":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"lastName":         &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var createAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateAccountPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"token":            &graphql.Field{Type: graphql.String},
			"account":          &graphql.Field{Type: accountType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createAccountOutput)
			return ok
		},
	},
)

// message

var messageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"uuid":         &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"text":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"destinations": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(channelEnumType))},
			"internal":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

// postMessage

type postMessageOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	ItemEdge         *Edge   `json:"itemEdge"`
	Thread           *thread `json:"thread"`
}

var postMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "PostMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"msg":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(messageInputType)},
		},
	},
)

var postMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PostMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"itemEdge":         &graphql.Field{Type: graphql.NewNonNull(threadItemConnectionType.EdgeType)},
			"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*postMessageOutput)
			return ok
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
	ClientMutationID   string            `json:"clientMutationId"`
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

// callEntity

const (
	callEntityTypeConnectParties    = "CONNECT_PARTIES"
	callEntityTypeReturnPhoneNumber = "RETURN_PHONE_NUMBER"
)

const (
	callEntityResultSuccess            = "SUCCESS"
	callEntityResultEntityNotFound     = "ENTITY_NOT_FOUND"
	callEntityResultEntityHasNoContact = "ENTITY_HAS_NO_CONTACT"
)

type callEntityOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Result           string `json:"result"`
	PhoneNumber      string `json:"phoneNumber,omitempty"`
}

var callEntityTypeEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CallEntityType",
		Description: "How to initiate the call",
		Values: graphql.EnumValueConfigMap{
			callEntityTypeConnectParties: &graphql.EnumValueConfig{
				Value:       callEntityTypeConnectParties,
				Description: "Connect parties by calling both numbers",
			},
			callEntityTypeReturnPhoneNumber: &graphql.EnumValueConfig{
				Value:       callEntityTypeReturnPhoneNumber,
				Description: "Return a phone number to call",
			},
		},
	},
)

var callEntityResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CallEntityResult",
		Description: "Result of callEntity",
		Values: graphql.EnumValueConfigMap{
			callEntityResultSuccess: &graphql.EnumValueConfig{
				Value:       callEntityResultSuccess,
				Description: "Success",
			},
			callEntityResultEntityNotFound: &graphql.EnumValueConfig{
				Value:       callEntityResultEntityNotFound,
				Description: "The requested entity does not exist",
			},
			callEntityResultEntityHasNoContact: &graphql.EnumValueConfig{
				Value:       callEntityResultEntityHasNoContact,
				Description: "An entity does not have a viable contact",
			},
		},
	},
)

var callEntityInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CallEntityInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"id":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"type":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(callEntityTypeEnumType)},
		},
	},
)

var callEntityOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CallEntityPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(callEntityResultType)},
			"phoneNumber": &graphql.Field{
				Type:        graphql.String,
				Description: "The phone number to use to contact the entity.",
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*callEntityOutput)
			return ok
		},
	},
)

/// registerDeviceForPush

type registerDeviceForPushOutput struct {
	ClientMutationID string `json:"clientMutationId"`
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
	ClientMutationID string `json:"clientMutationId"`
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
	ClientMutationID string `json:"clientMutationId"`
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

/// updateEntity

type updateEntityOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var updateEntityInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateEntityInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"firstName":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"middleInitial":    &graphql.InputObjectFieldConfig{Type: graphql.String},
			"lastName":         &graphql.InputObjectFieldConfig{Type: graphql.String},
			"groupName":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.String},
			"note":             &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var updateEntityOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateEntityPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateEntityOutput)
			return ok
		},
	},
)

/// addContacts

var unprovisionedContactInfoType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UnprovisionedContactInfo",
		Fields: graphql.InputObjectConfigFieldMap{
			"id":    &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"type":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(contactEnumType)},
			"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"label": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

type addContactsOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var addContactsInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AddContactsInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"contactInfos":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(unprovisionedContactInfoType)},
		},
	},
)

var addContactsOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AddContactsPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*addContactsOutput)
			return ok
		},
	},
)

/// updateContacts

type updateContactsOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var updateContactsInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateContactsInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"contactInfos":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(unprovisionedContactInfoType)},
		},
	},
)

var updateContactsOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateContactsPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateContactsOutput)
			return ok
		},
	},
)

/// deleteContacts

type deleteContactsOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var deleteContactsInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeleteContactsInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"contactIDs":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.String)},
		},
	},
)

var deleteContactsOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeleteContactsPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deleteContactsOutput)
			return ok
		},
	},
)

// deleteThread

type deleteThreadOutput struct {
	ClientMutationID string `json:"clientMutationId"`
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
		"authenticate": &graphql.Field{
			Type: graphql.NewNonNull(authenticateOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				email := input["email"].(string)
				if !validate.Email(email) {
					return nil, errors.New("invalid email")
				}
				password := input["password"].(string)
				res, err := svc.auth.AuthenticateLogin(ctx, &auth.AuthenticateLoginRequest{
					Email:    email,
					Password: password,
				})
				if err != nil {
					switch grpc.Code(err) {
					case auth.EmailNotFound:
						return &authenticateOutput{
							ClientMutationID: mutationID,
							Result:           authenticateResultInvalidEmail,
						}, nil
					case auth.BadPassword:
						return &authenticateOutput{
							ClientMutationID: mutationID,
							Result:           authenticateResultInvalidPassword,
						}, nil
					default:
						return nil, internalError(err)
					}
				}
				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("auth_token", res.Token.Value)
				result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))
				acc := &account{
					ID: res.Account.ID,
				}
				// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
				// that likely won't change, but this still isn't a great way to update the context.
				p.Context = ctxWithAccount(ctx, acc)
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Result:           authenticateResultSuccess,
					Token:            res.Token.Value,
					Account:          acc,
				}, nil
			},
		},
		"unauthenticate": &graphql.Field{
			Type: graphql.NewNonNull(unauthenticateOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: unauthenticateInputType},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				// TODO: get token from cookie if not provided in args
				token, ok := input["token"].(string)
				if !ok {
					return nil, internalError(errors.New("TODO: unauthenticate using cookie is not yet implemented"))
				}
				_, err := svc.auth.Unauthenticate(ctx, &auth.UnauthenticateRequest{Token: token})
				if err != nil {
					return nil, internalError(err)
				}
				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("unauthenticated", true)
				return &unauthenticateOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
		"createAccount": &graphql.Field{
			Type: graphql.NewNonNull(createAccountOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createAccountInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := p.Context
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				req := &auth.CreateAccountRequest{
					Email:    input["email"].(string),
					Password: input["password"].(string),
				}
				if !validate.Email(req.Email) {
					return nil, errors.New("invalid email")
				}
				if s, ok := input["firstName"].(string); ok {
					req.FirstName = s
				}
				if s, ok := input["lastName"].(string); ok {
					req.LastName = s
				}
				if s, ok := input["phoneNumber"].(string); ok {
					req.PhoneNumber = s
				}
				res, err := svc.auth.CreateAccount(ctx, req)
				if err != nil {
					switch grpc.Code(err) {
					case auth.DuplicateEmail:
						return nil, errors.New("account with email exists")
					case auth.InvalidEmail:
						return nil, errors.New("invalid email")
					case auth.InvalidPhoneNumber:
						return nil, errors.New("invalid phone number")
					}
					return nil, internalError(err)
				}
				accountID := res.Account.ID

				var orgEntityID string
				var accEntityID string
				{
					// Create organization
					res, err := svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
						EntityInfo: &directory.EntityInfo{
							GroupName:   "Test Organization",
							DisplayName: "Test Organization",
						},
						Type: directory.EntityType_ORGANIZATION,
					})
					if err != nil {
						return nil, internalError(err)
					}
					orgEntityID = res.Entity.ID

					// Create entity
					res, err = svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
						EntityInfo: &directory.EntityInfo{
							FirstName:   req.FirstName,
							LastName:    req.LastName,
							DisplayName: req.FirstName + " " + req.LastName,
						},
						Type:                      directory.EntityType_INTERNAL,
						ExternalID:                accountID,
						InitialMembershipEntityID: orgEntityID,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       req.PhoneNumber,
								Provisioned: false,
							},
						},
					})
					if err != nil {
						return nil, internalError(err)
					}
					accEntityID = res.Entity.ID
				}

				// Create a default saved query
				_, err = svc.threading.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
					OrganizationID: orgEntityID,
					EntityID:       accEntityID,
					// TODO: query
				})
				if err != nil {
					return nil, internalError(err)
				}

				pres, err := svc.exComms.ProvisionPhoneNumber(ctx, &excomms.ProvisionPhoneNumberRequest{
					ProvisionFor: orgEntityID,
					Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
						AreaCode: "801",
					},
				})
				if err != nil {
					return nil, internalError(err)
				}
				_, err = svc.directory.CreateContact(ctx, &directory.CreateContactRequest{
					Contact: &directory.Contact{
						ContactType: directory.ContactType_PHONE,
						Value:       pres.PhoneNumber,
						Provisioned: true,
					},
					EntityID: orgEntityID,
				})
				if err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("auth_token", res.Token.Value)
				result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

				acc := &account{
					ID: res.Account.ID,
				}
				// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
				// that likely won't change, but this still isn't a great way to update the context.
				p.Context = ctxWithAccount(ctx, acc)
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Token:            res.Token.Value,
					Account:          acc,
				}, nil
			},
		},
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
		"postMessage": &graphql.Field{
			Type: graphql.NewNonNull(postMessageOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(postMessageInputType)},
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
				threadID := input["threadID"].(string)
				msg := input["msg"].(map[string]interface{})

				tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
					ThreadID: threadID,
				})
				if err != nil {
					switch grpc.Code(err) {
					case codes.NotFound:
						return nil, errors.New("thread does not exist")
					}
					return nil, internalError(err)
				}
				thr := tres.Thread

				ent, err := svc.entityForAccountID(ctx, thr.OrganizationID, acc.ID)
				if err != nil {
					return nil, internalError(err)
				}
				if ent == nil {
					return nil, internalError(fmt.Errorf("entity for org %s and account %s not found", thr.OrganizationID, acc.ID))
				}

				var title bml.BML
				fromName := ent.Info.DisplayName
				if fromName == "" && len(ent.Contacts) != 0 {
					fromName = ent.Contacts[0].Value
				}
				title = append(title, &bml.Ref{ID: ent.ID, Type: bml.EntityRef, Text: fromName})

				text := msg["text"].(string)

				// Parse text and render as plain text so we can build a summary.
				textBML, err := bml.Parse(text)
				if e, ok := err.(bml.ErrParseFailure); ok {
					return nil, fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
				} else if err != nil {
					return nil, errors.New("text is not valid markup")
				}
				plainText, err := textBML.PlainText()
				if err != nil {
					// Shouldn't fail here since the parsing should have done validation
					return nil, internalError(err)
				}
				summary := fmt.Sprintf("%s: %s", fromName, plainText)

				req := &threading.PostMessageRequest{
					ThreadID:     threadID,
					Text:         text,
					Internal:     msg["internal"].(bool),
					FromEntityID: ent.ID,
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
						ID:      ent.ID,
					},
					Summary: summary,
				}

				// For a message to be considered by sending externally it needs to not be marked as internal,
				// sent by someone who is internal, and there needs to be a primary entity on the thread.
				isExternal := !req.Internal && thr.PrimaryEntityID != "" && ent.Type == directory.EntityType_INTERNAL
				var extEntity *directory.Entity
				if isExternal {
					dests, _ := msg["destinations"].([]interface{})
					// TODO: if no destinations specified then query routing service for default route
					if len(dests) != 0 {
						res, err := svc.directory.LookupEntities(ctx,
							&directory.LookupEntitiesRequest{
								LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
								LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
									EntityID: thr.PrimaryEntityID,
								},
								RequestedInformation: &directory.RequestedInformation{
									Depth: 0,
									EntityInformation: []directory.EntityInformation{
										directory.EntityInformation_CONTACTS,
									},
								},
							})
						if err != nil {
							return nil, internalError(err)
						}
						if len(res.Entities) > 1 {
							golog.Errorf("lookup entities returned more than 1 result for entity ID %s", thr.PrimaryEntityID)
						}
						if len(res.Entities) > 0 && res.Entities[0].Type == directory.EntityType_EXTERNAL {
							extEntity = res.Entities[0]
							updatedTitle := false // TODO: for now only putting one destination in the message title
							for _, d := range dests {
								channel := d.(string)
								var ct directory.ContactType
								var ec threading.Endpoint_Channel
								var action string
								switch channel {
								case endpointChannelEmail:
									ct = directory.ContactType_EMAIL
									ec = threading.Endpoint_EMAIL
									action = "emailed"
								case endpointChannelSMS:
									ct = directory.ContactType_PHONE
									ec = threading.Endpoint_SMS
									action = "texted"
								case endpointChannelApp:
									// App delivery selection is different since it doesn't need
									// contacts and no reason to update title. Not sure we even
									// need the APP destination type, but since it's there will
									// record that it was sent.
									req.Destinations = append(req.Destinations, &threading.Endpoint{
										Channel: threading.Endpoint_APP,
										ID:      thr.PrimaryEntityID,
									})
									continue
								default:
									return nil, errors.New("unsupported destination type " + channel)
								}
								var e *threading.Endpoint
								for _, c := range extEntity.Contacts {
									if c.ContactType == ct {
										e = &threading.Endpoint{
											Channel: ec,
											ID:      c.Value,
										}
										break
									}
								}
								if e == nil {
									return nil, errors.New("no contact info for destination channel " + channel)
								}
								req.Destinations = append(req.Destinations, e)
								if !updatedTitle {
									name := extEntity.Info.DisplayName
									if name == "" {
										name = e.ID
									}
									title = append(title, " "+action+" ", &bml.Ref{ID: extEntity.ID, Type: bml.EntityRef, Text: name})
									updatedTitle = true
								}
							}
						}
					}
				}
				if uuid, ok := msg["uuid"].(string); ok {
					req.UUID = uuid
				}

				titleStr, err := title.Format()
				if err != nil {
					return nil, internalError(fmt.Errorf("invalid title BML %+v: %s", title, err))
				}
				req.Title = titleStr

				pmres, err := svc.threading.PostMessage(ctx, req)
				if err != nil {
					return nil, internalError(err)
				}

				it, err := transformThreadItemToResponse(pmres.Item, req.UUID, acc.ID, svc.mediaSigner)
				if err != nil {
					return nil, internalError(fmt.Errorf("failed to transform thread item: %s", err))
				}
				th, err := transformThreadToResponse(pmres.Thread)
				if err != nil {
					return nil, internalError(fmt.Errorf("failed to transform thread: %s", err))
				}
				if extEntity != nil {
					th.Title = threadTitleForEntity(extEntity)
				} else if err := svc.hydrateThreadTitles(ctx, []*thread{th}); err != nil {
					return nil, internalError(err)
				}
				return &postMessageOutput{
					ClientMutationID: mutationID,
					ItemEdge:         &Edge{Node: it, Cursor: ConnectionCursor(pmres.Item.ID)},
					Thread:           th,
				}, nil
			},
		},
		"callEntity": &graphql.Field{
			Type: graphql.NewNonNull(callEntityOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(callEntityInputType)},
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
				entityID := input["id"].(string)
				calleeEnt, err := svc.entity(ctx, entityID)
				if err != nil {
					return nil, internalError(err)
				}
				if calleeEnt == nil || calleeEnt.Type != directory.EntityType_EXTERNAL {
					return &callEntityOutput{
						ClientMutationID: mutationID,
						Result:           callEntityResultEntityNotFound,
					}, nil
				}

				var org *directory.Entity
				for _, em := range calleeEnt.Memberships {
					if em.Type == directory.EntityType_ORGANIZATION {
						org = em
						break
					}
				}
				if org == nil {
					return &callEntityOutput{
						ClientMutationID: mutationID,
						Result:           callEntityResultEntityNotFound,
					}, nil
				}

				callerEnt, err := svc.entityForAccountID(ctx, org.ID, acc.ID)
				if err != nil {
					return nil, internalError(err)
				}
				if callerEnt == nil {
					return &callEntityOutput{
						ClientMutationID: mutationID,
						Result:           callEntityResultEntityNotFound,
					}, nil
				}

				var fromContact *directory.Contact
				for _, c := range callerEnt.Contacts {
					if c.ContactType == directory.ContactType_PHONE && !c.Provisioned {
						fromContact = c
					}
				}
				if fromContact == nil {
					return &callEntityOutput{
						ClientMutationID: mutationID,
						Result:           callEntityResultEntityHasNoContact,
					}, nil
				}

				var toContact *directory.Contact
				for _, c := range calleeEnt.Contacts {
					if c.ContactType == directory.ContactType_PHONE && !c.Provisioned {
						toContact = c
					}
				}
				if toContact == nil {
					return &callEntityOutput{
						ClientMutationID: mutationID,
						Result:           callEntityResultEntityHasNoContact,
					}, nil
				}

				ireq := &excomms.InitiatePhoneCallRequest{
					FromPhoneNumber: fromContact.Value,
					ToPhoneNumber:   toContact.Value,
					OrganizationID:  org.ID,
				}
				switch input["type"].(string) {
				case callEntityTypeConnectParties:
					ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_CONNECT_PARTIES
				case callEntityTypeReturnPhoneNumber:
					ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER
				}
				ires, err := svc.exComms.InitiatePhoneCall(ctx, ireq)
				if err != nil {
					return nil, internalError(err)
				}

				return &callEntityOutput{
					ClientMutationID: mutationID,
					Result:           callEntityResultSuccess,
					PhoneNumber:      ires.PhoneNumber,
				}, nil
			},
		},
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

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("registerDeviceForPush", true)
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

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("markThreadAsRead", true)
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
				}

				if err := svc.notification.SendNotification(&notification.Notification{
					ShortMessage:     message,
					ThreadID:         threadID,
					OrganizationID:   orgID,
					EntitiesToNotify: []string{ent.ID},
				}); err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("sendTestNotification", true)
				return &markThreadAsReadOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
		"provisionEmail": provisionEmailField,
		"updateEntity": &graphql.Field{
			Type: graphql.NewNonNull(updateEntityOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateEntityInputType)},
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
				ei, _ := input["info"]
				entID, _ := input["entityID"].(string)
				entityInfo, err := entityInfoFromFieldList(ei)
				if err != nil {
					return nil, internalError(err)
				}

				resp, err := svc.directory.UpdateEntity(ctx, &directory.UpdateEntityRequest{
					EntityID:   entID,
					EntityInfo: entityInfo,
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
				})
				if err != nil {
					return nil, internalError(err)
				}

				e, err := transformEntityToResponse(resp.Entity)
				if err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("updateEntity", true)
				return &updateEntityOutput{
					ClientMutationID: mutationID,
					Entity:           e,
				}, nil
			},
		},
		"addContacts": &graphql.Field{
			Type: graphql.NewNonNull(addContactsOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(addContactsInputType)},
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
				contactInfos, _ := input["contactInfos"].([]interface{})
				entID, _ := input["entityID"].(string)

				contacts, err := contactsFromFieldList(contactInfos)
				if err != nil {
					return nil, internalError(err)
				}

				resp, err := svc.directory.CreateContacts(ctx, &directory.CreateContactsRequest{
					EntityID: entID,
					Contacts: contacts,
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
				})
				if err != nil {
					return nil, internalError(err)
				}

				e, err := transformEntityToResponse(resp.Entity)
				if err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("addContacts", true)
				return &addContactsOutput{
					ClientMutationID: mutationID,
					Entity:           e,
				}, nil
			},
		},
		"updateContacts": &graphql.Field{
			Type: graphql.NewNonNull(updateContactsOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateContactsInputType)},
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
				contactInfos, _ := input["contactInfos"].([]interface{})
				entID, _ := input["entityID"].(string)

				contacts, err := contactsFromFieldList(contactInfos)
				if err != nil {
					return nil, internalError(err)
				}

				resp, err := svc.directory.UpdateContacts(ctx, &directory.UpdateContactsRequest{
					EntityID: entID,
					Contacts: contacts,
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
				})
				if err != nil {
					return nil, internalError(err)
				}

				e, err := transformEntityToResponse(resp.Entity)
				if err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("updateContacts", true)
				return &updateContactsOutput{
					ClientMutationID: mutationID,
					Entity:           e,
				}, nil
			},
		},
		"deleteContacts": &graphql.Field{
			Type: graphql.NewNonNull(deleteContactsOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteContactsInputType)},
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
				contactIDs, _ := input["contactIDs"].([]interface{})
				entID, _ := input["entityID"].(string)

				sContacts := make([]string, len(contactIDs))
				for i, ci := range contactIDs {
					sContacts[i] = ci.(string)
				}

				resp, err := svc.directory.DeleteContacts(ctx, &directory.DeleteContactsRequest{
					EntityID:         entID,
					EntityContactIDs: sContacts,
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
				})
				if err != nil {
					return nil, internalError(err)
				}

				e, err := transformEntityToResponse(resp.Entity)
				if err != nil {
					return nil, internalError(err)
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("deleteContacts", true)
				return &deleteContactsOutput{
					ClientMutationID: mutationID,
					Entity:           e,
				}, nil
			},
		},
		"deleteThread": &graphql.Field{
			Type: graphql.NewNonNull(deleteThreadOutputType),
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteThreadInputType)},
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
				threadID := input["threadID"].(string)

				// Make sure thread exists (wasn't deleted) and get organization ID to be able to fetch entity for the account
				tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
					ThreadID: threadID,
				})
				if err != nil {
					switch grpc.Code(err) {
					case codes.NotFound:
						return nil, errors.New("thread not found")
					}
					return nil, internalError(err)
				}

				ent, err := svc.entityForAccountID(ctx, tres.Thread.OrganizationID, acc.ID)
				if err != nil {
					return nil, internalError(err)
				}
				if ent == nil {
					return nil, errors.New("not a member of the organization")
				}

				if _, err := svc.threading.DeleteThread(ctx, &threading.DeleteThreadRequest{
					ThreadID:      threadID,
					ActorEntityID: ent.ID,
				}); err != nil {
					return nil, internalError(err)
				}

				return &deleteThreadOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
	},
})

func contactsFromFieldList(cis []interface{}) ([]*directory.Contact, error) {
	contacts := make([]*directory.Contact, len(cis))
	for i, ci := range cis {
		mci, ok := ci.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Unable to parse input contact data: %+v", ci)
		}

		id, _ := mci["id"].(string)
		t, _ := mci["type"].(string)
		v, _ := mci["value"].(string)
		l, _ := mci["label"].(string)

		ct, ok := directory.ContactType_value[t]
		if !ok {
			return nil, fmt.Errorf("Unknown contact type: %q", t)
		}
		contacts[i] = &directory.Contact{
			ID:          id,
			Value:       v,
			ContactType: directory.ContactType(ct),
			Label:       l,
		}
	}
	return contacts, nil
}

func entityInfoFromFieldList(ei interface{}) (*directory.EntityInfo, error) {
	mei, ok := ei.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to parse input entity info data: %+v", ei)
	}

	fn, _ := mei["firstName"].(string)
	mi, _ := mei["middleInitial"].(string)
	ln, _ := mei["lastName"].(string)
	gn, _ := mei["groupName"].(string)
	dn, _ := mei["displayName"].(string)
	n, _ := mei["note"].(string)

	// If no display name was provided then build one from our input
	if dn == "" {
		if fn != "" || ln != "" {
			if mi != " " {
				dn = fn + " " + mi + ". " + ln
			} else {
				dn = fn + " " + ln
			}
		} else if gn != "" {
			dn = gn
		} else {
			return nil, errors.New("Display name cannot be empty and not enough information was supplied to infer one")
		}
	}
	entityInfo := &directory.EntityInfo{
		FirstName:     fn,
		MiddleInitial: mi,
		LastName:      ln,
		GroupName:     gn,
		DisplayName:   dn,
		Note:          n,
	}
	return entityInfo, nil
}