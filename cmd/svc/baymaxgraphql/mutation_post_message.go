package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
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
	attachmentTypeAudio          = "AUDIO"
	attachmentTypeCarePlan       = "CARE_PLAN"
	attachmentTypeDocument       = "DOCUMENT"
	attachmentTypeGenericURL     = "GENERIC_URL"
	attachmentTypeImage          = "IMAGE"
	attachmentTypePaymentRequest = "PAYMENT_REQUEST"
	attachmentTypeVideo          = "VIDEO"
	attachmentTypeVisit          = "VISIT"
)

var attachmentInputTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AttachmentInputType",
	Values: graphql.EnumValueConfigMap{
		attachmentTypeAudio: &graphql.EnumValueConfig{
			Value:       attachmentTypeAudio,
			Description: "The attachment type representing an audio recording",
		},
		attachmentTypeCarePlan: &graphql.EnumValueConfig{
			Value:       attachmentTypeCarePlan,
			Description: "The attachment type representing a care plan",
		},
		attachmentTypeGenericURL: &graphql.EnumValueConfig{
			Value:       attachmentTypeGenericURL,
			Description: "The attachment type representing a generic URL",
		},
		attachmentTypeImage: &graphql.EnumValueConfig{
			Value:       attachmentTypeImage,
			Description: "The attachment type representing an image",
		},
		attachmentTypeVideo: &graphql.EnumValueConfig{
			Value:       attachmentTypeVideo,
			Description: "The attachment type representing a video",
		},
		attachmentTypeVisit: &graphql.EnumValueConfig{
			Value:       attachmentTypeVisit,
			Description: "The attachment type representing a visit",
		},
		attachmentTypePaymentRequest: &graphql.EnumValueConfig{
			Value:       attachmentTypePaymentRequest,
			Description: "The attachment type representing a payment request",
		},
		attachmentTypeDocument: &graphql.EnumValueConfig{
			Value:       attachmentTypeDocument,
			Description: "The attachemnt type representing a document",
		},
	},
})

var attachmentInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AttachmentInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"title": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"mediaID": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "DEPRECATED: use attachmentID instead",
			},
			"attachmentID":   &graphql.InputObjectFieldConfig{Type: graphql.String},
			"attachmentType": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(attachmentInputTypeEnum)},
		},
	},
)

func attachmentTypeAsEnum(a *threading.Attachment) (string, error) {
	switch a.Data.(type) {
	case *threading.Attachment_CarePlan:
		return attachmentTypeCarePlan, nil
	case *threading.Attachment_Image:
		return attachmentTypeImage, nil
	case *threading.Attachment_Video:
		return attachmentTypeVideo, nil
	case *threading.Attachment_Audio:
		return attachmentTypeAudio, nil
	case *threading.Attachment_Visit:
		return attachmentTypeVisit, nil
	case *threading.Attachment_Document:
		return attachmentTypeDocument, nil
	case *threading.Attachment_PaymentRequest:
		return attachmentTypePaymentRequest, nil
	}
	return "", fmt.Errorf("Unknown attachment type %T", a.Data)
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

type endpointInput struct {
	Channel string `gql:"channel"`
	ID      string `gql:"id"`
}

type attachmentInput struct {
	Title        string `gql:"title"`
	MediaID      string `gql:"mediaID"` // DEPRECATED
	AttachmentID string `gql:"attachmentID"`
	Type         string `gql:"attachmentType,nonempty"`
}

type messageInput struct {
	UUID         string            `gql:"uuid"`
	Text         string            `gql:"text"`
	Internal     bool              `gql:"internal"`
	Destinations []endpointInput   `gql:"destinations"`
	Attachments  []attachmentInput `gql:"attachments"`
}

type postMessageInput struct {
	ClientMutationID string       `gql:"clientMutationId"`
	ThreadID         string       `gql:"threadID,nonempty"`
	Msg              messageInput `gql:"msg,nonempty"`
}

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

		var in postMessageInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		thr, err := ram.Thread(ctx, in.ThreadID, "")
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

		if in.Msg.Internal && !allowInternalMessages(thr, acc) {
			return &postMessageOutput{
				Success:      false,
				ErrorCode:    postMessageErrorCodeInternalNotAllowed,
				ErrorMessage: "Internal messages are not allowed.",
			}, nil
		}

		if err := ram.CanPostMessage(ctx, thr.ID); err != nil {
			return nil, err
		}

		ent, err := entityInOrgForAccountID(ctx, ram, thr.OrganizationID, acc)
		if err != nil {
			return nil, err
		}

		var primaryEntity *directory.Entity
		if thr.PrimaryEntityID != "" {
			primaryEntity, err = raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: thr.PrimaryEntityID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth:             0,
					EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
				},
				Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		// Parse text and render as plain text so we can build a summary.
		textBML, err := bml.Parse(in.Msg.Text)
		if e, ok := err.(bml.ErrParseFailure); ok {
			return nil, fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
		} else if err != nil {
			return nil, errors.New("text is not valid markup")
		}

		// Validate referenced entities and covert tag to plain text for any that aren't allowed
		// first make a quick check to make sure there's any refs to avoid lookups
		hasRefs := false
		for _, e := range textBML {
			if _, ok := e.(*bml.Ref); ok {
				hasRefs = true
				break
			}
		}
		if hasRefs {
			threadType, err := transformThreadTypeToResponse(thr.Type)
			if err != nil {
				return nil, errors.Trace(err)
			}
			refEntities, err := addressableEntitiesForThread(ctx, ram, thr.OrganizationID, thr.ID, threadType)
			if err != nil {
				return nil, errors.Trace(err)
			}
			refEntitiesMap := make(map[string]*directory.Entity, len(refEntities))
			for _, e := range refEntities {
				refEntitiesMap[e.ID] = e
			}
			for i, e := range textBML {
				if r, ok := e.(*bml.Ref); ok {
					switch r.Type {
					case bml.EntityRef:
						e := refEntitiesMap[r.ID]
						if e == nil {
							// If the entitiy isn't in the addressable list then replace the tag with plain text
							textBML[i] = r.Text
						} else {
							// Make sure the name of the addressable entity matches what it should
							r.Text = e.Info.DisplayName
						}
					default:
						return nil, errors.Errorf("unknown reference type %s", r.Type)
					}
				}
			}
		}

		plainText, err := textBML.PlainText()
		if err != nil {
			// Shouldn't fail here since the parsing should have done validation
			return nil, errors.InternalError(ctx, err)
		}
		summary := summaryForEntityMessage(ent, plainText)

		attachments, carePlans, err := processIncomingAttachments(ctx, ram, svc, ent, thr.OrganizationID, in.Msg.Attachments, thr)
		if e, ok := err.(errInvalidAttachment); ok {
			return &postMessageOutput{
				Success:      false,
				ErrorCode:    postMessageErrorCodeInvalidAttachment,
				ErrorMessage: string(e),
			}, nil
		} else if err != nil {
			return nil, err
		}

		req := &threading.PostMessageRequest{
			ThreadID:     in.ThreadID,
			FromEntityID: ent.ID,
			UUID:         in.Msg.UUID,
			Message: &threading.MessagePost{
				Text:     in.Msg.Text,
				Internal: in.Msg.Internal,
				Source: &threading.Endpoint{
					Channel: threading.ENDPOINT_CHANNEL_APP,
					ID:      ent.ID,
				},
				Summary:     summary,
				Attachments: attachments,
			},
		}

		if primaryEntity == nil || primaryEntity.Type == directory.EntityType_ORGANIZATION {
			req.Message.Internal = false
		}

		title, err := buildMessageTitleBasedOnDestinations(req, in.Msg.Destinations, thr, ent, primaryEntity)
		if err != nil {
			return nil, err
		}

		if len(title) == 0 {
			for _, a := range req.Message.Attachments {
				if _, ok := a.Data.(*threading.Attachment_Visit); ok {
					title = append(title, "Shared a visit:")
					break
				}
				if _, ok := a.Data.(*threading.Attachment_CarePlan); ok {
					title = append(title, "Shared a care plan:")
					break
				}
				if _, ok := a.Data.(*threading.Attachment_PaymentRequest); ok {
					title = append(title, "Requested payment:")
					break
				}
				if _, ok := a.Data.(*threading.Attachment_Document); ok {
					title = append(title, "Shared a file:")
					break
				}
			}
		}

		titleStr, err := title.Format()
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("invalid title BML %+v: %s", title, err))
		}
		req.Message.Title = titleStr

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

		trackPostMessage(ctx, thr, req)

		it, err := transformThreadItemToResponse(pmres.Item, req.UUID, svc.webDomain, svc.mediaAPIDomain)
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
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			ItemEdge:         &Edge{Node: it, Cursor: ConnectionCursor(pmres.Item.ID)},
			Thread:           th,
		}, nil
	}),
}

func trackPostMessage(ctx context.Context, thr *threading.Thread, req *threading.PostMessageRequest) {
	acc := gqlctx.Account(ctx)

	for _, attachment := range req.Message.Attachments {
		switch attachment.GetData().(type) {
		case *threading.Attachment_Visit:
			analytics.SegmentTrack(&segment.Track{
				Event:  "visit-attached",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"organization_id": thr.OrganizationID,
					"thread_id":       req.ThreadID,
				},
			})
		case *threading.Attachment_CarePlan:
			analytics.SegmentTrack(&segment.Track{
				Event:  "care-plan-attached",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"organization_id": thr.OrganizationID,
					"thread_id":       req.ThreadID,
					"care_plan_id":    attachment.GetCarePlan().CarePlanID,
				},
			})
		case *threading.Attachment_PaymentRequest:
			analytics.SegmentTrack(&segment.Track{
				Event:  "payment-request-attached",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"organization_id": thr.OrganizationID,
					"thread_id":       req.ThreadID,
					"payment_id":      attachment.GetPaymentRequest().PaymentID,
				},
			})
		}
	}

	analytics.SegmentTrack(&segment.Track{
		Event:  fmt.Sprintf("posted-message-%s", strings.ToLower(thr.Type.String())),
		UserId: acc.ID,
		Properties: map[string]interface{}{
			"organization_id": thr.OrganizationID,
			"thread_id":       req.ThreadID,
		},
	})
}

func buildMessageTitleBasedOnDestinations(
	req *threading.PostMessageRequest,
	dests []endpointInput,
	thr *threading.Thread,
	fromEntity, primaryEntity *directory.Entity,
) (bml.BML, error) {
	var title bml.BML
	// For a message to be considered by sending externally it needs to marked as not internal,
	// sent by someone who is internal, and there needs to be a primary entity on the thread.
	isExternal := !req.Message.Internal && thr.PrimaryEntityID != "" && fromEntity.Type == directory.EntityType_INTERNAL && primaryEntity.Type == directory.EntityType_EXTERNAL
	if isExternal && len(dests) != 0 {
		destSet := make(map[string]struct{}, len(dests))
		for _, d := range dests {
			var ct directory.ContactType
			var ec threading.Endpoint_Channel
			switch d.Channel {
			case models.EndpointChannelEmail:
				ct = directory.ContactType_EMAIL
				ec = threading.ENDPOINT_CHANNEL_EMAIL
			case models.EndpointChannelSMS:
				ct = directory.ContactType_PHONE
				ec = threading.ENDPOINT_CHANNEL_SMS
			default:
				return nil, fmt.Errorf("unsupported destination endpoint channel %q", d.Channel)
			}
			var e *threading.Endpoint
			// Assert that the provided destination matches one of the contacts for the primary entity on the thread
			for _, c := range primaryEntity.Contacts {
				if c.ContactType == ct && c.Value == d.ID {
					e = &threading.Endpoint{
						Channel: ec,
						ID:      c.Value,
					}
					break
				}
			}
			if e == nil {
				return nil, fmt.Errorf("The provided destination contact info does not belong to the primary entity for this thread: %q, %q", d.Channel, d.ID)
			}
			req.Message.Destinations = append(req.Message.Destinations, e)
			switch e.Channel {
			case threading.ENDPOINT_CHANNEL_SMS:
				destSet["SMS"] = struct{}{}
			case threading.ENDPOINT_CHANNEL_EMAIL:
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
	} else if req.Message.Internal {
		title = append(title[:0], "Internal")
	}
	return title, nil
}

type errInvalidAttachment string

func (e errInvalidAttachment) Error() string {
	return string(e)
}

func processIncomingAttachments(ctx context.Context, ram raccess.ResourceAccessor, svc *service, ent *directory.Entity, orgID string, attachs []attachmentInput, thread *threading.Thread) ([]*threading.Attachment, []*care.CarePlan, error) {
	// Need to track the care plans so we can flag them as submitted after posting
	var carePlans []*care.CarePlan
	attachments := make([]*threading.Attachment, 0, len(attachs))
	for _, mAttachment := range attachs {
		// Backfill the attachmentID from the deprecated mediaID
		if mAttachment.MediaID != "" {
			mAttachment.AttachmentID = mAttachment.MediaID
		}

		// TODO: Verify that the media at the ID exists

		if thread != nil && !allowAttachment(thread, mAttachment.Type) {
			return nil, nil, errors.ErrNotSupported(ctx, fmt.Errorf("Cannot attach %s to thread of type %s", mAttachment.Type, thread.Type))
		}

		var attachment *threading.Attachment
		switch mAttachment.Type {
		case attachmentTypeVisit:
			if mAttachment.AttachmentID == "" {
				return nil, nil, errors.Errorf("Missing ID for visit attachment")
			}

			// ensure that the visit layout exists from which to create a visit
			visitLayoutRes, err := ram.VisitLayout(ctx, &layout.GetVisitLayoutRequest{
				ID: mAttachment.AttachmentID,
			})
			if err != nil {
				return nil, nil, err
			}

			// create the visit from the visit layout
			createVisitReq := &care.CreateVisitRequest{
				Name:            visitLayoutRes.VisitLayout.Name,
				LayoutVersionID: visitLayoutRes.VisitLayout.Version.ID,
				CreatorID:       ent.ID,
			}
			if thread != nil {
				createVisitReq.EntityID = thread.PrimaryEntityID
				createVisitReq.OrganizationID = thread.OrganizationID
			} else {
				createVisitReq.EntityID = ent.ID
				createVisitReq.OrganizationID = orgID
			}
			createVisitRes, err := ram.CreateVisit(ctx, createVisitReq)
			if err != nil {
				return nil, nil, err
			}

			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     createVisitRes.Visit.Name,
				Data: &threading.Attachment_Visit{
					Visit: &threading.VisitAttachment{
						VisitID:   createVisitRes.Visit.ID,
						VisitName: createVisitRes.Visit.Name,
					},
				},
			}
			if thread != nil {
				attachment.URL = deeplink.VisitURL(svc.webDomain, thread.ID, createVisitRes.Visit.ID)
			}
		case attachmentTypeCarePlan:
			// Make sure the care plan exists, the poster has access to it, and it hasn't yet been submitted
			cp, err := ram.CarePlan(ctx, mAttachment.AttachmentID)
			if err != nil {
				return nil, nil, err
			}
			if cp.Submitted {
				return nil, nil, errInvalidAttachment("The attached care plan has already been submitted.")
			}
			carePlans = append(carePlans, cp)

			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     cp.Name,
				Data: &threading.Attachment_CarePlan{
					CarePlan: &threading.CarePlanAttachment{
						CarePlanID:   cp.ID,
						CarePlanName: cp.Name,
					},
				},
			}
			if thread != nil {
				attachment.URL = deeplink.CarePlanURL(svc.webDomain, thread.ID, cp.ID)
			}
		case attachmentTypeImage:
			info, err := ram.MediaInfo(ctx, mAttachment.AttachmentID)
			if err != nil {
				return nil, nil, fmt.Errorf("Error while locating media info for %s: %s", mAttachment.AttachmentID, err)
			}
			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     mAttachment.Title,
				URL:       info.ID,
				Data: &threading.Attachment_Image{
					Image: &threading.ImageAttachment{
						Mimetype: info.MIME.Type + "/" + info.MIME.Subtype,
						MediaID:  info.ID,
					},
				},
			}
		case attachmentTypeAudio:
			info, err := ram.MediaInfo(ctx, mAttachment.AttachmentID)
			if err != nil {
				return nil, nil, fmt.Errorf("Error while locating media info for %s: %s", mAttachment.AttachmentID, err)
			}
			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     mAttachment.Title,
				URL:       info.ID,
				Data: &threading.Attachment_Audio{
					Audio: &threading.AudioAttachment{
						Mimetype:   info.MIME.Type + "/" + info.MIME.Subtype,
						MediaID:    info.ID,
						DurationNS: info.DurationNS,
					},
				},
			}
		case attachmentTypeVideo:
			info, err := ram.MediaInfo(ctx, mAttachment.AttachmentID)
			if err != nil {
				return nil, nil, fmt.Errorf("Error while locating media info for %s: %s", mAttachment.AttachmentID, err)
			}
			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     mAttachment.Title,
				URL:       info.ID,
				Data: &threading.Attachment_Video{
					Video: &threading.VideoAttachment{
						Mimetype:   info.MIME.Type + "/" + info.MIME.Subtype,
						MediaID:    info.ID,
						DurationNS: info.DurationNS,
					},
				},
			}
		case attachmentTypeDocument:
			info, err := ram.MediaInfo(ctx, mAttachment.AttachmentID)
			if err != nil {
				return nil, nil, fmt.Errorf("Error while locating media info for %s : %s", mAttachment.AttachmentID, err)
			}
			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     mAttachment.Title,
				URL:       info.ID,
				Data: &threading.Attachment_Document{
					Document: &threading.DocumentAttachment{
						Mimetype: media.MIMEType(info.MIME),
						MediaID:  info.ID,
						Name:     mAttachment.Title,
					},
				},
			}
		case attachmentTypePaymentRequest:
			resp, err := ram.Payment(ctx, &payments.PaymentRequest{
				PaymentID: mAttachment.AttachmentID,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("Error while locating payment info for %s: %s", mAttachment.AttachmentID, err)
			}
			attachment = &threading.Attachment{
				ContentID: mAttachment.AttachmentID,
				UserTitle: mAttachment.Title,
				Title:     payments.FormatAmount(resp.Payment.Amount, "USD"),
				// TODO: Deep link to payment
				Data: &threading.Attachment_PaymentRequest{
					PaymentRequest: &threading.PaymentRequestAttachment{
						PaymentID: resp.Payment.ID,
					},
				},
			}
		default:
			return nil, nil, fmt.Errorf("Unknown message attachment type %s", mAttachment.Type)
		}
		attachments = append(attachments, attachment)
	}
	return attachments, carePlans, nil
}
