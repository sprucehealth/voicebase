package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

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
	fullText := strings.ToLower(t.UserTitle + "⇄" + t.SystemTitle)
	for _, e := range q.Expressions {
		switch v := e.Value.(type) {
		case *models.Expr_Flag_:
			switch v.Flag {
			case models.EXPR_FLAG_UNREAD:
				if !isUnread(t, te, externalEntity) {
					return false, nil
				}
			case models.EXPR_FLAG_UNREAD_REFERENCE:
				if !hasUnreadReference(te) {
					return false, nil
				}
			case models.EXPR_FLAG_FOLLOWING:
				if te == nil || !te.Following {
					return false, nil
				}
			default:
				return false, errors.Errorf("unknown expression flag %s", v.Flag)
			}
		case *models.Expr_ThreadType_:
			switch v.ThreadType {
			case models.EXPR_THREAD_TYPE_PATIENT:
				if t.Type != models.ThreadTypeExternal && t.Type != models.ThreadTypeSecureExternal {
					return false, nil
				}
			case models.EXPR_THREAD_TYPE_TEAM:
				if t.Type != models.ThreadTypeTeam {
					return false, nil
				}
			default:
				return false, errors.Errorf("unknown expression thread type %s", v.ThreadType)
			}
		case *models.Expr_Token:
			if !strings.Contains(fullText, strings.ToLower(v.Token)) {
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
