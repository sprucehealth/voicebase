package patient_visit

import (
	"encoding/json"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"net/http"
)

type diagnosePatientHandler struct {
	dataApi     api.DataAPI
	authApi     api.AuthAPI
	environment string
}

const (
	acneDiagnosisQuestionTag      = "q_acne_diagnosis"
	notSuitableForSpruceAnswerTag = "a_doctor_acne_not_suitable_spruce"
)

var notSuitableForSpruceAnswerId int64
var acneDiagnosisQuestionId int64

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi api.AuthAPI, environment string) *diagnosePatientHandler {
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
		notSuitableForSpruceAnswerId = answerInfoList[0].AnswerId
	}

	// cache the question id of the question for which we expect answer option of not suitable for spruce
	question, err := dataApi.GetQuestionInfo(acneDiagnosisQuestionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err.Error())
	}
	acneDiagnosisQuestionId = question.QuestionId
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

func GetDiagnosisLayout(dataApi api.DataAPI, patientVisitId, doctorId int64) (*info_intake.DiagnosisIntake, error) {

	diagnosisLayout, err := getCurrentActiveDiagnoseLayoutForHealthCondition(dataApi, apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		return nil, err
	}
	diagnosisLayout.PatientVisitId = patientVisitId

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIds := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := dataApi.GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds, doctorId, patientVisitId)
	if err != nil {
		return nil, err
	}

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)
	return diagnosisLayout, nil
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
		})
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
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

func populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout *info_intake.DiagnosisIntake, doctorAnswers map[int64][]common.Answer) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			// go through each question to see if there exists a patient answer for it
			question.Answers = doctorAnswers[question.QuestionId]
		}
	}

	return questionIds
}

func getCurrentActiveDiagnoseLayoutForHealthCondition(dataApi api.DataAPI, healthConditionId int64) (*info_intake.DiagnosisIntake, error) {
	data, _, err := dataApi.GetActiveDoctorDiagnosisLayout(healthConditionId)
	if err != nil {
		return nil, err
	}

	var diagnosisLayout info_intake.DiagnosisIntake
	if err = json.Unmarshal(data, &diagnosisLayout); err != nil {
		return nil, err
	}

	return &diagnosisLayout, nil
}
