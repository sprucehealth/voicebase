package worker

import (
	"context"

	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/transcription"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/excomms"
)

type trackTranscriptionRequest struct {
	RawMessageID    uint64 `json:"raw_message_id"`
	MediaID         string `json:"media_id"`
	MediaDurationNS uint64 `json:"media_duration_ns"`
	JobID           string `json:"job_id"`
	UrgentVoicemail bool   `json:"urgent_voicemail"`
}

// transcriptionTracker is responsible for tracking transcription jobs in progress
// to push them through as a voicemail to the application when the job completes
// or is considered timed out (in which case the voicemail is pushed through without
// the transcription).
type transcriptionTracker struct {
	transcriptionProvider transcription.Provider
	snsAPI                snsiface.SNSAPI
	sqsAPI                sqsiface.SQSAPI
	externalMessageTopic  string
	resourceCleanerTopic  string
	dal                   dal.DAL
	worker                worker.Worker
}

type TranscriptionTrackingWorker interface {
	Start()
	Stop(wait time.Duration)
}

func NewTranscriptionTrackingWorker(
	transcriptionProvider transcription.Provider,
	snsAPI snsiface.SNSAPI,
	sqsAPI sqsiface.SQSAPI,
	externalMessageTopic, resourceCleanerTopic, transcriptionTrackingSQSURL string,
	dal dal.DAL) TranscriptionTrackingWorker {
	w := &transcriptionTracker{
		transcriptionProvider: transcriptionProvider,
		snsAPI:                snsAPI,
		externalMessageTopic:  externalMessageTopic,
		resourceCleanerTopic:  resourceCleanerTopic,
		dal:                   dal,
	}

	w.worker = awsutil.NewSQSWorker(sqsAPI, transcriptionTrackingSQSURL, w.processTranscription, awsutil.VisibilityTimeoutInSeconds(60))
	return w
}

func (w *transcriptionTracker) Start() {
	w.worker.Start()
}

func (w *transcriptionTracker) Stop(wait time.Duration) {
	w.worker.Stop(wait)
}

func (w *transcriptionTracker) processTranscription(ctx context.Context, data string) error {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return errors.Trace(err)
	}

	var req trackTranscriptionRequest
	if err := json.Unmarshal(decodedData, &req); err != nil {
		return errors.Trace(err)
	}

	var skip bool
	var job *models.TranscriptionJob
	if err := w.dal.Transact(func(dl dal.DAL) error {
		var err error
		job, err = dl.LookupTranscriptionJob(ctx, req.MediaID, dal.ForUpdate)
		if err != nil {
			return errors.Wrapf(err, "unable to lookup transcription job for %s", req.MediaID)
		}

		if job.Completed {
			// job completed, so nothing to do
			return nil
		}

		if time.Now().Before(job.AvailableAfter) {
			skip = true
			// job is currently in flight, cannot work on it yet
			return nil
		}

		if rowsUpdated, err := dl.UpdateTranscriptionJob(ctx, req.MediaID, &dal.TranscriptionJobUpdate{
			AvailableAfter: ptr.Time(time.Now().Add(1 * time.Minute)),
		}); err != nil {
			return errors.Wrapf(err, "unable to update transcription job for media %s", req.MediaID)
		} else if rowsUpdated > 1 {
			return errors.Errorf("expected at most 1 row to be updated for media %s but %d rows updated", req.MediaID, rowsUpdated)
		}

		return nil
	}); err != nil {
		return errors.Wrapf(err, "unable to process transcription for %s", req.MediaID)
	}

	// nothing to do if the job has already been completed
	if job.Completed {
		return nil
	}

	if skip {
		return awsutil.ErrMsgNotProcessedYet
	}

	// if we have waited too long for the transcription to be processed, skip the
	// transcription all together and publish the voicemail without the transcription.
	if time.Since(job.Created) > 15*time.Minute {
		if err := w.publishExternalMessage(&req, ""); err != nil {
			return errors.Errorf("unable to publish voicemail for media %s: %s", req.MediaID, err)
		}

		return w.jobCompleted(ctx, &req, &dal.TranscriptionJobUpdate{
			Completed:          ptr.Bool(true),
			CompletedTimestamp: ptr.Time(time.Now()),
			TimedOut:           ptr.Bool(true),
		})
	}

	// check if the transcription has completed
	// TODO check to see alternative end states for the transcription (like errored job status)
	// so that we are not waiting until the timeout to figure out whether or not the job reached
	// a completed state.
	jobStatus, err := w.transcriptionProvider.LookupTranscriptionJob(req.JobID)
	if err != nil {
		return errors.Errorf("unable to lookup status of transcription job for media %s : %s", req.MediaID, err)
	} else if jobStatus.Status != transcription.JobStatusCompleted {
		return awsutil.ErrMsgNotProcessedYet
	}

	if err := w.publishExternalMessage(&req, jobStatus.TranscriptionText); err != nil {
		return errors.Trace(err)
	}

	return w.jobCompleted(ctx, &req, &dal.TranscriptionJobUpdate{
		Completed:          ptr.Bool(true),
		CompletedTimestamp: ptr.Time(time.Now()),
	})
}

func (w *transcriptionTracker) publishExternalMessage(req *trackTranscriptionRequest, transcriptionText string) error {
	rm, err := w.dal.IncomingRawMessage(req.RawMessageID)
	if err != nil {
		return errors.Trace(err)
	}
	params := rm.GetTwilio()

	incomingType := excomms.IncomingCallEventItem_LEFT_VOICEMAIL
	if req.UrgentVoicemail {
		incomingType = excomms.IncomingCallEventItem_LEFT_URGENT_VOICEMAIL
	}

	return sns.Publish(w.snsAPI, w.externalMessageTopic, &excomms.PublishedExternalMessage{
		FromChannelID: params.From,
		ToChannelID:   params.To,
		Timestamp:     rm.Timestamp,
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:                incomingType,
				DurationInSeconds:   params.RecordingDuration,
				VoicemailMediaID:    req.MediaID,
				VoicemailDurationNS: req.MediaDurationNS,
				TranscriptionText:   transcriptionText,
			},
		},
	})
}

func (w *transcriptionTracker) jobCompleted(ctx context.Context, req *trackTranscriptionRequest, update *dal.TranscriptionJobUpdate) error {
	if err := w.dal.Transact(func(dl dal.DAL) error {
		if rowsUpdated, err := dl.UpdateTranscriptionJob(ctx, req.MediaID, update); err != nil {
			return errors.Errorf("unable to consider the job completed for media %s: %s", req.MediaID, err)
		} else if rowsUpdated > 1 {
			return errors.Errorf("expected at most 1 row to be updated for media %s but %d jobs updated", req.MediaID, rowsUpdated)
		}
		return nil
	}); err != nil {
		return errors.Trace(err)
	}

	cleaner.Publish(w.snsAPI, w.resourceCleanerTopic, &models.DeleteResourceRequest{
		Type:       models.DeleteResourceRequest_VOICEBASE_TRANSCRIPTION,
		ResourceID: req.MediaID,
	})

	return nil
}
