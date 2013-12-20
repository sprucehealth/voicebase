package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"errors"
	"github.com/gorilla/schema"
	"net/http"
)

type DiagnosePatientHandler struct {
	DataApi              api.DataAPI
	AuthApi              thriftapi.Auth
	LayoutStorageService api.CloudStorageAPI
	accountId            int64
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

func (d *DiagnosePatientHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func NewDiagnosePatientHandler(dataApi api.DataAPI, authApi thriftapi.Auth, cloudStorageApi api.CloudStorageAPI) *DiagnosePatientHandler {
	return &DiagnosePatientHandler{dataApi, authApi, cloudStorageApi, 0}
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

	doctorId, _, _, statusCode, err := d.validateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId)
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

	doctorId, _, _, httpStatusCode, err := d.validateDoctorAccessToPatientVisitAndGetRelevantData(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, err.Error())
		return
	}

	layoutVersionId, err := d.getLayoutVersionIdOfActiveDiagnosisLayout(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the layout version id of the diagnosis layout "+err.Error())
		return
	}

	// enumerate the answers to store from the top level questions as well as the sub questions
	answersToStore := populateAnswersToStore(api.DOCTOR_ROLE, answerIntakeRequestBody, doctorId, layoutVersionId)
	err = d.DataApi.StoreAnswersForQuestion(api.DOCTOR_ROLE, answerIntakeRequestBody.QuestionId, doctorId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStore)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, AnswerIntakeResponse{})

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

	data, err := d.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
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

func (d *DiagnosePatientHandler) validateDoctorAccessToPatientVisitAndGetRelevantData(PatientVisitId int64) (doctorId int64, patientVisit *common.PatientVisit, careTeam *common.PatientCareProviderGroup, httpStatusCode int, err error) {
	httpStatusCode = http.StatusOK
	doctorId, err = d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get doctor id from account id " + err.Error())
		return
	}

	patientVisit, err = d.DataApi.GetPatientVisitFromId(PatientVisitId)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get patient visit from id : " + err.Error())
		return
	}

	careTeam, err = d.DataApi.GetCareTeamForPatient(patientVisit.PatientId)
	if err != nil {
		httpStatusCode = http.StatusInternalServerError
		err = errors.New("Unable to get care team for patient visit id " + err.Error())
		return
	}

	if careTeam == nil {
		httpStatusCode = http.StatusForbidden
		err = errors.New("No care team assigned to patient visit so cannot diagnose patient visit")
		return
	}

	// ensure that the doctor is the current primary doctor for this patient
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.ProviderId != doctorId {
			httpStatusCode = http.StatusForbidden
			err = errors.New("Doctor is unable to diagnose patient because he/she is not the primary doctor")
			return
		}
	}

	return
}
