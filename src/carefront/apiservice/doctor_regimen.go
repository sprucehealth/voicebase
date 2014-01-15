package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorRegimenHandler struct {
	DataApi api.DataAPI
}

type GetDoctorRegimenRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

type DoctorRegimenRequestResponse struct {
	RegimenSteps     []*common.DoctorInstructionItem `json:"regimen_steps"`
	DrugInternalName string                          `json:"drug_internal_name,omitempty"`
	PatientVisitId   int64                           `json:"patient_visit_id,string,omitempty"`
}

func NewDoctorRegimenHandler(dataApi api.DataAPI) *DoctorRegimenHandler {
	return &DoctorRegimenHandler{DataApi: dataApi}
}

func (d *DoctorRegimenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getRegimenSteps(w, r)
	case "POST":
		d.updateRegimenSteps(w, r)
	}
}

func (d *DoctorRegimenHandler) getRegimenSteps(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(GetDoctorRegimenRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	regimenSteps, err := d.DataApi.GetRegimenStepsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen steps for doctor: "+err.Error())
		return
	}

	regimenPlan, err := d.DataApi.GetRegimenPlanForPatientVisit(requestData.PatientVisitId)
	if err != nil && err != api.NoRegimenPlanForPatientVisit {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to lookup regimen plan for patient visit: "+err.Error())
	}

	if regimenPlan == nil {
		regimenPlan = &common.RegimenPlan{}
	}
	regimenPlan.AllRegimenSteps = regimenSteps

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, regimenPlan)
}

func (d *DoctorRegimenHandler) updateRegimenSteps(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	requestData := &common.RegimenPlan{}

	err := jsonDecoder.Decode(requestData)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating regimen steps: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(d.DataApi, requestData.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Go through regimen steps to add, update and delete regimen steps before creating the regimen plan
	// for the user
	newOrUpdatedStepToIdMapping := make(map[string]int64)
	updatedAllRegimenSteps := make([]*common.DoctorInstructionItem, 0)
	for _, regimenStep := range requestData.AllRegimenSteps {
		switch regimenStep.State {
		case common.STATE_ADDED:
			err = d.DataApi.AddRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add reigmen step to doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newOrUpdatedStepToIdMapping[regimenStep.Text] = regimenStep.Id
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		case common.STATE_MODIFIED:
			err = d.DataApi.UpdateRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update regimen step for doctor: "+err.Error())
				return
			}
			// keep track of the new id for updated regimen steps so that we can update the regimen step in the
			// regimen section
			newOrUpdatedStepToIdMapping[regimenStep.Text] = regimenStep.Id
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		case common.STATE_DELETED:
			err = d.DataApi.MarkRegimenStepToBeDeleted(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete regimen step for doctor: "+err.Error())
				return
			}
		default:
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		}
		// empty out the state now that it has been taken care of
		regimenStep.State = ""
	}

	// go through regimen steps within the regimen sections to assign ids to the new steps that dont have them
	for _, regimenSection := range requestData.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			updatedOrNewId := newOrUpdatedStepToIdMapping[regimenStep.Text]
			if updatedOrNewId != 0 {
				regimenStep.Id = updatedOrNewId
			}
		}
	}

	err = d.DataApi.CreateRegimenPlanForPatientVisit(requestData)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create regimen plan for patient visit: "+err.Error())
		return
	}

	requestData.AllRegimenSteps = updatedAllRegimenSteps
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, requestData)
}
