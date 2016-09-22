package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

const newMessageNotificationKey = "new_message" // This is used for both collapse and dedupe

// TODO: mraines: This code is spagetti poo (I wrote it), refactor
func (s *threadsServer) notifyMembersOfPublishMessage(
	ctx context.Context,
	orgID string,
	savedQueryID models.SavedQueryID,
	thread *models.Thread,
	message *models.ThreadItem,
	publishingEntityID string,
	effectedEntityShouldBeNotified map[string]bool,
) {
	messageID := message.ID
	if s.notificationClient == nil || s.directoryClient == nil {
		golog.Debugf("Member notification aborted because either notification client or directory client is not configured")
		return
	}
	if orgID == "" || thread == nil || !messageID.IsValid {
		golog.Errorf("Invalid message information for notification: %v, %v, %v, %v", orgID, savedQueryID, thread, messageID)
		return
	}
	golog.Debugf("Notifying members of org %s of activity on thread %s by entity %s", orgID, thread.ID, publishingEntityID)
	conc.Go(func() {
		ctx = context.Background()

		threadEntities, err := s.dal.EntitiesForThread(ctx, thread.ID)
		if err != nil {
			golog.Errorf("Failed to get entities for thread %s: %s", thread.ID, err)
			return
		}

		threadMemberEntIDs := make([]string, 0, len(threadEntities))
		for _, te := range threadEntities {
			if te.Member {
				threadMemberEntIDs = append(threadMemberEntIDs, te.EntityID)
			}
		}

		receiverEntities, err := s.resolveInternalEntities(ctx, threadMemberEntIDs)
		if err != nil {
			golog.Errorf("Failed to resolve internal entities for ids %v: %s", threadMemberEntIDs, err)
			return
		}

		mentionedEntityIDs := getReferencedEntities(ctx, thread, message)

		// Track the messages we want to send and how many unread threads there were
		messages := make(map[string]string)
		receiverEntityIDs := make([]string, 0, len(receiverEntities))
		for _, e := range receiverEntities {
			if e.ID != publishingEntityID {
				receiverEntityIDs = append(receiverEntityIDs, e.ID)
			}
		}
		// If this is a secure external thread, then also notify the primary entity if the thread item is not internal
		if thread.Type == models.ThreadTypeSecureExternal && !message.Internal && thread.PrimaryEntityID != publishingEntityID {
			receiverEntityIDs = append(receiverEntityIDs, thread.PrimaryEntityID)
		}

		teMap := make(map[string]*models.ThreadEntity, len(threadEntities))
		for _, te := range threadEntities {
			teMap[te.EntityID] = te
		}

		// Get the unread and notification information
		if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
			// Update the memberships for everyone who needs to be notified
			// Note: It takes human interaction for this update state to trigger so shouldn't be too often.
			now := s.clk.Now()
			for _, entID := range receiverEntityIDs {
				if entID == publishingEntityID {
					continue
				}

				te := teMap[entID]
				if effectedEntityShouldBeNotified[entID] || entID == thread.PrimaryEntityID {
					if _, ok := mentionedEntityIDs[entID]; ok {
						messages[entID] = "You have a new mention in a thread"
					} else if s.isAlertAllMessagesEnabled(ctx, entID) {
						messages[entID] = s.getNotificationText(ctx, thread, message, entID)
					} else if te == nil || te.LastUnreadNotify == nil || (te.LastViewed != nil && te.LastViewed.After(*te.LastUnreadNotify)) {
						// Only send a notification if no notification has been sent or the person has viewed the thread since the last notification
						if err := dl.UpdateThreadEntity(ctx, thread.ID, entID, &dal.ThreadEntityUpdate{
							LastUnreadNotify: &now,
						}); err != nil {
							return errors.Trace(err)
						}
						messages[entID] = s.getNotificationText(ctx, thread, message, entID)
					}
				}
			}
			return nil
		}); err != nil {
			golog.Errorf("Encountered error while calculating and updating unread and notify status: %s", err)
			return
		}

		var nType notification.Type
		switch thread.Type {
		case models.ThreadTypeExternal, models.ThreadTypeSecureExternal:
			nType = notification.NewMessageOnExternalThread
		case models.ThreadTypeTeam, models.ThreadTypeLegacyTeam:
			nType = notification.NewMessageOnInternalThread
		}

		// Note: We always send the unread push to all interested entities.
		//   This is because clients rely on the push to update state.
		//   An empty ShortMessage for an entity indicated that a notification
		//   should be sent silently or flagged as such.
		//   Notifications with ShortMEssages for the entity will be displayed to the user
		if err := s.notificationClient.SendNotification(&notification.Notification{
			// UnreadCounts:     unreadCounts, TODO: currently don't support counts
			ShortMessages:    messages,
			OrganizationID:   orgID,
			SavedQueryID:     savedQueryID.String(),
			ThreadID:         thread.ID.String(),
			MessageID:        messageID.String(),
			EntitiesToNotify: receiverEntityIDs,
			// Note: Parameterizing with these may not be the best. The notification infterface needs to be
			//   rethought, but going with this for now
			DedupeKey:            newMessageNotificationKey,
			CollapseKey:          newMessageNotificationKey,
			EntitiesAtReferenced: mentionedEntityIDs,
			Type:                 nType,
		}); err != nil {
			golog.Errorf("Failed to notify members: %s", err)
		}
	})
}

func (s *threadsServer) getNotificationText(ctx context.Context, thread *models.Thread, message *models.ThreadItem, receiverEntityID string) string {
	notificationText := "You have a new message"
	isClearText := s.isClearTextMessageNotificationsEnabled(ctx, thread.Type, receiverEntityID)
	if isClearText {
		if message.Type == models.ItemTypeMessage {
			// TODO: Optimizatoin: Refactor and merge the converion of the data to models.Message for use by both notification text and refs
			msg, ok := message.Data.(*models.Message)
			if !ok {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s", message.ID)
				return notificationText
			}
			bmlText, err := bml.Parse(msg.Text)
			if err != nil {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s: %s", message.ID, err)
				return notificationText
			}
			plainText, err := bmlText.PlainText()
			if err != nil {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s: %s", message.ID, err)
				return notificationText
			}
			notificationText = plainText
			if thread.Type != models.ThreadTypeSupport {
				if thread.UserTitle != "" {
					notificationText = thread.UserTitle + ": " + notificationText
				} else {
					notificationText = thread.SystemTitle + ": " + notificationText
				}
			}
			if len(notificationText) > 256 {
				notificationText = textutil.TruncateUTF8(notificationText, 253) + "..."
			}
			return notificationText
		}
	}
	return notificationText
}

func (s *threadsServer) isAlertAllMessagesEnabled(ctx context.Context, entityID string) bool {
	booleanValue, err := settings.GetBooleanValue(ctx, s.settingsClient, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.AlertAllMessages}},
		NodeID: entityID,
	})
	if err != nil {
		golog.Errorf("Encountered an error when getting AlertAllMessages for entity %s: %s", entityID, err)
		return true
	}

	return booleanValue.Value
}

func (s *threadsServer) isClearTextMessageNotificationsEnabled(ctx context.Context, threadType models.ThreadType, receiverEntityID string) bool {
	var key string
	switch threadType {
	case models.ThreadTypeSecureExternal, models.ThreadTypeExternal:
		key = threading.PreviewPatientMessageContentInNotification
	case models.ThreadTypeTeam, models.ThreadTypeLegacyTeam:
		key = threading.PreviewTeamMessageContentInNotification
	case models.ThreadTypeSupport, models.ThreadTypeSetup:
		return true
	default:
		return false
	}

	booleanValue, err := settings.GetBooleanValue(ctx, s.settingsClient, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: key}},
		NodeID: receiverEntityID,
	})
	if err != nil {
		golog.Errorf("Encountered an error when getting %s for org %s: %s", key, receiverEntityID, err)
		return false
	}

	return booleanValue.Value
}
