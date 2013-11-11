package apiservice

import (
	"carefront/api"
	"carefront/info_intake"
	"encoding/json"
	"net/http"
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

	healthCondition, layoutVersionId, err := s.getCurrentActiveClientLayoutForHealthCondition(1, api.EN_LANGUAGE_ID)

	// check if there is an open patient visit for the given health condition and return
	// that to the patient
	patientVisitId, _ := s.DataApi.GetActivePatientVisitForHealthCondition(patientId, 1)
	if patientVisitId != -1 {
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		WriteJSONToHTTPResponseWriter(w, PatientVisitResponse{patientVisitId, healthCondition})
		return
	}

	patientVisitId, err = s.DataApi.CreateNewPatientVisit(patientId, 1, layoutVersionId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSONToHTTPResponseWriter(w, PatientVisitResponse{patientVisitId, healthCondition})
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
