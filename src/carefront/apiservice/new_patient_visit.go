package apiservice

import (
	"carefront/api"
	"carefront/info_intake"
	"encoding/json"
	"net/http"
)

type NewPatientVisitHandler struct {
	DataApi         api.DataAPI
	AuthApi         api.Auth
	CloudStorageApi api.CloudStorageAPI
}

type NewPatientVisitErrorResponse struct {
	ErrorString string `json:"error"`
}

type NewPatientVisitResponse struct {
	PatientVisitId int64                        `json:"patient_visit_id,string"`
	ClientLayout   *info_intake.HealthCondition `json:"health_condition,omitempty"`
}

func (s *NewPatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, err := GetAuthTokenFromHeader(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, accountId, err := s.AuthApi.ValidateToken(token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patientId, err := s.DataApi.GetPatientIdFromAccountId(accountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// check if there is an open patient visit for the given health condition and return
	// that to the patient
	patientVisitId, _ := s.DataApi.GetActivePatientVisitForHealthCondition(patientId, 1)
	if patientVisitId != -1 {
		healthCondition, err := s.getCurrentActiveClientLayoutForHealthCondition(1, 1)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		WriteJSONToHTTPResponseWriter(w, NewPatientVisitResponse{patientVisitId, healthCondition})
		return
	}

	patientVisitId, err = s.DataApi.CreateNewPatientVisit(patientId, 1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	healthCondition, err := s.getCurrentActiveClientLayoutForHealthCondition(1, 1)
	WriteJSONToHTTPResponseWriter(w, NewPatientVisitResponse{patientVisitId, healthCondition})
}

func (s *NewPatientVisitHandler) getCurrentActiveClientLayoutForHealthCondition(healthConditionId, languageId int64) (healthCondition *info_intake.HealthCondition, err error) {
	bucket, key, region, err := s.DataApi.GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId)
	if err != nil {
		return nil, err
	}

	data, err := s.CloudStorageApi.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}
	healthCondition = &info_intake.HealthCondition{}
	err = json.Unmarshal(data, healthCondition)
	if err != nil {
		return nil, err
	}

	return healthCondition, err
}
