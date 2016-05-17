package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
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
			"attachments":  &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(attachmentInputType))},
		},
	},
)

var (
	attachmentTypeImage    = "IMAGE"
	attachmentTypeVisit    = "VISIT"
	attachmentTypeCarePlan = "CARE_PLAN"
)

var attachmentInputTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AttachmentInputType",
	Values: graphql.EnumValueConfigMap{
		attachmentTypeImage: &graphql.EnumValueConfig{
			Value:       attachmentTypeImage,
			Description: "The attachment type representing an image",
		},
		attachmentTypeVisit: &graphql.EnumValueConfig{
			Value:       attachmentTypeVisit,
			Description: "The attachment type representing a visit",
		},
		attachmentTypeCarePlan: &graphql.EnumValueConfig{
			Value:       attachmentTypeCarePlan,
			Description: "The attachment type representing a care plan",
		},
	},
})

var attachmentInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AttachmentInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"title":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"mediaID":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"attachmentType": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(attachmentInputTypeEnum)},
		},
	},
)

func attachmentTypeEnumAsThreadingEnum(t string) (threading.Attachment_Type, error) {
	switch t {
	case attachmentTypeImage:
		return threading.Attachment_IMAGE, nil
	case attachmentTypeVisit:
		return threading.Attachment_VISIT, nil
	case attachmentTypeCarePlan:
		return threading.Attachment_CARE_PLAN, nil

	}
	return threading.Attachment_Type(0), fmt.Errorf("Unknown attachment type %s", t)
}

// postMessage

type postMessageOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	UUID             string         `json:"uuid,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	ItemEdge         *Edge          `json:"itemEdge"`
	Thread           *models.Thread `json:"thread"`
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
	postMessageErrorCodeInternalNotAllowed = "INTERNAL_MESSAGE_NOT_ALLOWED"
	postMessageErrorCodeInvalidAttachment  = "INVALID_ATTACHMENT"
)

var postMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "PostMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		postMessageErrorCodeThreadDoesNotExist: &graphql.EnumValueConfig{
			Value:       postMessageErrorCodeThreadDoesNotExist,
			Description: "Thread with provided ID does not exist.",
		},
		postMessageErrorCodeInternalNotAllowed: &graphql.EnumValueConfig{
			Value:       postMessageErrorCodeInternalNotAllowed,
			Description: "The caller is not allowed to post internal messages",
		},
		postMessageErrorCodeInvalidAttachment: &graphql.EnumValueConfig{
			Value:       postMessageErrorCodeInvalidAttachment,
			Description: "At least one attachment is invalid",
		},
	},
})

var postMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PostMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
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
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID := input["threadID"].(string)
		msg := input["msg"].(map[string]interface{})

		thr, err := ram.Thread(ctx, threadID, "")
		if err != nil {
			switch errors.Type(err) {
			case errors.ErrTypeNotFound:
				return &postMessageOutput{
					Success:      false,
					ErrorCode:    postMessageErrorCodeThreadDoesNotExist,
					ErrorMessage: "Thread does not exist.",
				}, nil
			}
			return nil, err
		}

		isInternalMsg := msg["internal"].(bool)
		if isInternalMsg && !allowInternalMessages(thr, acc) {
			return &postMessageOutput{
				Success:      false,
				ErrorCode:    postMessageErrorCodeInternalNotAllowed,
				ErrorMessage: "Internal messages are not allowed.",
			}, nil
		}

		if err := ram.CanPostMessage(ctx, thr.ID); err != nil {
			return nil, err
		}

		ent, err := ram.EntityForAccountID(ctx, thr.OrganizationID, acc.ID)
		if err != nil {
			return nil, err
		}

		var primaryEntity *directory.Entity
		if thr.PrimaryEntityID != "" {
			primaryEntity, err = ram.Entity(ctx, thr.PrimaryEntityID, []directory.EntityInformation{directory.EntityInformation_CONTACTS}, 0)
			if err != nil {
				return nil, err
			}
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
			return nil, errors.InternalError(ctx, err)
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

		// Need to track the care plans so we can flag them as submitted after posting
		var carePlans []*care.CarePlan

		var msgAttachments []interface{}
		if mas, ok := msg["attachments"]; ok {
			msgAttachments = mas.([]interface{})
		}
		attachments := make([]*threading.Attachment, len(msgAttachments))
		for i, ma := range msgAttachments {
			mAttachment, _ := ma.(map[string]interface{})
			mAttachmentType, err := attachmentTypeEnumAsThreadingEnum(mAttachment["attachmentType"].(string))
			if err != nil {
				return nil, err
			}
			// TODO: Verify that the media at the ID exists
			var title string
			if _, ok := mAttachment["title"]; ok {
				title = mAttachment["title"].(string)
			}
			var attachment *threading.Attachment
			switch mAttachmentType {
			case threading.Attachment_VISIT:
				// can only attach visits on secure external threads
				if thr.Type != threading.ThreadType_SECURE_EXTERNAL {
					return nil, errors.ErrNotSupported(ctx, fmt.Errorf("Cannot attach a visit to thread of type %s", thr.Type.String()))
				}

				// ensure that the visit layout exists from which to create a visit
				visitLayoutRes, err := ram.VisitLayout(ctx, &layout.GetVisitLayoutRequest{
					ID: mAttachment["mediaID"].(string),
				})
				if err != nil {
					return nil, err
				}

				// create the visit from the visit layout
				createVisitRes, err := ram.CreateVisit(ctx, &care.CreateVisitRequest{
					EntityID:        thr.PrimaryEntityID,
					Name:            visitLayoutRes.VisitLayout.Name,
					LayoutVersionID: visitLayoutRes.VisitLayout.Version.ID,
					OrganizationID:  thr.OrganizationID,
				})
				if err != nil {
					return nil, err
				}

				attachment = &threading.Attachment{
					Type:  mAttachmentType,
					Title: createVisitRes.Visit.Name,
					URL:   deeplink.VisitURL(svc.webDomain, thr.ID, createVisitRes.Visit.ID),
					Data: &threading.Attachment_Visit{
						Visit: &threading.VisitAttachment{
							VisitID:   createVisitRes.Visit.ID,
							VisitName: createVisitRes.Visit.Name,
						},
					},
				}
			case threading.Attachment_CARE_PLAN:
				// can only attach visits on secure external threads
				if thr.Type != threading.ThreadType_SECURE_EXTERNAL {
					return nil, errors.ErrNotSupported(ctx, fmt.Errorf("Cannot attach a care plan to thread of type %s", thr.Type.String()))
				}

				// Make sure the care plan exists, the poster has access to it, and it hasn't yet been submitted
				cp, err := ram.CarePlan(ctx, mAttachment["mediaID"].(string))
				if err != nil {
					return nil, err
				}
				if cp.Submitted {
					return &postMessageOutput{
						Success:      false,
						ErrorCode:    postMessageErrorCodeInvalidAttachment,
						ErrorMessage: "The attached care plan has already been submitted.",
					}, nil
				}
				carePlans = append(carePlans, cp)

				attachment = &threading.Attachment{
					Type:  mAttachmentType,
					Title: cp.Name,
					URL:   deeplink.CarePlanURL(svc.webDomain, thr.ID, cp.ID),
					Data: &threading.Attachment_CarePlan{
						CarePlan: &threading.CarePlanAttachment{
							CarePlanID:   cp.ID,
							CarePlanName: cp.Name,
						},
					},
				}
			case threading.Attachment_IMAGE:
				mediaID := svc.media.URL(mAttachment["mediaID"].(string))
				meta, err := svc.media.GetMeta(mAttachment["mediaID"].(string))
				if err != nil {
					return nil, err
				}
				attachment = &threading.Attachment{
					Type:  mAttachmentType,
					Title: title,
					URL:   mediaID,
					Data: &threading.Attachment_Image{
						Image: &threading.ImageAttachment{
							Mimetype: meta.MimeType,
							URL:      mediaID,
						},
					},
				}
			default:
				return nil, fmt.Errorf("Unknown message attachment type %d", mAttachmentType)
			}
			attachments[i] = attachment
		}

		req := &threading.PostMessageRequest{
			ThreadID:     threadID,
			Text:         text,
			Internal:     isInternalMsg,
			FromEntityID: ent.ID,
			Source: &threading.Endpoint{
				Channel: threading.Endpoint_APP,
				ID:      ent.ID,
			},
			Summary:     summary,
			Attachments: attachments,
		}

		if primaryEntity == nil || primaryEntity.Type == directory.EntityType_ORGANIZATION {
			req.Internal = false
		}

		dests, _ := msg["destinations"].([]interface{})

		title, err := buildMessageTitleBasedOnDestinations(req, dests, thr, ent, primaryEntity)
		if err != nil {
			return nil, err
		}

		if len(title) == 0 {
			for _, a := range req.Attachments {
				if a.Type == threading.Attachment_VISIT {
					title = append(title, "Shared a visit:")
					break
				}
				if a.Type == threading.Attachment_CARE_PLAN {
					title = append(title, "Shared a care plan:")
					break
				}
			}
		}

		uuid, ok := msg["uuid"].(string)
		if ok {
			req.UUID = uuid
		}

		titleStr, err := title.Format()
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("invalid title BML %+v: %s", title, err))
		}
		req.Title = titleStr

		pmres, err := ram.PostMessage(ctx, req)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		// Flag care plans as submitted and attached to this message
		for _, cp := range carePlans {
			if err := ram.SubmitCarePlan(ctx, cp, pmres.Item.ID); err != nil {
				// Don't return an error here since it's too late to do much about this. Best to let the
				// mutation succeed and log these to be fixed up by hand.
				golog.Errorf("[MANUAL_INTERVENTION] Failed to submit care plan %s for thread item %s: %s", cp.ID, pmres.Item.ID, err)
			}
		}

		svc.segmentio.Track(&analytics.Track{
			Event:  fmt.Sprintf("posted-message-%s", strings.ToLower(thr.Type.String())),
			UserId: acc.ID,
			Properties: map[string]interface{}{
				"organization_id": thr.OrganizationID,
				"thread_id":       req.ThreadID,
			},
		})

		it, err := transformThreadItemToResponse(pmres.Item, req.UUID, acc.ID, svc.webDomain, svc.mediaSigner)
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform thread item: %s", err))
		}
		th, err := transformThreadToResponse(ctx, ram, pmres.Thread, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform thread: %s", err))
		}

		if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return &postMessageOutput{
			ClientMutationID: mutationID,
			Success:          true,
			ItemEdge:         &Edge{Node: it, Cursor: ConnectionCursor(pmres.Item.ID)},
			Thread:           th,
		}, nil
	}),
}

func buildMessageTitleBasedOnDestinations(
	req *threading.PostMessageRequest,
	dests []interface{},
	thr *threading.Thread,
	fromEntity, primaryEntity *directory.Entity,
) (bml.BML, error) {
	var title bml.BML
	// For a message to be considered by sending externally it needs to marked as not internal,
	// sent by someone who is internal, and there needs to be a primary entity on the thread.
	isExternal := !req.Internal && thr.PrimaryEntityID != "" && fromEntity.Type == directory.EntityType_INTERNAL && primaryEntity.Type == directory.EntityType_EXTERNAL
	if isExternal && len(dests) != 0 {
		destSet := make(map[string]struct{}, len(dests))
		for _, d := range dests {
			endpoint, _ := d.(map[string]interface{})
			endpointChannel, _ := endpoint["channel"].(string)
			endpointID, _ := endpoint["id"].(string)
			var ct directory.ContactType
			var ec threading.Endpoint_Channel
			switch endpointChannel {
			case models.EndpointChannelEmail:
				ct = directory.ContactType_EMAIL
				ec = threading.Endpoint_EMAIL
			case models.EndpointChannelSMS:
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
		destTitles := make([]string, 0, len(destSet))
		for d := range destSet {
			destTitles = append(destTitles, d)
		}
		sort.Strings(destTitles)
		for _, d := range destTitles {
			if len(title) != 0 {
				title = append(title, " & ")
			}
			title = append(title, d)
		}
	} else if req.Internal {
		title = append(title[:0], "Internal")
	}
	return title, nil
}
