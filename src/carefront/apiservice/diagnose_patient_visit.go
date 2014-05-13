package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/dispatch"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

type DiagnosePatientHandler struct {
	DataApi              api.DataAPI
	AuthApi              thriftapi.Auth
	LayoutStorageService api.CloudStorageAPI
	Environment          string
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId  int64 `schema:"patient_visit_id,required"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi thriftapi.Auth, cloudStorageApi api.CloudStorageAPI) *DiagnosePatientHandler {
	return &DiagnosePatientHandler{DataApi: dataApi, AuthApi: authApi, LayoutStorageService: cloudStorageApi}
}

func (d *DiagnosePatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		d.getDiagnosis(w, r)
	case HTTP_POST:
		d.diagnosePatient(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *DiagnosePatientHandler) getDiagnosis(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(DiagnosePatientRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := EnsureTreatmentPlanOrPatientVisitIdPresent(d.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	diagnosisLayout, err := d.getCurrentActiveDiagnoseLayoutForHealthCondition(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis layout for doctor to diagnose patient visit "+err.Error())
		return
	}
	diagnosisLayout.PatientVisitId = patientVisitId

	if treatmentPlanId == 0 {
		treatmentPlanId, err = d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to get active treatment plan for patient visit: "+err.Error())
			return
		}
	}
	diagnosisLayout.TreatmentPlanId = treatmentPlanId

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIds := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := d.DataApi.GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds, patientVisitReviewData.DoctorId, patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get answers for question: "+err.Error())
		return
	}

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *DiagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	var answerIntakeRequestBody AnswerIntakeRequestBody
	if err := json.NewDecoder(r.Body).Decode(&answerIntakeRequestBody); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get answer intake parameters from request body "+err.Error())
		return
	}

	if err := validateRequestBody(&answerIntakeRequestBody, w); err != nil {
		WriteDeveloperError(w, http.StatusBadGateway, "Bad parameters for question intake to diagnose patient: "+err.Error())
		return
	}

	patientVisitReviewData, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(answerIntakeRequestBody.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, err.Error())
		return
	}

	if err := EnsurePatientVisitInExpectedStatus(d.DataApi, answerIntakeRequestBody.PatientVisitId, api.CASE_STATUS_REVIEWING); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	layoutVersionId, err := d.getLayoutVersionIdOfActiveDiagnosisLayout(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the layout version id of the diagnosis layout "+err.Error())
		return
	}

	answersToStorePerQuestion := make(map[int64][]*common.AnswerIntake)
	for _, questionItem := range answerIntakeRequestBody.Questions {
		// enumerate the answers to store from the top level questions as well as the sub questions
		answersToStorePerQuestion[questionItem.QuestionId] = populateAnswersToStoreForQuestion(api.DOCTOR_ROLE, questionItem, patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId, layoutVersionId)
	}

	if err := d.DataApi.DeactivatePreviousDiagnosisForPatientVisit(patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deactivate responses from previous diagnosis of this patient visit: "+err.Error())
		return
	}

	if err := d.DataApi.StoreAnswersForQuestion(api.DOCTOR_ROLE, patientVisitReviewData.DoctorId, patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), layoutVersionId, answersToStorePerQuestion); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
		return
	}

	treatmentPlanId, err := d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan for patient visit: "+err.Error())
		return
	}

	dispatch.Default.Publish(&DiagnosisModifiedEvent{
		DoctorId:        patientVisitReviewData.DoctorId,
		PatientVisitId:  patientVisitReviewData.PatientVisit.PatientVisitId.Int64(),
		TreatmentPlanId: treatmentPlanId,
	})

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, AnswerIntakeResponse{Result: "success"})
}

func getQuestionIdsInDiagnosisLayout(diagnosisLayout *info_intake.DiagnosisIntake) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			questionIds = append(questionIds, question.QuestionId)
		}
	}

	return questionIds
}

func populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout *info_intake.DiagnosisIntake, doctorAnswers map[int64][]*common.AnswerIntake) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			// go through each question to see if there exists a patient answer for it
			if doctorAnswers[question.QuestionId] != nil {
				question.Answers = doctorAnswers[question.QuestionId]
			}
		}
	}

	return questionIds
}

func (d *DiagnosePatientHandler) getCurrentActiveDiagnoseLayoutForHealthCondition(healthConditionId int64) (*info_intake.DiagnosisIntake, error) {
	bucket, key, region, _, err := d.DataApi.GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId)
	if err != nil {
		return nil, err
	}

	data, _, err := d.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}

	var diagnosisLayout info_intake.DiagnosisIntake
	if err = json.Unmarshal(data, &diagnosisLayout); err != nil {
		return nil, err
	}

	return &diagnosisLayout, nil
}

func (d *DiagnosePatientHandler) getLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (layoutVersionId int64, err error) {
	_, _, _, layoutVersionId, err = d.DataApi.GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId)
	return
}
