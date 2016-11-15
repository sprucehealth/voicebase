package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type cloneMessageInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	OrganizationID   string `gql:"organizationID"`
	SourceMessageID  string `gql:"sourceMessageID"`
	ForThreadID      string `gql:"forThreadID"`
}

var cloneMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CloneMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"sourceMessageID": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "ID of either a tream item or saved message",
			},
			"organizationID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"forThreadID": &graphql.InputObjectFieldConfig{
				Type:        graphql.ID,
				Description: "Optional ID of thread for which the message is intended. If provided the attachment are filtered to the supported set and alerts are returned for any removed attachments.",
			},
		},
	},
)

const cloneMessageErrorCodeSourceMessageDoesNotExist = "SOURCE_MESSAGE_DOES_NOT_EXIST"

var cloneMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CloneMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		cloneMessageErrorCodeSourceMessageDoesNotExist: &graphql.EnumValueConfig{
			Value:       cloneMessageErrorCodeSourceMessageDoesNotExist,
			Description: "Soruce message does not exist",
		},
	},
})

type cloneMessageOutput struct {
	ClientMutationID string             `json:"clientMutationId,omitempty"`
	Success          bool               `json:"success"`
	ErrorCode        string             `json:"errorCode,omitempty"`
	ErrorMessage     string             `json:"errorMessage,omitempty"`
	ThreadItem       *models.ThreadItem `json:"threadItem,omitempty"`
	Alerts           []string           `json:"alerts,omitempty"`
}

var cloneMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CloneMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: cloneMessageErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"threadItem":       &graphql.Field{Type: threadItemType},
			"alerts":           &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*cloneMessageOutput)
			return ok
		},
	},
)

var cloneMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(cloneMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(cloneMessageInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		acc := gqlctx.Account(ctx)

		var in cloneMessageInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, in.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, in.OrganizationID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var thread *threading.Thread
		if in.ForThreadID != "" {
			thread, err = ram.Thread(ctx, in.ForThreadID, ent.ID)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		newItem := &threading.ThreadItem{
			ActorEntityID:     ent.ID,
			OrganizationID:    in.OrganizationID,
			CreatedTimestamp:  uint64(time.Now().Unix()),
			ModifiedTimestamp: uint64(time.Now().Unix()),
		}

		var msg *threading.Message
		if strings.HasPrefix(in.SourceMessageID, "ti_") {
			item, err := ram.ThreadItem(ctx, in.SourceMessageID)
			if grpc.Code(err) == codes.NotFound {
				return nil, errors.ErrNotFound(ctx, in.SourceMessageID)
			} else if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if _, ok := item.Item.(*threading.ThreadItem_Message); !ok {
				return nil, errors.ErrNotSupported(ctx, fmt.Errorf("Cannot clone item of type %T", item.Item))
			}
			// TODO: for now don't allow cloning across organizations
			if item.OrganizationID != in.OrganizationID {
				return nil, errors.ErrNotAuthorized(ctx, item.ID)
			}
			if item.Deleted {
				return nil, errors.ErrNotSupported(ctx, errors.New("Cannot clone a delete item"))
			}
			newItem.Internal = item.Internal
			msg = item.GetMessage()
		} else {
			res, err := ram.SavedMessages(ctx, &threading.SavedMessagesRequest{
				By: &threading.SavedMessagesRequest_IDs{
					IDs: &threading.IDList{
						IDs: []string{in.SourceMessageID},
					},
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if len(res.SavedMessages) == 0 {
				return nil, errors.ErrNotFound(ctx, in.SourceMessageID)
			}
			sm := res.SavedMessages[0]
			if sm.OwnerEntityID != ent.ID && sm.OwnerEntityID != in.OrganizationID {
				return nil, errors.ErrNotAuthorized(ctx, sm.ID)
			}
			newItem.Internal = sm.Internal
			msg = sm.GetMessage()
		}

		newMsg := &threading.Message{
			Text:     msg.Text,
			TextRefs: msg.TextRefs,
			Summary:  msg.Summary,
			Title:    msg.Title,
		}
		clonedAttachments, unsupportedAttachments, err := cloneAttachments(ctx, ram, ent, in.OrganizationID, msg.Attachments, thread)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		newMsg.Attachments = clonedAttachments
		newItem.Item = &threading.ThreadItem_Message{
			Message: newMsg,
		}

		// If any attachments were stripped then return an alert
		var alerts []string
		if len(unsupportedAttachments) != 0 {
			// TODO: right now this just uses the raw attachment type, we should make the message nicer
			atypes := make([]string, len(unsupportedAttachments))
			for i, a := range unsupportedAttachments {
				atypes[i], err = attachmentTypeAsEnum(a)
				if err != nil {
					// This shouldn't ever happen but handle it anyway
					golog.Errorf(err.Error())
					atypes[i] = fmt.Sprintf("%T", a.Data)
				}
			}
			alerts = []string{"The following attachments are not supported for this thread and have been removed: %s", strings.Join(atypes, ", ")}
		}

		rti, err := transformThreadItemToResponse(newItem, "", svc.webDomain, svc.mediaAPIDomain)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return &cloneMessageOutput{
			Success:    true,
			ThreadItem: rti,
			Alerts:     alerts,
		}, nil
	}),
}

func cloneAttachments(ctx context.Context, ram raccess.ResourceAccessor, ent *directory.Entity, orgID string, attachments []*threading.Attachment, forThread *threading.Thread) (cloned []*threading.Attachment, unsupported []*threading.Attachment, err error) {
	if ent.AccountID == "" {
		return nil, nil, errors.Errorf("entity %s missing account ID", ent.ID)
	}
	acc := gqlctx.Account(ctx)
	newAtts := make([]*threading.Attachment, 0, len(attachments))
	var unsupportedAttachments []*threading.Attachment
	par := conc.NewParallel()
	for _, att := range attachments {
		atype, err := attachmentTypeAsEnum(att)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		if forThread != nil && !allowAttachment(forThread, atype) {
			unsupportedAttachments = append(unsupportedAttachments, att)
			continue
		}
		newAtt := &threading.Attachment{}
		*newAtt = *att
		newAtts = append(newAtts, newAtt)
		par.Go(func() error {
			switch a := newAtt.Data.(type) {
			case *threading.Attachment_Image:
				res, err := ram.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: media.MediaOwnerType_ACCOUNT, OwnerID: ent.AccountID, MediaID: a.Image.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Image.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Video:
				res, err := ram.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: media.MediaOwnerType_ACCOUNT, OwnerID: ent.AccountID, MediaID: a.Video.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Video.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Audio:
				res, err := ram.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: media.MediaOwnerType_ACCOUNT, OwnerID: ent.AccountID, MediaID: a.Audio.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Audio.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Document:
				res, err := ram.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: media.MediaOwnerType_ACCOUNT, OwnerID: ent.AccountID, MediaID: a.Document.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Document.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Visit:
				res, err := ram.Visit(ctx, &care.GetVisitRequest{ID: a.Visit.VisitID})
				if err != nil {
					return errors.Trace(err)
				}
				vres, err := ram.VisitLayoutByVersion(ctx, &layout.GetVisitLayoutByVersionRequest{
					VisitLayoutVersionID: res.Visit.LayoutVersionID,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = vres.VisitLayout.ID
			case *threading.Attachment_PaymentRequest:
				pres, err := ram.Payment(ctx, &payments.PaymentRequest{
					PaymentID: a.PaymentRequest.PaymentID,
				})
				if err != nil {
					return errors.Trace(err)
				}
				p := pres.Payment
				res, err := ram.CreatePayment(ctx, &payments.CreatePaymentRequest{
					RequestingEntityID: orgID,
					Amount:             p.Amount,
					Currency:           p.Currency,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.Payment.ID
			case *threading.Attachment_CarePlan:
				cp, err := ram.CarePlan(ctx, a.CarePlan.CarePlanID)
				if err != nil {
					return errors.Trace(err)
				}
				res, err := ram.CreateCarePlan(ctx, &care.CreateCarePlanRequest{
					Name:         cp.Name,
					CreatorID:    acc.ID,
					Instructions: cp.Instructions,
					Treatments:   cp.Treatments,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.CarePlan.ID
				a.CarePlan.CarePlanID = res.CarePlan.ID
			default:
				return errors.Errorf("unknown attachment type %T", newAtt.Data)
			}
			return nil
		})
	}
	if err := par.Wait(); err != nil {
		return nil, nil, errors.Trace(err)
	}
	return newAtts, unsupportedAttachments, nil
}
