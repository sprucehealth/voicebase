package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
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
		"callEntity": callEntityMutation,
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
