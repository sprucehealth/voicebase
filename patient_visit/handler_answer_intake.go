package patient_visit

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type answerIntakeHandler struct {
	dataAPI api.DataAPI
}

func NewAnswerIntakeHandler(dataAPI api.DataAPI) http.Handler {
	return &answerIntakeHandler{
		dataAPI: dataAPI,
	}
}

const (
	// Error we get from mysql is: "Error 1213: Deadlock found when trying to get lock; try restarting transaction"
	mysqlDeadlockError    = "Error 1213"
	waitTimeBeforeTxRetry = 100
)

func (a *answerIntakeHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (a *answerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd apiservice.AnswerIntakeRequestBody
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if err := apiservice.ValidateRequestBody(&rd, w); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	patientID, err := a.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientVisit, err := a.dataAPI.GetPatientVisitFromId(rd.PatientVisitId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patientVisit.PatientId.Int64() != patientID {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	answers := make(map[int64][]*common.AnswerIntake)
	for _, qItem := range rd.Questions {
		// enumerate the answers to store from the top level questions as well as the sub questions
		answers[qItem.QuestionId] = apiservice.PopulateAnswersToStoreForQuestion(
			api.PATIENT_ROLE,
			qItem,
			rd.PatientVisitId,
			patientID,
			patientVisit.LayoutVersionId.Int64())
	}

	patientIntake := &api.PatientIntake{
		PatientID:      patientID,
		PatientVisitID: rd.PatientVisitId,
		LVersionID:     patientVisit.LayoutVersionId.Int64(),
		SID:            rd.SessionID,
		SCounter:       rd.SessionCounter,
		Intake:         answers,
	}

	if err := a.dataAPI.StoreAnswersForQuestion(patientIntake); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
