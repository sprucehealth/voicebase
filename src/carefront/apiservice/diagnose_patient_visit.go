package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"fmt"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
)

const (
	question_acne_diagnosis = "q_acne_diagnosis"
	question_acne_severity  = "q_acne_severity"
	question_acne_type      = "q_acne_type"
	question_rosacea_type   = "q_acne_rosacea_type"

	diagnoseSummaryTemplate = `Hi %s,

Based on the photographs you have provided, it looks like you have %s.

Acne is completely treatable but it will take some work and time to see results. I've put together the best treatment plan for your skin and with regular application you should begin to see results in 1-3 months.

Dr. %s`
)

type DiagnosePatientHandler struct {
	DataApi              api.DataAPI
	AuthApi              thriftapi.Auth
	LayoutStorageService api.CloudStorageAPI
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi thriftapi.Auth, cloudStorageApi api.CloudStorageAPI) *DiagnosePatientHandler {
	return &DiagnosePatientHandler{dataApi, authApi, cloudStorageApi}
}

func (d *DiagnosePatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDiagnosis(w, r)
	case "POST":
		d.diagnosePatient(w, r)
	}

}

func (d *DiagnosePatientHandler) getDiagnosis(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(DiagnosePatientRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	diagnosisLayout, err := d.getCurrentActiveDiagnoseLayoutForHealthCondition(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis layout for doctor to diagnose patient visit "+err.Error())
		return
	}
	diagnosisLayout.PatientVisitId = requestData.PatientVisitId

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIds := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := d.DataApi.GetAnswersForQuestionsInPatientVisit(api.DOCTOR_ROLE, questionIds, doctorId, requestData.PatientVisitId)

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *DiagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	answerIntakeRequestBody := &AnswerIntakeRequestBody{}

	err := jsonDecoder.Decode(answerIntakeRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get answer intake parameters from request body "+err.Error())
		return
	}

	err = validateRequestBody(answerIntakeRequestBody, w)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadGateway, "Bad parameters for question intake to diagnose patient: "+err.Error())
		return
	}

	doctorId, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(answerIntakeRequestBody.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(d.DataApi, answerIntakeRequestBody.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
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
		answersToStorePerQuestion[questionItem.QuestionId] = populateAnswersToStoreForQuestion(api.DOCTOR_ROLE, questionItem, answerIntakeRequestBody.PatientVisitId, doctorId, layoutVersionId)
	}

	err = d.DataApi.DeactivatePreviousDiagnosisForPatientVisit(answerIntakeRequestBody.PatientVisitId, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deactivate responses from previous diagnosis of this patient visit: "+err.Error())
		return
	}

	err = d.DataApi.StoreAnswersForQuestion(api.DOCTOR_ROLE, doctorId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStorePerQuestion)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
		return
	}
	err = d.addDiagnosisSummaryForPatientVisit(doctorId, answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong when trying to add and store the summary to the diagnosis of the patient visit: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, AnswerIntakeResponse{Result: "success"})

}

func (d *DiagnosePatientHandler) addDiagnosisSummaryForPatientVisit(doctorId int64, patientVisitId int64) error {
	// lookup answers for the following questions
	acneDiagnosisAnswer, err := d.DataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_diagnosis, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	acneSeverityAnswer, err := d.DataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_severity, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	acneTypeAnswer, err := d.DataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_type, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	rosaceaTypeAnswer, err := d.DataApi.GetDiagnosisResponseToQuestionWithTag(question_rosacea_type, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	diagnosisMessage := acneDiagnosisAnswer.AnswerSummary

	// for acne vulgaris, we only want the diagnosis to indicate acne
	if acneDiagnosisAnswer != nil && acneSeverityAnswer != nil {
		if acneTypeAnswer != nil {
			diagnosisMessage = fmt.Sprintf("%s %s %s", acneSeverityAnswer.AnswerSummary, acneTypeAnswer.AnswerSummary, acneDiagnosisAnswer.AnswerSummary)
		} else if rosaceaTypeAnswer != nil {
			diagnosisMessage = fmt.Sprintf("%s %s %s", acneSeverityAnswer.AnswerSummary, rosaceaTypeAnswer.AnswerSummary, acneDiagnosisAnswer.AnswerSummary)
		} else {
			diagnosisMessage = fmt.Sprintf("%s %s", acneSeverityAnswer.AnswerSummary, acneDiagnosisAnswer.PotentialAnswer)
		}
	}

	// nothing to do if the patient was not properly diagnosed by doctor so as to create a message
	if diagnosisMessage == "" {
		return nil
	}

	doctor, err := d.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		return err
	}

	patient, err := d.DataApi.GetPatientFromPatientVisitId(patientVisitId)
	if err != nil {
		return err
	}

	doctorFullName := fmt.Sprintf("%s %s", doctor.FirstName, doctor.LastName)
	diagnosisSummary := fmt.Sprintf(diagnoseSummaryTemplate, strings.Title(patient.FirstName), strings.ToLower(diagnosisMessage), strings.Title(doctorFullName))
	err = d.DataApi.AddDiagnosisSummaryForPatientVisit(diagnosisSummary, patientVisitId, doctorId)
	return err
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
				question.DoctorAnswers = doctorAnswers[question.QuestionId]
			}
		}
	}

	return questionIds
}

func (d *DiagnosePatientHandler) getCurrentActiveDiagnoseLayoutForHealthCondition(healthConditionId int64) (diagnosisLayout *info_intake.DiagnosisIntake, err error) {
	bucket, key, region, _, err := d.DataApi.GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId)
	if err != nil {
		return
	}

	data, _, err := d.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return
	}

	diagnosisLayout = &info_intake.DiagnosisIntake{}
	err = json.Unmarshal(data, diagnosisLayout)
	if err != nil {
		return
	}

	return
}

func (d *DiagnosePatientHandler) getLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (layoutVersionId int64, err error) {
	_, _, _, layoutVersionId, err = d.DataApi.GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId)
	return
}
