package server

import (
	"context"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

func (s *threadsServer) CloneAttachments(ctx context.Context, in *threading.CloneAttachmentsRequest) (*threading.CloneAttachmentsResponse, error) {
	if in.OwnerID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OwnerID required")
	}
	if len(in.Attachments) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "At least 1 attachment is required")
	}
	mediaOwnerType, err := clonedAttachmentOwnerAsMediaOwner(in.OwnerType)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	newAtts := make([]*threading.Attachment, 0, len(in.Attachments))
	par := conc.NewParallel()
	for _, att := range in.Attachments {
		newAtt := &threading.Attachment{}
		*newAtt = *att
		newAtts = append(newAtts, newAtt)
		par.Go(func() error {
			switch a := newAtt.Data.(type) {
			case *threading.Attachment_Image:
				res, err := s.mediaClient.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: mediaOwnerType, OwnerID: in.OwnerID, MediaID: a.Image.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Image.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Video:
				res, err := s.mediaClient.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: mediaOwnerType, OwnerID: in.OwnerID, MediaID: a.Video.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Video.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Audio:
				res, err := s.mediaClient.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: mediaOwnerType, OwnerID: in.OwnerID, MediaID: a.Audio.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				a.Audio.MediaID = res.MediaInfo.ID
			case *threading.Attachment_Document:
				res, err := s.mediaClient.CloneMedia(ctx, &media.CloneMediaRequest{OwnerType: mediaOwnerType, OwnerID: in.OwnerID, MediaID: a.Document.MediaID})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.MediaInfo.ID
				newAtt.URL = res.MediaInfo.ID
				newAtt.Title = res.MediaInfo.Name
				a.Document.MediaID = res.MediaInfo.ID
				a.Document.Name = res.MediaInfo.Name
			case *threading.Attachment_Visit:
				res, err := s.careClient.GetVisit(ctx, &care.GetVisitRequest{ID: a.Visit.VisitID})
				if err != nil {
					return errors.Trace(err)
				}
				vres, err := s.layoutClient.GetVisitLayoutByVersion(ctx, &layout.GetVisitLayoutByVersionRequest{
					VisitLayoutVersionID: res.Visit.LayoutVersionID,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = vres.VisitLayout.ID
			case *threading.Attachment_PaymentRequest:
				pres, err := s.paymentsClient.Payment(ctx, &payments.PaymentRequest{
					PaymentID: a.PaymentRequest.PaymentID,
				})
				if err != nil {
					return errors.Trace(err)
				}
				res, err := s.paymentsClient.CreatePayment(ctx, &payments.CreatePaymentRequest{
					RequestingEntityID: pres.Payment.RequestingEntityID,
					Amount:             pres.Payment.Amount,
					Currency:           pres.Payment.Currency,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.Payment.ID
				a.PaymentRequest.PaymentID = res.Payment.ID
			case *threading.Attachment_CarePlan:
				cpres, err := s.careClient.CarePlan(ctx, &care.CarePlanRequest{ID: a.CarePlan.CarePlanID})
				if err != nil {
					return errors.Trace(err)
				}
				res, err := s.careClient.CreateCarePlan(ctx, &care.CreateCarePlanRequest{
					Name:         cpres.CarePlan.Name,
					CreatorID:    in.OwnerID,
					Instructions: cpres.CarePlan.Instructions,
					Treatments:   cpres.CarePlan.Treatments,
				})
				if err != nil {
					return errors.Trace(err)
				}
				newAtt.ContentID = res.CarePlan.ID
				a.CarePlan.CarePlanID = res.CarePlan.ID
				a.CarePlan.CarePlanName = res.CarePlan.Name
			default:
				return errors.Errorf("unknown attachment type %T", newAtt.Data)
			}
			return nil
		})
	}
	if err := par.Wait(); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CloneAttachmentsResponse{
		Attachments: newAtts,
	}, nil
}

func clonedAttachmentOwnerAsMediaOwner(ot threading.CloneAttachmentsRequest_OwnerType) (media.MediaOwnerType, error) {
	switch ot {
	case threading.CLONED_ATTACHMENT_OWNER_ACCOUNT:
		return media.MediaOwnerType_ACCOUNT, nil
	case threading.CLONED_ATTACHMENT_OWNER_TRIGGERED_MESSAGE:
		return media.MediaOwnerType_TRIGGERED_MESSAGE, nil
	case threading.CLONED_ATTACHMENT_OWNER_SAVED_MESSAGE:
		return media.MediaOwnerType_SAVED_MESSAGE, nil
	case threading.CLONED_ATTACHMENT_OWNER_THREAD:
		return media.MediaOwnerType_THREAD, nil
	case threading.CLONED_ATTACHMENT_OWNER_VISIT:
		return media.MediaOwnerType_VISIT, nil
	}
	return media.MediaOwnerType_OWNER_TYPE_UNKNOWN, errors.Errorf("Unhandled media owner type for cloned message - %s", ot)
}
