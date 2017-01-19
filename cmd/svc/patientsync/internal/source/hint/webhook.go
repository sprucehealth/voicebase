package hint

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	patientsyncsettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/go-hint"
)

type webhookHandler struct {
	dl                 dal.DAL
	settingsCLI        settings.SettingsClient
	syncEventsQueueURL string
	sqsAPI             sqsiface.SQSAPI
}

func NewWebhookHandler(dl dal.DAL, settingsCLI settings.SettingsClient, syncEventsQueueURL string, sqsAPI sqsiface.SQSAPI) http.Handler {
	return &webhookHandler{
		dl:                 dl,
		settingsCLI:        settingsCLI,
		syncEventsQueueURL: syncEventsQueueURL,
		sqsAPI:             sqsAPI,
	}
}

type event struct {
	ID         string          `json:"id"`
	CreatedAt  time.Time       `json:"created_at"`
	Type       string          `json:"type"`
	PracticeID string          `json:"practice_id"`
	Object     json.RawMessage `json:"object"`
}

func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only deal with posts for now
	if r.Method != "POST" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var ev event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		httpError(w, "expected object of type event but got none", http.StatusBadRequest)
		return
	}

	var createPatient bool
	switch ev.Type {
	case "patient.created":
		createPatient = true
	case "patient.updated":
	default:
		w.WriteHeader(http.StatusOK)
		return
	}

	// lookup sync config based on practiceID
	syncConfig, err := h.dl.SyncConfigForExternalID(ev.PracticeID)
	if err != nil {
		httpError(w, fmt.Sprintf("Unable to get sync config for %s : %s", ev.PracticeID, err.Error()), http.StatusInternalServerError)
		return
	} else if syncConfig.Source != sync.SOURCE_HINT {
		httpError(w, fmt.Sprintf("Unexpected source %s", syncConfig.Source), http.StatusInternalServerError)
		return
	}

	// ensure that sync has been initiated and is in connected state
	syncBookmark, err := h.dl.SyncBookmarkForOrg(syncConfig.OrganizationEntityID)
	if err != nil && errors.Cause(err) != dal.NotFound {
		httpError(w, fmt.Sprintf("Unable to get sync bookmark for org %s : %s", syncConfig.OrganizationEntityID, err.Error()), http.StatusInternalServerError)
		return
	} else if syncBookmark == nil || syncBookmark.Status != dal.SyncStatusConnected {
		// nothing to do since this patient creation will be taken into account in the initial sync
		w.WriteHeader(http.StatusOK)
		return
	}

	var autoInvitePatients bool
	val, err := settings.GetBooleanValue(r.Context(), h.settingsCLI, &settings.GetValuesRequest{
		NodeID: syncConfig.OrganizationEntityID,
		Keys: []*settings.ConfigKey{
			{
				Key: patientsyncsettings.ConfigKeyAutoInvitePatients,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to get setting for orgID %s : %s", syncConfig.OrganizationEntityID, err)
	} else {
		autoInvitePatients = val.Value
	}

	var patient hint.Patient
	if err := json.Unmarshal(ev.Object, &patient); err != nil {
		httpError(w, fmt.Sprintf("Unable to unmarshal json for event object: %s", err), http.StatusInternalServerError)
		return
	}

	syncPatient := transformPatient(&patient, syncConfig)

	if err := func() error {

		var syncEvent *sync.Event
		if createPatient {
			syncEvent = &sync.Event{
				Source:               sync.SOURCE_HINT,
				OrganizationEntityID: syncConfig.OrganizationEntityID,
				Event: &sync.Event_PatientAddEvent{
					PatientAddEvent: &sync.PatientAddEvent{
						Patients:           []*sync.Patient{syncPatient},
						AutoInvitePatients: autoInvitePatients,
					},
				},
			}
		} else {
			syncEvent = &sync.Event{
				Source:               sync.SOURCE_HINT,
				OrganizationEntityID: syncConfig.OrganizationEntityID,
				Event: &sync.Event_PatientUpdateEvent{
					PatientUpdateEvent: &sync.PatientUpdatedEvent{
						Patients:           []*sync.Patient{syncPatient},
						AutoInvitePatients: autoInvitePatients,
					},
				},
			}
		}

		data, err := syncEvent.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		msg := base64.StdEncoding.EncodeToString(data)
		if _, err := h.sqsAPI.SendMessage(&sqs.SendMessageInput{
			MessageBody: &msg,
			QueueUrl:    &h.syncEventsQueueURL,
		}); err != nil {
			return errors.Trace(err)
		}
		return nil
	}(); err != nil {
		httpError(w, fmt.Sprintf("Unable to create sync event for adding patients for %s : %s", syncConfig.OrganizationEntityID, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func httpError(w http.ResponseWriter, errMsg string, statusCode int) {
	golog.Errorf(errMsg)
	http.Error(w, errMsg, http.StatusInternalServerError)
}
