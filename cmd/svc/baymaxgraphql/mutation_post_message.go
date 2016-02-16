package main

import (
	"errors"
	"fmt"
	"github.com/sprucehealth/backend/libs/phone"

	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// message

var endpointInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name:        "EndpointInput",
		Description: "Communication endpoint",
		Fields: graphql.InputObjectConfigFieldMap{
			"channel": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(channelEnumType)},
			"id":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var messageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"uuid":         &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"text":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"destinations": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(endpointInputType))},
			"internal":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

// postMessage

type postMessageOutput struct {
	ClientMutationID string  `json:"clientMutationId,omitempty"`
	Success          bool    `json:"success"`
	ErrorCode        string  `json:"errorCode,omitempty"`
	ErrorMessage     string  `json:"errorMessage,omitempty"`
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

const (
	postMessageErrorCodeThreadDoesNotExist = "THREAD_DOES_NOT_EXIST"
)

var postMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "PostMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		postMessageErrorCodeThreadDoesNotExist: &graphql.EnumValueConfig{
			Value:       postMessageErrorCodeThreadDoesNotExist,
			Description: "Thread with provided ID does not exist.",
		},
	},
})

var postMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PostMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: postMessageErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"itemEdge":         &graphql.Field{Type: graphql.NewNonNull(threadItemConnectionType.EdgeType)},
			"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*postMessageOutput)
			return ok
		},
	},
)

var postMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(postMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(postMessageInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
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
				return &postMessageOutput{
					Success:      false,
					ErrorCode:    postMessageErrorCodeThreadDoesNotExist,
					ErrorMessage: "Thread does not exist.",
				}, nil
			}
			return nil, internalError(ctx, err)
		}
		thr := tres.Thread

		ent, err := svc.entityForAccountID(ctx, thr.OrganizationID, acc.ID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if ent == nil || ent.Type != directory.EntityType_INTERNAL {
			return nil, userError(ctx, errTypeNotAuthorized, "Not a member of the organization")
		}

		var primaryEntity *directory.Entity
		if thr.PrimaryEntityID != "" {
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
				return nil, internalError(ctx, err)
			}
			if len(res.Entities) != 1 {
				return nil, internalError(ctx, fmt.Errorf("lookup entities returned %d result(s) expected 1 entity ID %s", len(res.Entities), thr.PrimaryEntityID))
			}
			primaryEntity = res.Entities[0]
		}

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
			return nil, internalError(ctx, err)
		}
		fromName := ent.Info.DisplayName
		if fromName == "" && len(ent.Contacts) != 0 {
			switch c := ent.Contacts[0]; c.ContactType {
			case directory.ContactType_PHONE:
				fromName, err = phone.Format(c.Value, phone.Pretty)
				if err != nil {
					fromName = c.Value
				}
			default:
				fromName = c.Value
			}
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

		if primaryEntity == nil || primaryEntity.Type != directory.EntityType_EXTERNAL {
			// Internal messages only make sense on external threads
			req.Internal = false
		}

		dests, _ := msg["destinations"].([]interface{})

		var title bml.BML

		// For a message to be considered by sending externally it needs to not be marked as internal,
		// sent by someone who is internal, and there needs to be a primary entity on the thread.
		isExternal := !req.Internal && thr.PrimaryEntityID != "" && ent.Type == directory.EntityType_INTERNAL && primaryEntity.Type == directory.EntityType_EXTERNAL
		if isExternal && len(dests) != 0 {
			destSet := make(map[string]struct{}, len(dests))
			for _, d := range dests {
				endpoint, _ := d.(map[string]interface{})
				endpointChannel, _ := endpoint["channel"].(string)
				endpointID, _ := endpoint["id"].(string)
				var ct directory.ContactType
				var ec threading.Endpoint_Channel
				switch endpointChannel {
				case endpointChannelEmail:
					ct = directory.ContactType_EMAIL
					ec = threading.Endpoint_EMAIL
				case endpointChannelSMS:
					ct = directory.ContactType_PHONE
					ec = threading.Endpoint_SMS
				default:
					return nil, fmt.Errorf("unsupported destination endpoint channel %q", endpointChannel)
				}
				var e *threading.Endpoint
				// Assert that the provided destination matches one of the contacts for the primary entity on the thread
				for _, c := range primaryEntity.Contacts {
					if c.ContactType == ct && c.Value == endpointID {
						e = &threading.Endpoint{
							Channel: ec,
							ID:      c.Value,
						}
						break
					}
				}
				if e == nil {
					return nil, fmt.Errorf("The provided destination contact info does not belong to the primary entity for this thread: %q, %q", endpointChannel, endpointID)
				}
				req.Destinations = append(req.Destinations, e)
				switch e.Channel {
				case threading.Endpoint_SMS:
					destSet["SMS"] = struct{}{}
				case threading.Endpoint_EMAIL:
					destSet["Email"] = struct{}{}
				}
			}
			for d := range destSet {
				if len(title) != 0 {
					title = append(title, " & ")
				}
				title = append(title, d)
			}
		} else if req.Internal {
			title = append(title[:0], "Internal")
		}
		if uuid, ok := msg["uuid"].(string); ok {
			req.UUID = uuid
		}

		titleStr, err := title.Format()
		if err != nil {
			return nil, internalError(ctx, fmt.Errorf("invalid title BML %+v: %s", title, err))
		}
		req.Title = titleStr

		pmres, err := svc.threading.PostMessage(ctx, req)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		it, err := transformThreadItemToResponse(pmres.Item, req.UUID, acc.ID, svc.mediaSigner)
		if err != nil {
			return nil, internalError(ctx, fmt.Errorf("failed to transform thread item: %s", err))
		}
		th, err := transformThreadToResponse(pmres.Thread)
		if err != nil {
			return nil, internalError(ctx, fmt.Errorf("failed to transform thread: %s", err))
		}
		if primaryEntity != nil {
			th.Title = threadTitleForEntity(primaryEntity)
			th.AllowInternalMessages = primaryEntity.Type != directory.EntityType_SYSTEM
			th.IsDeletable = primaryEntity.Type != directory.EntityType_SYSTEM
		} else if err := svc.hydrateThreads(ctx, []*thread{th}); err != nil {
			return nil, internalError(ctx, err)
		}
		return &postMessageOutput{
			ClientMutationID: mutationID,
			Success:          true,
			ItemEdge:         &Edge{Node: it, Cursor: ConnectionCursor(pmres.Item.ID)},
			Thread:           th,
		}, nil
	},
}
