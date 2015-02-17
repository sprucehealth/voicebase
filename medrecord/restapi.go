package medrecord

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type apiHandler struct {
	dataAPI api.DataAPI
	queue   *common.SQSQueue
}

func NewRequestAPIHandler(dataAPI api.DataAPI, queue *common.SQSQueue) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&apiHandler{
			dataAPI: dataAPI,
			queue:   queue,
		}), []string{"POST"})
}

func (h *apiHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID, err := h.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	mrID, err := h.dataAPI.CreateMedicalRecord(patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	js, err := json.Marshal(&queueMessage{
		MedicalRecordID: mrID,
		PatientID:       patientID,
	})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := h.queue.QueueService.SendMessage(h.queue.QueueURL, 0, string(js)); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		MedicalRecordID int64 `json:"medical_record_id"`
	}{
		MedicalRecordID: mrID,
	})
}
