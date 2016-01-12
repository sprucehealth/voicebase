package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/conc"
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

const (
	createAccountResultSuccess             = "SUCCESS"
	createAccountResultEmailExists         = "EMAIL_EXISTS"
	createAccountResultEmailNotValid       = "EMAIL_NOT_VALID"
	createAccountResultPhoneNumberNotValid = "PHONE_NUMBER_NOT_VALID"
)

type createAccountOutput struct {
	ClientMutationID string   `json:"clientMutationId"`
	Result           string   `json:"result"`
	Token            string   `json:"token,omitempty"`
	Account          *account `json:"account,omitempty"`
}

var createAccountResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CreateAccountResult",
		Description: "Result of createAccount mutation",
		Values: graphql.EnumValueConfigMap{
			createAccountResultSuccess: &graphql.EnumValueConfig{
				Value:       createAccountResultSuccess,
				Description: "Success",
			},
			createAccountResultEmailExists: &graphql.EnumValueConfig{
				Value:       createAccountResultEmailExists,
				Description: "An account with the provided email already exists",
			},
			createAccountResultEmailNotValid: &graphql.EnumValueConfig{
				Value:       createAccountResultEmailNotValid,
				Description: "Provided email is not valid",
			},
			createAccountResultPhoneNumberNotValid: &graphql.EnumValueConfig{
				Value:       createAccountResultPhoneNumberNotValid,
				Description: "Provided phone number is not valid",
			},
		},
	},
)

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
			"result":           &graphql.Field{Type: graphql.NewNonNull(createAccountResultType)},
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
			"uuid":         &graphql.InputObjectFieldConfig{Type: graphql.String},
			"text":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"destinations": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(channelEnumType))},
			"internal":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

// postMessage

const (
	postMessageResultSuccess            = "SUCCESS"
	postMessageResultThreadDoesNotExist = "THREAD_DOES_NOT_EXIST"
)

type postMessageOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Result           string `json:"result"`
	ItemID           string `json:"itemID,omitempty"`
	ItemEdge         *Edge  `json:"itemEdge,omitempty"`
}

var postMessageResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "PostMessageResult",
		Description: "Result of postMessage mutation",
		Values: graphql.EnumValueConfigMap{
			postMessageResultSuccess: &graphql.EnumValueConfig{
				Value:       postMessageResultSuccess,
				Description: "Success",
			},
			postMessageResultThreadDoesNotExist: &graphql.EnumValueConfig{
				Value:       postMessageResultThreadDoesNotExist,
				Description: "Thread does not exist",
			},
		},
	},
)

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
			"result":           &graphql.Field{Type: graphql.NewNonNull(postMessageResultType)},
			"itemID":           &graphql.Field{Type: graphql.ID},
			"itemEdge":         &graphql.Field{Type: threadItemConnectionType.EdgeType},
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
			"result":             &graphql.Field{Type: graphql.NewNonNull(postMessageResultType)},
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
				ctx := contextFromParams(p)
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				email := input["email"].(string)
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
				// TODO: this is not thread safe.. fix once graphql lib supports passing down context through query
				p.Source.(map[string]interface{})["context"] = ctxWithAccount(ctx, acc)
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
				ctx := contextFromParams(p)
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
				ctx := contextFromParams(p)
				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				req := &auth.CreateAccountRequest{
					Email:    input["email"].(string),
					Password: input["password"].(string),
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
						return &authenticateOutput{
							ClientMutationID: mutationID,
							Result:           createAccountResultEmailExists,
						}, nil
					case auth.InvalidEmail:
						return &authenticateOutput{
							ClientMutationID: mutationID,
							Result:           createAccountResultEmailNotValid,
						}, nil
					case auth.InvalidPhoneNumber:
						return &authenticateOutput{
							ClientMutationID: mutationID,
							Result:           createAccountResultPhoneNumberNotValid,
						}, nil
					default:
						return nil, internalError(err)
					}
				}
				accountID := res.Account.ID

				var orgEntityID string
				var accEntityID string
				{
					// Create organization
					res, err := svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
						Name: "Test Organization", // TODO
						Type: directory.EntityType_ORGANIZATION,
					})
					if err != nil {
						return nil, internalError(err)
					}
					orgEntityID = res.Entity.ID

					// Create entity
					res, err = svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
						Name:                      req.FirstName + " " + req.LastName, // TODO
						Type:                      directory.EntityType_INTERNAL,
						ExternalID:                accountIDType + ":" + accountID,
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
						AreaCode: "415",
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
				// TODO: this is not thread safe.. fix once graphql lib supports passing down context through query
				p.Source.(map[string]interface{})["context"] = ctxWithAccount(ctx, acc)
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Result:           createAccountResultSuccess,
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
				ctx := contextFromParams(p)
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
				ctx := contextFromParams(p)
				acc := accountFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				threadID := input["threadID"].(string)

				tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
					ThreadID: threadID,
				})
				if err != nil {
					switch grpc.Code(err) {
					case codes.NotFound:
						return &postMessageOutput{
							ClientMutationID: mutationID,
							Result:           postMessageResultThreadDoesNotExist,
						}, nil
					}
					return nil, internalError(err)
				}
				thread := tres.Thread

				ent, err := svc.entityForAccountID(ctx, thread.OrganizationID, acc.ID)
				if err != nil {
					return nil, internalError(err)
				}
				if ent == nil {
					return nil, internalError(fmt.Errorf("entity for org %s and account %s not found", thread.OrganizationID, acc.ID))
				}

				var title bml.BML
				title = append(title, &bml.Ref{ID: ent.ID, Type: bml.EntityRef, Text: ent.Name})
				titleStr, err := title.Format()
				if err != nil {
					return nil, internalError(fmt.Errorf("invalid title BML %+v: %s", title, err))
				}
				// TODO: check destinations for additional title content

				msg := input["msg"].(map[string]interface{})
				req := &threading.PostMessageRequest{
					ThreadID:     threadID,
					Title:        titleStr,
					Text:         msg["text"].(string),
					Internal:     msg["internal"].(bool),
					FromEntityID: ent.ID,
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
						ID:      ent.ID,
					},
				}
				// TODO
				// if dests, ok := msg["destinations"].([]interface{}); ok && len(dests) != 0 {
				// 	for _, d := range dests {
				// 		channel := d.(string)
				// 		req.Destinations = append(req.Destinations, &threading.Endpoint{
				// 			Channel: threading.En
				// 		})
				// 	}
				// }
				if uuid, ok := msg["uuid"].(string); ok {
					req.UUID = uuid
				}

				pmres, err := svc.threading.PostMessage(ctx, req)
				if err != nil {
					return nil, internalError(err)
				}

				it, err := transformThreadItemToResponse(pmres.Item)
				if err != nil {
					return nil, internalError(fmt.Errorf("failed to transform thread item: %s", err))
				}
				return &postMessageOutput{
					ClientMutationID: mutationID,
					Result:           postMessageResultSuccess,
					ItemID:           it.ID,
					ItemEdge:         &Edge{Node: it, Cursor: ConnectionCursor(pmres.Item.ID)},
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
				ctx := contextFromParams(p)
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
				"input": &graphql.ArgumentConfig{Type: registerDeviceForPushInputType},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ctx := contextFromParams(p)
				acc := accountFromContext(ctx)
				sh := spruceHeadersFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated
				}

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
					return nil, errors.New("device registration failed")
				}

				result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
				result.Set("registerDeviceForPush", true)
				return &registerDeviceForPushOutput{
					ClientMutationID: mutationID,
				}, nil
			},
		},
	},
})
