package hint

import (
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/go-hint"
)

// DoInitialSync is responsible for paginating through all patients in the existing account for the practice
// and publishing sync events to create corresponding conversations in the spruce account.
func DoInitialSync(dl dal.DAL, orgID string, syncEventsQueueURL string, sqsAPI sqsiface.SQSAPI) error {
	golog.Debugf("Attempting initial sync for %s", orgID)
	// get sync config for orgID
	syncConfig, err := dl.SyncConfigForOrg(orgID, sync.SOURCE_HINT.String())
	if err != nil {
		return errors.Trace(err)
	} else if syncConfig.Source != sync.SOURCE_HINT {
		return errors.Errorf("Expected source for config to be %s but it was %s", sync.SOURCE_HINT, syncConfig.Source)
	} else if syncConfig.GetHint() == nil {
		return errors.Errorf("Hint token not available for org %s", orgID)
	}

	// check if we have a bookmark for where the sync last stopped
	sb, err := dl.SyncBookmarkForOrg(orgID)
	if errors.Cause(err) != dal.NotFound && err != nil {
		return errors.Trace(err)
	}

	practiceKey := syncConfig.GetHint().AccessToken

	var queryItems []*hint.QueryItem
	var syncBookmark time.Time
	if sb != nil {

		// nothing to do if the sync is already complete
		if sb.Status == dal.SyncStatusConnected {
			return nil
		}

		syncBookmark = sb.Bookmark

		queryItems = []*hint.QueryItem{
			{
				Field: "created_at",
				Operations: []*hint.Operation{
					{
						Operator: hint.OperatorGreaterThan,
						Operand:  sb.Bookmark.String(),
					},
				},
			},
		}
	}

	iter := hint.ListPatient(practiceKey, &hint.ListParams{
		Sort: &hint.Sort{
			By: "created_at",
		},
		Items: queryItems,
	})

	// initiate adding of 20 items at a time
	patients := make([]*sync.Patient, 0, 20)
	for iter.Next() {

		hintPatient := iter.Current().(*hint.Patient)
		syncPatient := transformPatient(hintPatient)

		patients = append(patients, syncPatient)
		syncBookmark = hintPatient.CreatedAt

		// create sync event and drain list
		if len(patients) == 20 {

			if err := createSyncEvent(orgID, syncEventsQueueURL, patients, sqsAPI); err != nil {
				return errors.Trace(err)
			}

			// update bookmark
			if err := dl.UpdateSyncBookmarkForOrg(orgID, syncBookmark, dal.SyncStatusInitiated); err != nil {
				return errors.Trace(err)
			}
			patients = patients[:0]
		}
	}

	if err := iter.Err(); err != nil {
		return errors.Trace(err)
	}

	if len(patients) > 0 {
		if err := createSyncEvent(orgID, syncEventsQueueURL, patients, sqsAPI); err != nil {
			return errors.Trace(err)
		}
		// update bookmark
		if err := dl.UpdateSyncBookmarkForOrg(orgID, syncBookmark, dal.SyncStatusConnected); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
