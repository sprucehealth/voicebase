package apiservice

import (
	"carefront/api"
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
	PatientVisitId int64  `json:"patient_visit_id,string"`
	ClientLayout   string `json:"client_layout,omitempty"`
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
		data, err := s.getCurrentActiveClientLayoutForHealthCondition(1, 1)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		WriteJSONToHTTPResponseWriter(w, NewPatientVisitResponse{patientVisitId, string(data)})
		return
	}

	patientVisitId, err = s.DataApi.CreateNewPatientVisit(patientId, 1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := s.getCurrentActiveClientLayoutForHealthCondition(1, 1)
	WriteJSONToHTTPResponseWriter(w, NewPatientVisitResponse{patientVisitId, string(data)})
}

func (s *NewPatientVisitHandler) getCurrentActiveClientLayoutForHealthCondition(healthConditionId, languageId int64) (data []byte, err error) {
	bucket, key, region, err := s.DataApi.GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId)
	if err != nil {
		return nil, err
	}

	data, err = s.CloudStorageApi.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}

	return data, err
}
