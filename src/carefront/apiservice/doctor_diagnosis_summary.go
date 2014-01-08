package apiservice

import (
	"carefront/api"
	"carefront/common"
	"github.com/gorilla/schema"
	"net/http"
)

type DiagnosisSummaryHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type DiagnosisSummaryRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

func (d *DiagnosisSummaryHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DiagnosisSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDiagnosisSummaryForPatientVisit(w, r)
	case "POST":
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (d *DiagnosisSummaryHandler) getDiagnosisSummaryForPatientVisit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(DiagnosisSummaryRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	_, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	summary, err := d.DataApi.GetDiagnosisSummaryForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis summary for patient visit: "+err.Error())
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &common.DiagnosisSummary{Type: "text", Summary: summary})
}
