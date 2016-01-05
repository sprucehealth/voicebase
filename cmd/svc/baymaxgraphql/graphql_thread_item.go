package main

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
)

var channelEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ChannelType",
		Description: "Type of communication channel",
		Values: graphql.EnumValueConfigMap{
			"APP": &graphql.EnumValueConfig{
				Value:       "APP",
				Description: "Application or web",
			},
			"SMS": &graphql.EnumValueConfig{
				Value:       "SMS",
				Description: "SMS text message",
			},
			"VOICE": &graphql.EnumValueConfig{
				Value:       "VOICE",
				Description: "Voice call",
			},
			"EMAIL": &graphql.EnumValueConfig{
				Value:       "EMAIL",
				Description: "Email message",
			},
		},
	},
)

var endpointType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Endpoint",
		Description: "Communication endpoint",
		Fields: graphql.Fields{
			"channel": &graphql.Field{Type: graphql.NewNonNull(channelEnumType)},
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var threadItemConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "ThreadItem",
	NodeType: threadItemType,
})

var messageStatusType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "MessageStatus",
		Description: "Status of a thread message",
		Values: graphql.EnumValueConfigMap{
			"NORMAL": &graphql.EnumValueConfig{
				Value:       "NORMAL",
				Description: "Normal thread message",
			},
			"DELETED": &graphql.EnumValueConfig{
				Value:       "DELETED",
				Description: "Message has been deleted",
			},
		},
	},
)

var messageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Message",
		Fields: graphql.Fields{
			"status":       &graphql.Field{Type: graphql.NewNonNull(messageStatusType)},
			"text":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"source":       &graphql.Field{Type: graphql.NewNonNull(endpointType)},
			"destinations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(endpointType))},
			"attachments":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(attachmentType))},
			// TODO: "editor: Entity"
			// TODO: "editedTimestamp: Int"
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*message)
			return ok
		},
	},
)

var imageAttachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ImageAttachment",
		Fields: graphql.Fields{
			"mimetype": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"width":    &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"height":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*imageAttachment)
			return ok
		},
	},
)

var audioAttachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AudioAttachment",
		Fields: graphql.Fields{
			"mimetype":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":               &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"durationInSeconds": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*audioAttachment)
			return ok
		},
	},
)

var attachmentDataType = graphql.NewUnion(
	graphql.UnionConfig{
		Name:        "AttachmentData",
		Description: "Possible types for the attachment data field",
		Types: []*graphql.Object{
			imageAttachmentType,
			audioAttachmentType,
		},
	},
)

var attachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Attachment",
		Fields: graphql.Fields{
			"title": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"data":  &graphql.Field{Type: attachmentDataType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*attachment)
			return ok
		},
	},
)

var threadItemDataType = graphql.NewUnion(
	graphql.UnionConfig{
		Name:        "ThreadItemData",
		Description: "Possible types for the thread item data field",
		Types: []*graphql.Object{
			messageType,
			// messageUpdatedType,
			// followerUpdatedType,
		},
	},
)

var threadItemType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ThreadItem",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"timestamp":     &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"actorEntityID": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"internal":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"data":          &graphql.Field{Type: graphql.NewNonNull(threadItemDataType)},
			"actor": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					it := p.Source.(*threadItem)
					if it == nil {
						return nil, internalError(errors.New("thread item is nil"))
					}
					if selectingOnlyID(p) {
						return &entity{ID: it.ActorEntityID}, nil
					}

					svc := serviceFromParams(p)
					ctx := contextFromParams(p)
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: it.ActorEntityID,
							},
						})
					if err != nil {
						return nil, internalError(err)
					}
					for _, e := range res.Entities {
						return &entity{
							ID:   e.ID,
							Name: e.Name,
						}, nil
					}
					return nil, errors.New("actor not found")
				},
			},
		},
	},
)
