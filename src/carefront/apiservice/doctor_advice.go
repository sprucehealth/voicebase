package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorAdviceHandler struct {
	DataApi api.DataAPI
}

type GetDoctorAdviceRequestData struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

func NewDoctorAdviceHandler(dataApi api.DataAPI) *DoctorAdviceHandler {
	return &DoctorAdviceHandler{DataApi: dataApi}
}

func (d *DoctorAdviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getAdvicePoints(w, r)
	case "POST":
		d.updateAdvicePoints(w, r)
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

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	err = ensureTreatmentPlanOrPatientVisitIdPresent(d.DataApi, &treatmentPlanId, &patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	advicePoints, err := d.DataApi.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor: "+err.Error())
		return
	}

	if treatmentPlanId == 0 {
		treatmentPlanId, err = d.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
			return
		}
	}

	selectedAdvicePoints, err := d.DataApi.GetAdvicePointsForPatientVisit(treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the selected advice points for this patient visit: "+err.Error())
		return
	}

	responseData := &common.Advice{}
	responseData.AllAdvicePoints = advicePoints
	responseData.SelectedAdvicePoints = selectedAdvicePoints
	responseData.PatientVisitId = patientVisitId
	responseData.TreatmentPlanId = requestData.TreatmentPlanId

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, responseData)
}

func (d *DoctorAdviceHandler) updateAdvicePoints(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	requestData := &common.Advice{}

	err := jsonDecoder.Decode(requestData)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating advice points: "+err.Error())
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

	treatmentPlanId, err := d.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	// first, ensure that all selected advice points are actually in the global list on the client side
	for _, selectedAdvicePoint := range requestData.SelectedAdvicePoints {
		advicePointFound := false
		for _, advicePoint := range requestData.AllAdvicePoints {
			if advicePoint.Id == 0 {
				if advicePoint.Text == selectedAdvicePoint.Text {
					advicePointFound = true
					break
				}
			} else if advicePoint.Id == selectedAdvicePoint.Id {
				advicePointFound = true
				break
			}
		}
		if !advicePointFound {
			WriteDeveloperError(w, http.StatusBadRequest, "There is an advice point in the selected list that is not in the global list")
			return
		}
	}

	currentActiveAdvicePoints, err := d.DataApi.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active advice points for the doctor")
		return
	}

	advicePointsToDelete := make([]*common.DoctorInstructionItem, 0)
	for _, currentAdvicePoint := range currentActiveAdvicePoints {
		// now, search for whether this particular item (based on the id) is present on the list coming from the client
		advicePointFound := false
		for _, advicePointFromClient := range requestData.AllAdvicePoints {
			if currentAdvicePoint.Id == advicePointFromClient.Id {
				advicePointFound = true
				break
			}
		}
		if !advicePointFound {
			advicePointsToDelete = append(advicePointsToDelete, currentAdvicePoint)
		}
	}

	// mark all advice points that are not present in the list coming from the client to be deleted
	err = d.DataApi.MarkAdvicePointsToBeDeleted(advicePointsToDelete, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete advice points: "+err.Error())
		return
	}

	// Go through advice points to add, update and delete advice points before creating the advice points for this patient visit
	// for the user
	newOrUpdatedPointToIdMapping := make(map[string]int64)
	updatedAdvicePoints := make([]*common.DoctorInstructionItem, 0)
	for _, advicePoint := range requestData.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_ADDED, common.STATE_MODIFIED:
			err = d.DataApi.AddOrUpdateAdvicePointForDoctor(advicePoint, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newOrUpdatedPointToIdMapping[advicePoint.Text] = advicePoint.Id
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_DELETED:
			err = d.DataApi.MarkAdvicePointToBeDeleted(advicePoint, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete advice point for doctor: "+err.Error())
				return
			}
		default:
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		}
		// empty out the state now that it has been taken care of
		advicePoint.State = ""
	}

	// go through advice points to assign ids to the new points that dont have them
	for _, advicePoint := range requestData.SelectedAdvicePoints {
		updatedOrNewId := newOrUpdatedPointToIdMapping[advicePoint.Text]
		if updatedOrNewId != 0 {
			advicePoint.Id = updatedOrNewId
		}
		// empty out the state information given that it is taken care of
		advicePoint.State = ""
	}

	err = d.DataApi.CreateAdviceForPatientVisit(requestData.SelectedAdvicePoints, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add advice for patient visit: "+err.Error())
		return
	}

	requestData.AllAdvicePoints = updatedAdvicePoints
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, requestData)
}
