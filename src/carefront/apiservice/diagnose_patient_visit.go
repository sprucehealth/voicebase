package apiservice

import (
	"carefront/api"
	"carefront/info_intake"
	thriftapi "carefront/thrift/api"
	"encoding/json"
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

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id "+err.Error())
		return
	}

	careTeam, err := d.DataApi.GetCareTeamForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient visit id "+err.Error())
		return
	}

	if careTeam == nil {
		WriteDeveloperError(w, http.StatusForbidden, "No care team assigned to patient visit so cannot diagnose patient visit")
		return
	}
	// ensure that the doctor is the primary doctor on this case
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == "PRIMARY_DOCTOR" && assignment.ProviderId != doctorId {
			WriteDeveloperError(w, http.StatusForbidden, "Doctor is unable to diagnose patient because he/she is not the primary doctor")
			return
		}
	}

	diagnosisLayout, err := d.getCurrentActiveDiagnoseLayoutForHealthCondition(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis layout for doctor to diagnose patient visit "+err.Error())
		return
	}

	diagnosisLayout.PatientVisitId = requestData.PatientVisitId

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
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
