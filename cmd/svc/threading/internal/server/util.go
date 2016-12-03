package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func processMessagePost(msg *threading.MessagePost, allowEmptySummary bool) ([]*models.Reference, error) {
	if msg == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Message is required")
	}
	if !allowEmptySummary && msg.Summary == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Summary is required")
	}
	msg.Summary = textutil.TruncateUTF8(msg.Summary, maxSummaryLength)
	if msg.Title != "" {
		if _, err := bml.Parse(msg.Title); err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Title is invalid format: %s", err.Error())
		}
	}
	var err error
	var textRefs []*models.Reference
	msg.Text, textRefs, err = parseRefsAndNormalize(msg.Text)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Text is invalid format: %s", errors.Cause(err).Error())
	}
	return textRefs, nil
}

func validateTags(tags []string) (string, bool) {
	for _, t := range tags {
		if !threading.ValidateTag(t, true) {
			return t, false
		}
	}
	return "", true
}

// validateEntityIDs makes sure a list of IDs are valid entity IDs. If one is not then
// it returns the bad id and false. Otherwise it returns an emptry string anf true.
func validateEntityIDs(ids []string) (string, bool) {
	for _, id := range ids {
		if !strings.HasPrefix(id, directory.EntityIDPrefix) {
			return id, false
		}
	}
	return "", true
}

// threadMatchesQuery returns true iff the thread matches the provided query for the entity
func threadMatchesQuery(q *models.Query, t *models.Thread, te *models.ThreadEntity, externalEntity bool) (bool, error) {
	// For efficiency with multiple tokens generate the full set of text to match against using
	// a delimiter that's very unlikely to be found in a token expression.
	var fullText string
	if externalEntity {
		fullText = t.SystemTitle + "⇄" + t.UserTitle + "⇄" + t.LastExternalMessageSummary
	} else {
		fullText = t.SystemTitle + "⇄" + t.UserTitle + "⇄" + t.LastMessageSummary
	}
	fullText = strings.ToLower(fullText)
	for _, e := range q.Expressions {
		switch v := e.Value.(type) {
		case *models.Expr_Flag_:
			switch v.Flag {
			case models.EXPR_FLAG_UNREAD:
				if isUnread(t, te, externalEntity) == e.Not {
					return false, nil
				}
			case models.EXPR_FLAG_UNREAD_REFERENCE:
				if hasUnreadReference(te) == e.Not {
					return false, nil
				}
			case models.EXPR_FLAG_FOLLOWING:
				if (te == nil || !te.Following) != e.Not {
					return false, nil
				}
			default:
				return false, errors.Errorf("unknown expression flag %s", v.Flag)
			}
		case *models.Expr_ThreadType_:
			switch v.ThreadType {
			case models.EXPR_THREAD_TYPE_PATIENT:
				if (t.Type != models.ThreadTypeExternal && t.Type != models.ThreadTypeSecureExternal) != e.Not {
					return false, nil
				}
			case models.EXPR_THREAD_TYPE_TEAM:
				if (t.Type != models.ThreadTypeTeam) != e.Not {
					return false, nil
				}
			case models.EXPR_THREAD_TYPE_SUPPORT:
				if (t.Type != models.ThreadTypeSupport && t.Type != models.ThreadTypeSetup) != e.Not {
					return false, nil
				}
			case models.EXPR_THREAD_TYPE_PATIENT_SECURE:
				if (t.Type != models.ThreadTypeSecureExternal) != e.Not {
					return false, nil
				}
			case models.EXPR_THREAD_TYPE_PATIENT_STANDARD:
				if (t.Type != models.ThreadTypeExternal) != e.Not {
					return false, nil
				}

			default:
				return false, errors.Errorf("unknown expression thread type %s", v.ThreadType)
			}
		case *models.Expr_Tag:
			hasTag := false
			for _, tag := range t.Tags {
				if strings.EqualFold(v.Tag, tag.Name) {
					hasTag = true
					break
				}
			}
			if hasTag == e.Not {
				return false, nil
			}
		case *models.Expr_Token:
			if strings.Contains(fullText, strings.ToLower(v.Token)) == e.Not {
				return false, nil
			}
		default:
			return false, errors.Errorf("unknown expression value type %T", e.Value)
		}
	}
	return true, nil
}

func mediaIDsFromAttachments(as []*models.Attachment) []string {
	mediaIDs := make([]string, 0, len(as))
	for _, a := range as {
		switch a := a.Data.(type) {
		case *models.Attachment_Audio:
			mediaIDs = append(mediaIDs, a.Audio.MediaID)
		case *models.Attachment_Image:
			mediaIDs = append(mediaIDs, a.Image.MediaID)
		case *models.Attachment_Video:
			mediaIDs = append(mediaIDs, a.Video.MediaID)
		case *models.Attachment_Document:
			mediaIDs = append(mediaIDs, a.Document.MediaID)
		}
	}
	return mediaIDs
}

func paymentsIDsFromAttachments(as []*models.Attachment) []string {
	paymentIDs := make([]string, 0, len(as))
	for _, a := range as {
		switch a := a.Data.(type) {
		case *models.Attachment_PaymentRequest:
			paymentIDs = append(paymentIDs, a.PaymentRequest.PaymentID)
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
		return nil, grpc.Errorf(codes.Internal, fmt.Sprintf("Unhandled thread type %s", ttype))
	}
	return memberEntityIDs, nil
}

func getReferencedEntities(ctx context.Context, thread *models.Thread, item *models.ThreadItem) map[string]struct{} {
	referencedEntityIDs := make(map[string]struct{})
	if msg, ok := item.Data.(*models.Message); ok {
		// TODO: Optimization: Refactor and merge the converion of the data to models.Message for use by both notification text and refs
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
				return "", nil, errors.Errorf("unknown reference type %s", r.Type)
			}
		}
	}
	s, err = b.Format()
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return s, refs, nil
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

// isUnread returns true iff the thread has not been read by the entity
func isUnread(t *models.Thread, te *models.ThreadEntity, externalEntity bool) bool {
	// Threads without message are never unread
	if t.MessageCount == 0 {
		return false
	}
	if te == nil || te.LastViewed == nil {
		return true
	}
	if externalEntity {
		return te.LastViewed.Before(t.LastExternalMessageTimestamp.Truncate(time.Second))
	}
	return te.LastViewed.Before(t.LastMessageTimestamp.Truncate(time.Second))
}

// hasUnreadReference returns true iff the entity has an unread reference
func hasUnreadReference(te *models.ThreadEntity) bool {
	if te == nil || te.LastReferenced == nil {
		return false
	}
	if te.LastViewed == nil {
		return true
	}
	return te.LastViewed.Before(te.LastReferenced.Truncate(time.Second))
}

func isExternalEntity(e *directory.Entity) bool {
	return e.Type == directory.EntityType_PATIENT || e.Type == directory.EntityType_EXTERNAL
}

// NOTE: This should remain idempotent since it is called for both scheduling and posting a message
func claimAttachments(ctx context.Context, mediaClient media.MediaClient, paymentsClient payments.PaymentsClient, threadID models.ThreadID, attachments []*models.Attachment) error {
	mediaIDs := mediaIDsFromAttachments(attachments)
	if len(mediaIDs) > 0 {
		// Before posting the actual message, map all the attached media to the thread
		// Failure scenarios:
		// 1. This call succeeds and the post fails. The media is now mapped to the thread which should still allow a repost.
		// 2. This call fails. The media is still mapped to the caller
		_, err := mediaClient.ClaimMedia(ctx, &media.ClaimMediaRequest{
			MediaIDs:  mediaIDs,
			OwnerType: media.MediaOwnerType_THREAD,
			OwnerID:   threadID.String(),
		})
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, pID := range paymentsIDsFromAttachments(attachments) {
		// This call should be idempotent as long as the payment request is just being submitted
		if _, err := paymentsClient.SubmitPayment(ctx, &payments.SubmitPaymentRequest{
			PaymentID: pID,
			ThreadID:  threadID.String(),
		}); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func createPostMessageRequest(ctx context.Context, threadID models.ThreadID, fromEntityID string, postMessage *threading.MessagePost) (*dal.PostMessageRequest, error) {
	textRefs, err := processMessagePost(postMessage, false)
	if err != nil {
		return nil, err
	}

	// TODO: validate any attachments
	attachments, err := transformAttachmentsFromRequest(postMessage.Attachments)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var source *models.Endpoint
	if postMessage.Source != nil {
		source, err = transformEndpointFromRequest(postMessage.Source)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	destinations := make([]*models.Endpoint, 0, len(postMessage.Destinations))
	for _, dc := range postMessage.Destinations {
		d, err := transformEndpointFromRequest(dc)
		if err != nil {
			return nil, errors.Trace(err)
		}
		destinations = append(destinations, d)
	}

	return &dal.PostMessageRequest{
		ThreadID:     threadID,
		FromEntityID: fromEntityID,
		Internal:     postMessage.Internal,
		Text:         postMessage.Text,
		Title:        postMessage.Title,
		TextRefs:     textRefs,
		Summary:      postMessage.Summary,
		Attachments:  attachments,
		Source:       source,
		Destinations: destinations,
	}, nil
}
