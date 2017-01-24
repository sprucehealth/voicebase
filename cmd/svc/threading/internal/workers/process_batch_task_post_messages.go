package workers

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/threading"
)

type processPostMessagesThreadClient interface {
	cloneAttachmentsThreadClient
	postMessagesThreadClient
}

func processBatchTaskPostMessages(ctx context.Context, task *models.BatchTask, dl dal.DAL, threadClient processPostMessagesThreadClient) (string, error) {
	if task.Type != models.BatchTaskTypePostMessages {
		return InternalErrorMessage, errors.Errorf("Cannot handle batch task type %s with processBatchTaskPostMessages", task.Type)
	}
	// deserialize our PostMessagesRequest
	req := &threading.PostMessagesRequest{}
	if err := req.Unmarshal(task.Data); err != nil {
		return InternalErrorMessage, errors.Wrap(err, "Error while unmarhsaling into *threading.PostMessageRequest for processBatchTaskPostMessages")
	}
	threadID, err := models.ParseThreadID(req.ThreadID)
	if err != nil {
		return InternalErrorMessage, errors.Wrapf(err, "Error parsing Thread ID %s", req.ThreadID)
	}
	threads, err := dl.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return InternalErrorMessage, errors.Wrapf(err, "Error while looking up thread %s for batch post message", threadID)
	}
	if len(threads) == 0 {
		return fmt.Sprintf("No such thread %s", threadID), errors.Errorf("Expected to find 1 thread for ID %v but got %d", threadID, len(threads))
	}
	if len(threads) != 1 {
		return InternalErrorMessage, errors.Errorf("Expected to find 1 thread for ID %v but got %d", threadID, len(threads))
	}
	thread := threads[0]
	if thread.Deleted {
		golog.Infof("Found PostMessages BatchTask for DELETED thread %s - IGNORING", threadID)
		return "", nil
	}
	// Clone our attachments
	parallel := conc.NewParallel()
	for _, msg := range req.Messages {
		// Capture msg
		cMsg := msg
		if len(msg.Attachments) != 0 {
			parallel.Go(func() error {
				// TODO: Build generic pattern for cloning context
				cctx := context.Background()
				resp, err := threadClient.CloneAttachments(cctx, &threading.CloneAttachmentsRequest{
					Attachments: cMsg.Attachments,
					OwnerType:   threading.CLONED_ATTACHMENT_OWNER_THREAD,
					OwnerID:     threadID.String(),
				})
				if err != nil {
					return errors.Wrap(err, "Error cloning attachments for batch post message")
				}
				cMsg.Attachments = resp.Attachments
				return nil
			})
		}
	}
	if err := parallel.Wait(); err != nil {
		return InternalErrorMessage + " " + thread.UserTitle, errors.Wrap(err, "Error while waiting cloning attachments in parallel")
	}
	// Send our prepared message
	if _, err := threadClient.PostMessages(ctx, req); err != nil {
		return InternalErrorMessage + " " + thread.UserTitle, errors.Wrap(err, "Error during PostMessages for batch PostMessages")
	}
	return "", nil
}
