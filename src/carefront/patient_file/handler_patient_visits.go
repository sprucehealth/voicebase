package patient_file

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type patientVisitsHandler struct {
	DataApi api.DataAPI
}

type request struct {
	PatientId string `schema:"patient_id,required"`
}

type response struct {
	PatientVisits []*common.PatientVisit `json:"patient_visits"`
}

func NewPatientVisitsHandler(dataApi api.DataAPI) *patientVisitsHandler {
	return &patientVisitsHandler{
		DataApi: dataApi,
	}
}

func (p *patientVisitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := request{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientId, err := strconv.ParseInt(requestData.PatientId, 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "PatientId not correctly specified as request parameter: "+err.Error())
		return
	}

	doctor, err := p.DataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromId(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from id: "+err.Error())
		return
	}

	if err := apiservice.VerifyDoctorPatientRelationship(p.DataApi, doctor, patient); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusForbidden, "Unable to verify the patient-doctor relationship: "+err.Error())
		return
	}

	patientVisits, err := p.DataApi.GetPatientVisitsForPatient(patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visits for patient: "+err.Error())
		return
	}

	responseData := response{
		PatientVisits: patientVisits,
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &responseData)
}
