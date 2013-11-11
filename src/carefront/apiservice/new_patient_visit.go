package apiservice

import (
	"carefront/api"
	"net/http"
)

type NewPatientVisitHandler struct {
	DataApi api.DataAPI
	AuthApi api.Auth
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

	patientVisitId, err := s.DataApi.CreateNewPatientVisit(patientId, 1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSONToHTTPResponseWriter(w, NewPatientVisitResponse{patientVisitId, ""})
}
