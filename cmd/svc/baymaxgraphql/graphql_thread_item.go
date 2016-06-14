package main

import (
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
	"golang.org/x/net/context"
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
			// TODO: "editor: Entity"
			// TODO: "editedTimestamp: Int"
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.Message)
			return ok
		},
	},
)

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
						url = media.URL(svc.mediaAPIDomain, mediaID)
					} else {
						url = media.ThumbnailURL(svc.mediaAPIDomain, mediaID, height, width, crop)
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
			"title": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"data":  &graphql.Field{Type: attachmentDataType},
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
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
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
					ent, err := transformEntityToResponse(svc.staticURLPrefix, entity, sh, gqlctx.Account(ctx))
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
	account := gqlctx.Account(ctx)
	if account == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}

	it, err := transformThreadItemToResponse(threadItem, "", account.ID, webdomain, mediaAPIDomain)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return it, nil
}
