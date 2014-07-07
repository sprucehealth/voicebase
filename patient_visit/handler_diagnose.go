package patient_visit

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type diagnosePatientHandler struct {
	dataApi     api.DataAPI
	authApi     api.AuthAPI
	environment string
}

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi api.AuthAPI, environment string) *diagnosePatientHandler {
	cacheInfoForUnsuitableVisit(dataApi)
	return &diagnosePatientHandler{
		dataApi:     dataApi,
		authApi:     authApi,
		environment: environment,
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

func (d *diagnosePatientHandler) getDiagnosis(w http.ResponseWriter, r *http.Request) {

	requestData := new(DiagnosePatientRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	} else if requestData.PatientVisitId == 0 {
		apiservice.WriteValidationError("patient_visit_id must be specified", w, r)
		return
	}

	patientVisit, err := d.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := apiservice.ValidateReadAccessToPatientCase(doctorId, patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), d.dataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	diagnosisLayout, err := GetDiagnosisLayout(d.dataApi, requestData.PatientVisitId, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *diagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	var answerIntakeRequestBody apiservice.AnswerIntakeRequestBody
	if err := apiservice.DecodeRequestData(&answerIntakeRequestBody, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if answerIntakeRequestBody.PatientVisitId == 0 {
		apiservice.WriteValidationError("patient_visit_id must be specified", w, r)
		return
	} else if err := apiservice.ValidateRequestBody(&answerIntakeRequestBody, w); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Bad parameters for question intake to diagnose patient: "+err.Error())
		return
	}

	patientVisit, err := d.dataApi.GetPatientVisitFromId(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := apiservice.ValidateWriteAccessToPatientCase(doctorId, patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), d.dataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

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
	if wasVisitMarkedUnsuitableForSpruce(&answerIntakeRequestBody) {
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
		dispatch.Default.Publish(&DiagnosisModifiedEvent{
			DoctorId:       doctorId,
			PatientVisitId: answerIntakeRequestBody.PatientVisitId,
			PatientCaseId:  patientVisit.PatientCaseId.Int64(),
			Diagnosis:      determineDiagnosisFromAnswers(&answerIntakeRequestBody),
		})
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
