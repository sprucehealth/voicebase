package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/smet"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var sentStatus = models.ScheduledMessageStatusSent

type scheduledMessageThreadClient interface {
	PostMessage(ctx context.Context, req *threading.PostMessageRequest, opts ...grpc.CallOption) (*threading.PostMessageResponse, error)
}

// processPaymentNoneAccepted asserts the existance of the customer and payment method in the context of the vendor
func (w *Workers) processPendingScheduledMessage() {
	ctx := context.Background()
	// Find some work to do without locking
	scheduledMessages, err := w.dal.ScheduledMessages(ctx, []models.ScheduledMessageStatus{models.ScheduledMessageStatusPending}, w.clk.Now())
	if err != nil {
		smet.Errorf(workerErrMetricName, "Encountered error looking for scheduled message work: %s", err)
		return
	}
	if len(scheduledMessages) == 0 {
		return
	}

	// Process each message individually
	for _, sm := range scheduledMessages {
		if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
			// Grab the row and lock it before we start sending
			scheduledMessage, err := dl.ScheduledMessage(ctx, sm.ID, dal.ForUpdate)
			if err != nil {
				return errors.Trace(err)
			}
			// Make sure someone else didn't already send it, if they did move on
			if scheduledMessage.Status != models.ScheduledMessageStatusPending {
				return nil
			}
			// Decode the message to build the request we'll use to post the mesage
			message, err := server.TransformMessageToResponse(scheduledMessage.Data.(*models.Message), false)
			if err != nil {
				return errors.Trace(err)
			}
			// Post the message
			resp, err := w.threadingCli.PostMessage(ctx, &threading.PostMessageRequest{
				UUID:         scheduledMessage.ID.String(),
				ThreadID:     scheduledMessage.ThreadID.String(),
				FromEntityID: scheduledMessage.ActorEntityID,
				Message: &threading.MessagePost{
					Internal:     scheduledMessage.Internal,
					Source:       message.Source,
					Destinations: message.Destinations,
					Text:         message.Text,
					Attachments:  message.Attachments,
					Title:        message.Title,
					Summary:      message.Summary,
				},
			})
			if err != nil {
				return errors.Trace(err)
			}
			// IF we fail to parse the id of the sent item just log it since we don't want to sent it twice
			threadItemID := models.EmptyThreadItemID()
			itemID, err := models.ParseThreadItemID(resp.Item.ID)
			if err != nil {
				golog.Errorf("Error parsing new thread item id after posting scheduled message: %s", errors.Trace(err))
			} else {
				threadItemID = itemID
			}
			if _, err := dl.UpdateScheduledMessage(ctx, scheduledMessage.ID, &models.ScheduledMessageUpdate{
				Status:           &sentStatus,
				SentThreadItemID: &threadItemID,
			}); err != nil {
				return errors.Trace(err)
			}
			return nil
		}); err != nil {
			smet.Errorf(workerErrMetricName, "Encountered error while processing pending scheduled messages: %s", err)
		}
	}
}
