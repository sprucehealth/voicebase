package patient_visit

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

type diagnosePatientHandler struct {
	dataApi api.DataAPI
	authApi api.AuthAPI
}

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi api.AuthAPI) *diagnosePatientHandler {
	cacheInfoForUnsuitableVisit(dataApi)
	return &diagnosePatientHandler{
		dataApi: dataApi,
		authApi: authApi,
	}
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

func (d *diagnosePatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getDiagnosis(w, r)
	case apiservice.HTTP_POST:
		d.diagnosePatient(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *diagnosePatientHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	switch r.Method {
	case apiservice.HTTP_GET:
		requestData := new(DiagnosePatientRequestData)
		if err := apiservice.DecodeRequestData(requestData, r); err != nil {
			return false, apiservice.NewValidationError(err.Error(), r)
		} else if requestData.PatientVisitId == 0 {
			return false, apiservice.NewValidationError("patient_id must be specified", r)
		}
		ctxt.RequestCache[apiservice.RequestData] = requestData

		patientVisit, err := d.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}
	case apiservice.HTTP_POST:
		answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{}
		if err := apiservice.DecodeRequestData(answerIntakeRequestBody, r); err != nil {
			return false, apiservice.NewValidationError(err.Error(), r)
		} else if answerIntakeRequestBody.PatientVisitId == 0 {
			return false, apiservice.NewValidationError("patient_visit_id must be specified", r)
		}
		ctxt.RequestCache[apiservice.RequestData] = answerIntakeRequestBody

		patientVisit, err := d.dataApi.GetPatientVisitFromId(answerIntakeRequestBody.PatientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *diagnosePatientHandler) getDiagnosis(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*DiagnosePatientRequestData)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)

	diagnosisLayout, err := GetDiagnosisLayout(d.dataApi, requestData.PatientVisitId, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *diagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	answerIntakeRequestBody := ctxt.RequestCache[apiservice.RequestData].(*apiservice.AnswerIntakeRequestBody)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataApi, answerIntakeRequestBody.PatientVisitId, common.PVStatusReviewing); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	layoutVersionId, err := d.dataApi.GetLayoutVersionIdOfActiveDiagnosisLayout(apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the layout version id of the diagnosis layout "+err.Error())
		return
	}

	answersToStorePerQuestion := make(map[int64][]*common.AnswerIntake)
	for _, questionItem := range answerIntakeRequestBody.Questions {
		// enumerate the answers to store from the top level questions as well as the sub questions
		answersToStorePerQuestion[questionItem.QuestionId] = apiservice.PopulateAnswersToStoreForQuestion(api.DOCTOR_ROLE, questionItem, answerIntakeRequestBody.PatientVisitId, doctorId, layoutVersionId)
	}

	if err := d.dataApi.DeactivatePreviousDiagnosisForPatientVisit(answerIntakeRequestBody.PatientVisitId, doctorId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataApi.StoreAnswersForQuestion(api.DOCTOR_ROLE, doctorId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStorePerQuestion); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// check if the doctor diagnosed the patient's visit as being unsuitable for spruce
	if wasVisitMarkedUnsuitableForSpruce(answerIntakeRequestBody) {
		err = d.dataApi.ClosePatientVisit(answerIntakeRequestBody.PatientVisitId, common.PVStatusTriaged)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to closed: "+err.Error())
			return
		}

		dispatch.Default.Publish(&PatientVisitMarkedUnsuitableEvent{
			DoctorId:       doctorId,
			PatientVisitId: answerIntakeRequestBody.PatientVisitId,
		})

	} else {
		diagnosis := determineDiagnosisFromAnswers(answerIntakeRequestBody)

		if err := d.dataApi.UpdateDiagnosisForPatientVisit(patientVisit.PatientVisitId.Int64(), diagnosis); err != nil {
			golog.Errorf("Unable to update diagnosis for patient visit: %s", err)
		}

		dispatch.Default.Publish(&DiagnosisModifiedEvent{
			DoctorId:       doctorId,
			PatientVisitId: answerIntakeRequestBody.PatientVisitId,
			PatientCaseId:  patientVisit.PatientCaseId.Int64(),
			Diagnosis:      diagnosis,
		})
	}

	apiservice.WriteJSONSuccess(w)
}
