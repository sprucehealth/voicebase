package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/golog"
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/graphql"
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
			"channel":      &graphql.Field{Type: graphql.NewNonNull(channelEnumType)},
			"id":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"displayValue": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var threadItemConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "ThreadItem",
	NodeType: threadItemType,
})

var messageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Message",
		Fields: graphql.Fields{
			"summaryMarkup": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"textMarkup":    &graphql.Field{Type: graphql.String},
			"refs": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(nodeInterfaceType)),
				Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					ctx := p.Context

					msg := p.Source.(*models.Message)
					if msg == nil {
						return nil, errors.InternalError(ctx, errors.New("message is nil"))
					}

					refs := make([]interface{}, 0, len(msg.Refs))
					for _, r := range msg.Refs {
						switch r.Type {
						case models.EntityRef:
							e, err := lookupEntity(ctx, svc, ram, r.ID)
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
				}),
			},
			"source":       &graphql.Field{Type: graphql.NewNonNull(endpointType)},
			"destinations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(endpointType))},
			"attachments":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(attachmentType))},
			"viewDetails": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(threadItemViewDetailsType)),
				Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					// patients cannot see item view details
					acc := gqlctx.Account(ctx)
					if acc.Type == auth.AccountType_PATIENT {
						return []interface{}{}, nil
					}

					m := p.Source.(*models.Message)
					if m == nil {
						return nil, errors.InternalError(ctx, errors.New("message is nil"))
					}
					ram := raccess.ResourceAccess(p)
					return lookupThreadItemViewDetails(ctx, ram, m.ThreadItemID)
				}),
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Message)
			return ok
		},
	},
)

// deletedMessageType is a sentinel that replaces messages that have been deleted.
var deletedMessageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "DeletedMessage",
		Description: "This structure is a tombstone for a message that has been deleted.",
		Fields: graphql.Fields{
			"placeholder": &graphql.Field{
				Description: "Structs can't be empty but we have no useful fields to include so here we are.",
				Type:        graphql.String,
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.DeletedMessage)
			return ok
		},
	},
)

var messageUpdateType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MessageUpdate",
		Fields: graphql.Fields{
			// Place holder field is overwritten in the init below
			"threadItem": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.MessageUpdate)
			return ok
		},
	},
)

var messageDeleteType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MessageDelete",
		Fields: graphql.Fields{
			// Place holder field is overwritten in the init below
			"threadItem": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.MessageDelete)
			return ok
		},
	},
)

func init() {
	// Can't create the threadItem field at decleration because it's a recursive type
	messageUpdateType.AddFieldConfig("threadItem", &graphql.Field{
		Type: graphql.NewNonNull(threadItemType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			ram := raccess.ResourceAccess(p)
			svc := serviceFromParams(p)
			ctx := p.Context
			update := p.Source.(*models.MessageUpdate)
			if selectingOnlyID(p) {
				return &models.ThreadItem{ID: update.ThreadItemID}, nil
			}
			return lookupThreadItem(ctx, ram, update.ThreadItemID, svc.webDomain, svc.mediaAPIDomain)
		},
	})
	messageDeleteType.AddFieldConfig("threadItem", &graphql.Field{
		Type: graphql.NewNonNull(threadItemType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			ram := raccess.ResourceAccess(p)
			svc := serviceFromParams(p)
			ctx := p.Context
			delete := p.Source.(*models.MessageDelete)
			if selectingOnlyID(p) {
				return &models.ThreadItem{ID: delete.ThreadItemID}, nil
			}
			return lookupThreadItem(ctx, ram, delete.ThreadItemID, svc.webDomain, svc.mediaAPIDomain)
		},
	})
}

var imageAttachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ImageAttachment",
		Fields: graphql.Fields{
			"mimetype":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"thumbnailURL": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
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
					account := gqlctx.Account(ctx)
					if account == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}

					attachment := p.Source.(*models.ImageAttachment)
					if attachment == nil {
						return nil, errors.InternalError(ctx, errors.New("attachment is nil"))
					}

					width := p.Args["width"].(int)
					height := p.Args["height"].(int)
					crop := p.Args["crop"].(bool)

					mediaID, err := lmedia.ParseMediaID(attachment.MediaID)
					if err != nil {
						golog.Errorf("Unable to parse mediaID out of url %s.", attachment.URL)
					}

					var url string
					if width == 0 && height == 0 {
						url = media.URL(svc.mediaAPIDomain, mediaID, attachment.Mimetype)
					} else {
						url = media.ThumbnailURL(svc.mediaAPIDomain, mediaID, attachment.Mimetype, height, width, crop)
					}

					return &models.Image{
						URL:    url,
						Width:  width,
						Height: height,
					}, nil
				},
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.ImageAttachment)
			return ok
		},
	},
)

var videoAttachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VideoAttachment",
		Fields: graphql.Fields{
			"mimetype":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"thumbnailURL": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.VideoAttachment)
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
			_, ok := value.(*models.AudioAttachment)
			return ok
		},
	},
)

var bannerButtonAttachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "BannerButtonAttachment",
		Fields: graphql.Fields{
			"title":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"ctaText": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"iconURL": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"tapURL":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.BannerButtonAttachment)
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
			bannerButtonAttachmentType,
			videoAttachmentType,
		},
	},
)

var attachmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Attachment",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:              graphql.NewNonNull(graphql.ID),
				DeprecationReason: `Use attachment.dataID instead if you are looking for the ID of the data contained within the attachment. attacment.id will have _attachment appended to ensure that the attachment id is not the same as the data contained within it.`,
			},
			"dataID":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":          &graphql.Field{Type: graphql.NewNonNull(attachmentInputTypeEnum)},
			"originalTitle": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"title":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"data":          &graphql.Field{Type: attachmentDataType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Attachment)
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
			deletedMessageType,
			messageUpdateType,
			messageDeleteType,
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
			"id":                &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"uuid":              &graphql.Field{Type: graphql.ID},
			"timestamp":         &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"modifiedTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"actorEntityID":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"internal":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"data":              &graphql.Field{Type: graphql.NewNonNull(threadItemDataType)},
			"actor": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					svc := serviceFromParams(p)
					it := p.Source.(*models.ThreadItem)
					if it == nil {
						return nil, errors.InternalError(ctx, errors.New("thread item is nil"))
					}
					if selectingOnlyID(p) {
						return &models.Entity{ID: it.ActorEntityID}, nil
					}

					ram := raccess.ResourceAccess(p)
					entity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						Key: &directory.LookupEntitiesRequest_EntityID{
							EntityID: it.ActorEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
					})
					if err != nil {
						return nil, err
					}

					sh := devicectx.SpruceHeaders(ctx)
					ent, err := transformEntityToResponse(ctx, svc.staticURLPrefix, entity, sh, gqlctx.Account(ctx))
					if err != nil {
						return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %s", err))
					}
					return ent, nil
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Args: graphql.FieldConfigArgument{
					"savedQueryID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ti := p.Source.(*models.ThreadItem)
					svc := serviceFromParams(p)
					savedQueryID := p.Args["savedQueryID"].(string)
					return deeplink.ThreadMessageURL(svc.webDomain, ti.OrganizationID, savedQueryID, ti.ThreadID, ti.ID), nil
				},
			},
			"shareableDeeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ti := p.Source.(*models.ThreadItem)
					svc := serviceFromParams(p)
					return deeplink.ThreadMessageURLShareable(svc.webDomain, ti.OrganizationID, ti.ThreadID, ti.ID), nil
				},
			},
		},
	},
)

func lookupThreadItem(ctx context.Context, ram raccess.ResourceAccessor, threadItemID, webdomain, mediaAPIDomain string) (interface{}, error) {
	threadItem, err := ram.ThreadItem(ctx, threadItemID)
	if err != nil {
		return nil, err
	}
	it, err := transformThreadItemToResponse(threadItem, "", webdomain, mediaAPIDomain)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return it, nil
}
