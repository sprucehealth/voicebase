package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
			"titleMarkup": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":      &graphql.Field{Type: graphql.NewNonNull(messageStatusType)},
			"textMarkup":  &graphql.Field{Type: graphql.String},
			"refs": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(nodeInterfaceType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context

					msg := p.Source.(*message)
					if msg == nil {
						return nil, internalError(errors.New("message is nil"))
					}

					refs := make([]interface{}, 0, len(msg.Refs))
					for _, r := range msg.Refs {
						switch r.Type {
						case entityRef:
							e, err := lookupEntity(ctx, svc, r.ID)
							if err != nil {
								return nil, err
							}
							refs = append(refs, e)
						default:
							// Log this but continue as it's a better soft-fail state
							golog.Errorf("unknown reference type %s", r.Type)
						}
					}
					return refs, nil
				},
			},
			"source":       &graphql.Field{Type: graphql.NewNonNull(endpointType)},
			"destinations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(endpointType))},
			"attachments":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(attachmentType))},
			"viewDetails": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(threadItemViewDetailsType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					m := p.Source.(*message)
					if m == nil {
						return nil, internalError(errors.New("message is nil"))
					}
					svc := serviceFromParams(p)
					ctx := p.Context
					return lookupThreadItemViewDetails(ctx, svc, m.ThreadItemID)
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Args: graphql.FieldConfigArgument{
					"savedQueryID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ti := p.Source.(*threadItem)
					svc := serviceFromParams(p)
					savedQueryID := p.Args["savedQueryID"].(string)
					return deeplink.ThreadMessageURL(svc.webDomain, ti.OrganizationID, savedQueryID, ti.ThreadID, ti.ID), nil
				},
			},
			"shareableDeeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ti := p.Source.(*threadItem)
					svc := serviceFromParams(p)
					return deeplink.ThreadMessageURLShareable(svc.webDomain, ti.OrganizationID, ti.ThreadID, ti.ID), nil
				},
			},
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
			"image": &graphql.Field{
				Type: graphql.NewNonNull(imageType),
				Args: graphql.FieldConfigArgument{
					"width": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
					"height": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
					"crop": &graphql.ArgumentConfig{
						Type:         graphql.Boolean,
						DefaultValue: false,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					account := accountFromContext(ctx)
					if account == nil {
						return nil, errNotAuthenticated
					}

					attachment := p.Source.(*imageAttachment)
					if attachment == nil {
						return nil, internalError(errors.New("attachment is nil"))
					}

					width := p.Args["width"].(int)
					height := p.Args["height"].(int)
					crop := p.Args["crop"].(bool)

					mediaID, err := media.ParseMediaID(attachment.URL)
					if err != nil {
						golog.Errorf("Unable to parse mediaID out of url %s.", attachment.URL)
					}
					url, err := svc.mediaSigner.SignedURL(mediaID, attachment.Mimetype, account.ID, width, height, crop)
					if err != nil {
						return nil, internalError(err)
					}
					return &image{
						URL:    url,
						Width:  width,
						Height: height,
					}, nil
				},
			},
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
			"durationInSeconds": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
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
			"uuid":          &graphql.Field{Type: graphql.ID},
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
					ctx := p.Context
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: it.ActorEntityID,
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
					for _, e := range res.Entities {
						ent, err := transformEntityToResponse(e)
						if err != nil {
							return nil, internalError(fmt.Errorf("failed to transform entity: %s", err))
						}
						return ent, nil
					}
					return nil, errors.New("actor not found")
				},
			},
		},
	},
)

func lookupThreadItem(ctx context.Context, svc *service, id string) (interface{}, error) {
	res, err := svc.threading.ThreadItem(ctx, &threading.ThreadItemRequest{
		ItemID: id,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, errors.New("thread item not found")
		}
		return nil, internalError(err)
	}
	account := accountFromContext(ctx)
	if account == nil {
		return nil, errNotAuthenticated
	}

	it, err := transformThreadItemToResponse(res.Item, "", account.ID, svc.mediaSigner)
	if err != nil {
		return nil, internalError(err)
	}
	return it, nil
}
