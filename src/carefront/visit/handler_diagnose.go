package visit

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/dispatch"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

type diagnosePatientHandler struct {
	dataApi     api.DataAPI
	authApi     thriftapi.Auth
	environment string
}

const (
	acneDiagnosisQuestionTag      = "q_acne_diagnosis"
	notSuitableForSpruceAnswerTag = "a_doctor_acne_not_suitable_spruce"
)

var notSuitableForSpruceAnswerId int64
var acneDiagnosisQuestionId int64

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi thriftapi.Auth, environment string) *diagnosePatientHandler {
	cacheInfoForUnsuitableVisit(dataApi)
	return &diagnosePatientHandler{
		dataApi:     dataApi,
		authApi:     authApi,
		environment: environment,
	}
}

func cacheInfoForUnsuitableVisit(dataApi api.DataAPI) {
	// cache the answer id of the not suitable for spruce answer
	answerInfoList, err := dataApi.GetAnswerInfoForTags([]string{notSuitableForSpruceAnswerTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err.Error())
	} else if len(answerInfoList) != 1 {
		panic("Expected 1 answer for not suitable for spruce tag")
	} else {
		notSuitableForSpruceAnswerId = answerInfoList[0].PotentialAnswerId
	}

	// cache the question id of the question for which we expect answer option of not suitable for spruce
	question, err := dataApi.GetQuestionInfo(acneDiagnosisQuestionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err.Error())
	}
	acneDiagnosisQuestionId = question.Id
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId  int64 `schema:"patient_visit_id,required"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
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
	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(DiagnosePatientRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := apiservice.EnsureTreatmentPlanOrPatientVisitIdPresent(d.dataApi, treatmentPlanId, &patientVisitId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, apiservice.GetContext(r).AccountId, d.dataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	diagnosisLayout, err := d.getCurrentActiveDiagnoseLayoutForHealthCondition(apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis layout for doctor to diagnose patient visit "+err.Error())
		return
	}
	diagnosisLayout.PatientVisitId = patientVisitId

	if treatmentPlanId == 0 {
		treatmentPlanId, err = d.dataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get active treatment plan for patient visit: "+err.Error())
			return
		}
	}
	diagnosisLayout.TreatmentPlanId = treatmentPlanId

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIds := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := d.dataApi.GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds, patientVisitReviewData.DoctorId, patientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get answers for question: "+err.Error())
		return
	}

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *diagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	var answerIntakeRequestBody apiservice.AnswerIntakeRequestBody
	if err := json.NewDecoder(r.Body).Decode(&answerIntakeRequestBody); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get answer intake parameters from request body "+err.Error())
		return
	}

	if err := apiservice.ValidateRequestBody(&answerIntakeRequestBody, w); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Bad parameters for question intake to diagnose patient: "+err.Error())
		return
	}

	patientVisitReviewData, httpStatusCode, err := apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(answerIntakeRequestBody.PatientVisitId, apiservice.GetContext(r).AccountId, d.dataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, httpStatusCode, err.Error())
		return
	}

	if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataApi, answerIntakeRequestBody.PatientVisitId, api.CASE_STATUS_REVIEWING); err != nil {
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
		answersToStorePerQuestion[questionItem.QuestionId] = apiservice.PopulateAnswersToStoreForQuestion(api.DOCTOR_ROLE, questionItem, patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId, layoutVersionId)
	}

	if err := d.dataApi.DeactivatePreviousDiagnosisForPatientVisit(patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deactivate responses from previous diagnosis of this patient visit: "+err.Error())
		return
	}

	if err := d.dataApi.StoreAnswersForQuestion(api.DOCTOR_ROLE, patientVisitReviewData.DoctorId, patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), layoutVersionId, answersToStorePerQuestion); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
		return
	}

	// check if the doctor diagnosed the patient's visit as being unsuitable for spruce
	unsuitableForSpruceVisit := false
	for _, questionItem := range answerIntakeRequestBody.Questions {
		if questionItem.QuestionId == acneDiagnosisQuestionId {
			for _, answerItem := range questionItem.AnswerIntakes {
				if answerItem.PotentialAnswerId == notSuitableForSpruceAnswerId {
					unsuitableForSpruceVisit = true
					break
				}
			}
		}
	}

	if unsuitableForSpruceVisit {
		err = d.dataApi.ClosePatientVisit(patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), api.CASE_STATUS_TRIAGED)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to closed: "+err.Error())
			return
		}

		dispatch.Default.Publish(&PatientVisitMarkedUnsuitableEvent{
			DoctorId:       patientVisitReviewData.DoctorId,
			PatientVisitId: patientVisitReviewData.PatientVisit.PatientVisitId.Int64(),
		})

	} else {
		treatmentPlanId, err := d.dataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, answerIntakeRequestBody.PatientVisitId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan for patient visit: "+err.Error())
			return
		}

		dispatch.Default.Publish(&DiagnosisModifiedEvent{
			DoctorId:        patientVisitReviewData.DoctorId,
			PatientVisitId:  patientVisitReviewData.PatientVisit.PatientVisitId.Int64(),
			TreatmentPlanId: treatmentPlanId,
		})
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.AnswerIntakeResponse{Result: "success"})
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

func (d *diagnosePatientHandler) getCurrentActiveDiagnoseLayoutForHealthCondition(healthConditionId int64) (*info_intake.DiagnosisIntake, error) {
	data, _, err := d.dataApi.GetActiveDoctorDiagnosisLayout(healthConditionId)
	if err != nil {
		return nil, err
	}

	var diagnosisLayout info_intake.DiagnosisIntake
	if err = json.Unmarshal(data, &diagnosisLayout); err != nil {
		return nil, err
	}

	return &diagnosisLayout, nil
}
