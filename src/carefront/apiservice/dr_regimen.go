package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
)

type DoctorRegimenHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type GetDoctorRegimenRequestResponse struct {
	RegimenSteps     []*common.DoctorInstructionItem `json:"regimen_steps"`
	DrugInternalName string                          `json:"drug_internal_name"`
	PatientVisitId   int64                           `json:"patient_visit_id,string,omitempty`
}

func NewDoctorRegimenHandler(dataApi api.DataAPI) *DoctorRegimenHandler {
	return &DoctorRegimenHandler{DataApi: dataApi, accountId: 0}
}

func (d *DoctorRegimenHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorRegimenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getRegimenSteps(w, r)
	}
}

func (d *DoctorRegimenHandler) getRegimenSteps(w http.ResponseWriter, r *http.Request) {
	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor id from the account id "+err.Error())
		return
	}

	regimenSteps, err := d.DataApi.GetRegimenStepsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen steps for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDoctorRegimenRequestResponse{RegimenSteps: regimenSteps})
}
