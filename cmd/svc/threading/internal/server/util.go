package server

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

func mediaIDsFromAttachments(as []*models.Attachment) []string {
	mediaIDs := make([]string, 0, len(as))
	for _, a := range as {
		switch a.Type {
		case models.ATTACHMENT_TYPE_AUDIO:
			mediaIDs = append(mediaIDs, a.GetAudio().MediaID)
		case models.ATTACHMENT_TYPE_IMAGE:
			mediaIDs = append(mediaIDs, a.GetImage().MediaID)
		case models.ATTACHMENT_TYPE_VIDEO:
			mediaIDs = append(mediaIDs, a.GetVideo().MediaID)
		}
	}
	return mediaIDs
}

func paymentsIDsFromAttachments(as []*models.Attachment) []string {
	paymentIDs := make([]string, 0, len(as))
	for _, a := range as {
		switch a.Type {
		case models.ATTACHMENT_TYPE_PAYMENT_REQUEST:
			paymentIDs = append(paymentIDs, a.GetPaymentRequest().PaymentID)
		}
	}
	return paymentIDs
}

func memberEntityIDsForNewThread(ttype threading.ThreadType, orgID, fromEntityID string, memberEntityIDs []string) ([]string, error) {
	switch ttype {
	case threading.THREAD_TYPE_EXTERNAL, threading.THREAD_TYPE_SECURE_EXTERNAL, threading.THREAD_TYPE_SETUP, threading.THREAD_TYPE_SUPPORT:
		// Make sure org is a member
		memberEntityIDs = appendStringToSet(memberEntityIDs, orgID)
	case threading.THREAD_TYPE_TEAM:
		// Make sure creator is a member
		if fromEntityID != "" {
			memberEntityIDs = appendStringToSet(memberEntityIDs, fromEntityID)
		}
	default:
		return nil, grpcErrorf(codes.Internal, fmt.Sprintf("Unhandled thread type %s", ttype))
	}
	return memberEntityIDs, nil
}

func getReferencedEntities(ctx context.Context, thread *models.Thread, message *models.ThreadItem) map[string]struct{} {
	referencedEntityIDs := make(map[string]struct{})
	if message.Type == models.ItemTypeMessage {
		// TODO: Optimizatoin: Refactor and merge the converion of the data to models.Message for use by both notification text and refs
		msg, ok := message.Data.(*models.Message)
		if !ok {
			golog.Errorf("Failed to convert thread item data to message for referenced entities for item id %s", message.ID)
			return referencedEntityIDs
		}
		for _, ref := range msg.TextRefs {
			if ref.Type == models.REFERENCE_TYPE_ENTITY {
				referencedEntityIDs[ref.ID] = struct{}{}
			}
		}
	}
	return referencedEntityIDs
}

func parseRefsAndNormalize(s string) (string, []*models.Reference, error) {
	if s == "" {
		return "", nil, nil
	}
	b, err := bml.Parse(s)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	var refs []*models.Reference
	for _, e := range b {
		if r, ok := e.(*bml.Ref); ok {
			switch r.Type {
			case bml.EntityRef:
				refs = append(refs, &models.Reference{
					ID:   r.ID,
					Type: models.REFERENCE_TYPE_ENTITY,
				})
			default:
				return "", nil, errors.Trace(fmt.Errorf("unknown reference type %s", r.Type))
			}
		}
	}
	s, err = b.Format()
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return s, refs, nil
}

func internalError(err error) error {
	golog.LogDepthf(-1, golog.ERR, err.Error())
	return grpcErrorf(codes.Internal, errors.Trace(err).Error())
}

// appendStringToSet appends a string to the slice if it's not already included
func appendStringToSet(set []string, s string) []string {
	for _, id := range set {
		if id == s {
			return set
		}
	}
	return append(set, s)
}
