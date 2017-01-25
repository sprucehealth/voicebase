package cleaner

import (
	"encoding/json"
	"net/http"
	"time"

	"context"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/transcription"
	"github.com/sprucehealth/backend/libs/twilio"
)

type Worker struct {
	twilio        *twilio.Client
	dal           dal.DAL
	sqs           sqsiface.SQSAPI
	cleanupWorker *awsutil.SQSWorker
	// TODO connect the voicebase client here directly
	transcriptionProvider transcription.Provider
}

type snsMessage struct {
	Message []byte
}

func NewWorker(
	twilio *twilio.Client,
	dal dal.DAL,
	sqs sqsiface.SQSAPI,
	transcriptionProvider transcription.Provider,
	cleanupQueueURL string) *Worker {
	w := &Worker{
		twilio: twilio,
		dal:    dal,
		sqs:    sqs,
		transcriptionProvider: transcriptionProvider,
	}
	w.cleanupWorker = awsutil.NewSQSWorker(sqs, cleanupQueueURL, w.processSNSEvent)
	return w
}

func (w *Worker) Start() {
	w.cleanupWorker.Start()
}

func (w *Worker) Stop(wait time.Duration) {
	w.Stop(wait)
}

func (w *Worker) processSNSEvent(ctx context.Context, msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err.Error())
		return nil
	}
	var drr models.DeleteResourceRequest
	if err := drr.Unmarshal(snsMsg.Message); err != nil {
		golog.Errorf("Failed to unmarshal delete resource request: %s", err.Error())
		return nil
	}

	return errors.Trace(w.processEvent(&drr))
}

func (w *Worker) processEvent(drr *models.DeleteResourceRequest) error {
	switch drr.Type {
	case models.DeleteResourceRequest_TWILIO_CALL:
		// only delete a call if it is not queued, ringing or in-progress
		call, _, err := w.twilio.Calls.Get(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			}
			return errors.Trace(err)
		}
		switch call.Status {
		case "busy", "completed", "failed", "canceled", "no-answer":
		default:
			golog.Warningf("Waiting for call %s to reach a completed state before deleting. Current status: %s", drr.ResourceID, call.Status)
			return awsutil.ErrMsgNotProcessedYet
		}
		_, err = w.twilio.Calls.Delete(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			}
			return errors.Trace(err)
		}
	case models.DeleteResourceRequest_TWILIO_MEDIA:
		_, err := w.twilio.Messages.DeleteMedia(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			}
			return errors.Trace(err)
		}
	case models.DeleteResourceRequest_TWILIO_RECORDING:
		_, err := w.twilio.Recording.Delete(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			}
			return errors.Trace(err)
		}

	case models.DeleteResourceRequest_TWILIO_SMS:
		_, err := w.twilio.Messages.Delete(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			} else if e.Status == http.StatusBadRequest {
				golog.Warningf("Unable to delete message: %s", err)
				return awsutil.ErrMsgNotProcessedYet
			}
			return errors.Trace(err)
		}
	case models.DeleteResourceRequest_TWILIO_TRANSCRIPTION:
		_, err := w.twilio.Transcription.Delete(drr.ResourceID)
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok && e.Code == twilio.ErrorCodeResourceNotFound {
				return nil
			}
			return errors.Trace(err)
		}
	case models.DeleteResourceRequest_VOICEBASE_TRANSCRIPTION:
		if err := w.transcriptionProvider.DeleteMedia(drr.ResourceID); err != nil {
			// TODO voicebase specific error handling
			return errors.Trace(err)
		}
	}

	if err := w.dal.CreateDeletedResource(drr.Type.String(), drr.ResourceID); err != nil {
		golog.Errorf("Unable to create deleted resource in database :%s", err.Error())
	}

	return nil
}
