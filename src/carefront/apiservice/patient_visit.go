package apiservice

import (
	"carefront/api"
	"carefront/info_intake"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	HEALTH_CONDITION_ACNE_ID = 1
)

type PatientVisitHandler struct {
	DataApi         api.DataAPI
	AuthApi         api.Auth
	CloudStorageApi api.CloudStorageAPI
	accountId       int64
}

type PatientVisitErrorResponse struct {
	ErrorString string `json:"error"`
}

type PatientVisitResponse struct {
	PatientVisitId int64                        `json:"patient_visit_id,string"`
	ClientLayout   *info_intake.HealthCondition `json:"health_condition,omitempty"`
}

func NewPatientVisitHandler(dataApi api.DataAPI, authApi api.Auth, cloudStorageApi api.CloudStorageAPI) *PatientVisitHandler {
	return &PatientVisitHandler{dataApi, authApi, cloudStorageApi, 0}
}

func (s *PatientVisitHandler) AccountIdFromAuthToken(accountId int64) {
	s.accountId = accountId
}

func (s *PatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.returnNewOrOpenPatientVisit(w, r)
	}
}

func (s *PatientVisitHandler) returnNewOrOpenPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientId, err := s.DataApi.GetPatientIdFromAccountId(s.accountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	healthCondition, layoutVersionId, err := s.getCurrentActiveClientLayoutForHealthCondition(HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	rawBytes, _ := json.MarshalIndent(healthCondition, "", " ")
	fmt.Println(string(rawBytes))

	// check if there is an open patient visit for the given health condition and return
	// that to the patient
	patientVisitId, err := s.DataApi.GetActivePatientVisitForHealthCondition(patientId, HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if patientVisitId == -1 {
		patientVisitId, err = s.DataApi.CreateNewPatientVisit(patientId, HEALTH_CONDITION_ACNE_ID, layoutVersionId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	globalSectionPatientAnswers, err := s.DataApi.GetPatientAnswersFromGlobalSections(patientId)
	fmt.Println(globalSectionPatientAnswers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	PopulateHealthConditionWithPatientAnswers(healthCondition, globalSectionPatientAnswers)

	visitPatientAnswers, err := s.DataApi.GetPatientAnswersForVisit(patientId, patientVisitId)
	fmt.Println(visitPatientAnswers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	PopulateHealthConditionWithPatientAnswers(healthCondition, visitPatientAnswers)
	WriteJSONToHTTPResponseWriter(w, PatientVisitResponse{patientVisitId, healthCondition})
}

func PopulateHealthConditionWithPatientAnswers(healthCondition *info_intake.HealthCondition, patientAnswers map[int64][]api.PatientAnswerToQuestion) {
	for _, section := range healthCondition.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				// go through each question to see if there exists a patient answer for it
				if patientAnswers[question.QuestionId] != nil {
					question.PatientAnswers = make([]*info_intake.PatientAnswer, 0, len(patientAnswers[question.QuestionId]))
					for _, patientAnswerToQuestion := range patientAnswers[question.QuestionId] {
						question.PatientAnswers = append(question.PatientAnswers, &info_intake.PatientAnswer{
							PatientAnswerId:   patientAnswerToQuestion.PatientInfoIntakeId,
							PotentialAnswerId: patientAnswerToQuestion.PotentialAnswerId,
							AnswerText:        patientAnswerToQuestion.AnswerText})
					}
					fmt.Println(question.PatientAnswers)
				}
			}
		}
	}
}

func (s *PatientVisitHandler) getCurrentActiveClientLayoutForHealthCondition(healthConditionId, languageId int64) (healthCondition *info_intake.HealthCondition, layoutVersionId int64, err error) {
	bucket, key, region, layoutVersionId, err := s.DataApi.GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId)
	if err != nil {
		return nil, 0, err
	}

	data, err := s.CloudStorageApi.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, 0, err
	}
	healthCondition = &info_intake.HealthCondition{}
	err = json.Unmarshal(data, healthCondition)
	if err != nil {
		return nil, 0, err
	}

	return healthCondition, layoutVersionId, err
}
