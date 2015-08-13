package patient_visit

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type answerIntakeHandler struct {
	dataAPI api.DataAPI
}

func NewAnswerIntakeHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&answerIntakeHandler{
				dataAPI: dataAPI,
			}), httputil.Post)
}

func (a *answerIntakeHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	if account.Role != api.RolePatient {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (a *answerIntakeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var rd apiservice.IntakeData
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if err := rd.Validate(w); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	account := apiservice.MustCtxAccount(ctx)
	patientID, err := a.dataAPI.GetPatientIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	patientVisit, err := a.dataAPI.GetPatientVisitFromID(rd.PatientVisitID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if patientVisit.PatientID.Int64() != patientID {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	answers := make(map[int64][]*common.AnswerIntake)
	for _, qItem := range rd.Questions {
		// enumerate the answers to store from the top level questions as well as the sub questions
		answers[qItem.QuestionID] = apiservice.PopulateAnswersToStoreForQuestion(
			api.RolePatient,
			qItem,
			rd.PatientVisitID,
			patientID,
			patientVisit.LayoutVersionID.Int64())
	}

	patientIntake := &api.PatientIntake{
		PatientID:      patientID,
		PatientVisitID: rd.PatientVisitID,
		LVersionID:     patientVisit.LayoutVersionID.Int64(),
		SID:            rd.SessionID,
		SCounter:       rd.SessionCounter,
		Intake:         answers,
	}

	if err := a.dataAPI.StoreAnswersForIntakes([]api.IntakeInfo{patientIntake}); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
