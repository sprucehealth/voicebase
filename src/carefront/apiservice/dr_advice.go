package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"github.comm/gorilla/schema"
	"net/http"
)

type DoctorAdviceHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type GetDoctorAdviceRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

type DoctorAdviceRequestResponse struct {
	AllAdvicePoints      []*common.DoctorInstructionItem `json:"all_advice_points"`
	SelectedAdvicePoints []*common.DoctorInstructionItem `json:"selected_advice_points,omitempty"`
	PatientVisitId       int64                           `json:"patient_visit_id,omitempty"`
}

func NewDoctorAdviceHandler(dataApi api.DataAPI) *DoctorAdviceHandler {
	return &DoctorAdviceHandler{DataApi: dataApi, accountId: 0}
}

func (d *DoctorAdviceHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorAdviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getAdvicePoints(w, r)
	}
}

func (d *DoctorAdviceHandler) getAdvicePoints(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(GetDoctorAdviceRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	advicePoints, err := d.DataApi.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor: "+err.Error())
		return
	}

	selectedAdvicePoints, err := d.DataApi.GetAdvicePointsForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the selected advice points for this patient visit: "+err.Error())
		return
	}

	responseData := &DoctorAdviceRequestResponse{}
	responseData.AllAdvicePoints = advicePoints
	responseData.SelectedAdvicePoints = selectedAdvicePoints
	responseData.PatientVisitId = requestData.PatientVisitId

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, responseData)
}

func (d *DoctorAdviceHandler) updateAdvicePoints(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	requestData := &DoctorAdviceRequestResponse{}

	err := jsonDecoder.Decode(requestData)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating advice points: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// Go through regimen steps to add, update and delete regimen steps before creating the regimen plan
	// for the user
	newOrUpdatedPointToIdMapping := make(map[string]int64)
	for _, advicePoint := range requestData.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_ADDED:
			err = d.DataApi.AddRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add reigmen step to doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newOrUpdatedStepToIdMapping[regimenStep.Text] = regimenStep.Id
		case common.STATE_MODIFIED:
			err = d.DataApi.UpdateRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update regimen step for doctor: "+err.Error())
				return
			}
			// keep track of the new id for updated regimen steps so that we can update the regimen step in the
			// regimen section
			newOrUpdatedStepToIdMapping[regimenStep.Text] = regimenStep.Id
		case common.STATE_DELETED:
			err = d.DataApi.MarkRegimenStepToBeDeleted(regimenStep, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete regimen step for doctor: "+err.Error())
				return
			}
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
}
