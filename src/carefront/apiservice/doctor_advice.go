package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
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
	case HTTP_GET:
		d.getAdvicePoints(w, r)
	case HTTP_POST:
		d.updateAdvicePoints(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *DoctorAdviceHandler) getAdvicePoints(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData GetDoctorAdviceRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := ensureTreatmentPlanOrPatientVisitIdPresent(d.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	advicePoints, err := d.DataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor: "+err.Error())
		return
	}

	if treatmentPlanId == 0 {
		treatmentPlanId, err = d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
			return
		}
	}

	selectedAdvicePoints, err := d.DataApi.GetAdvicePointsForTreatmentPlan(treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the selected advice points for this patient visit: "+err.Error())
		return
	}

	responseData := &common.Advice{
		AllAdvicePoints:      advicePoints,
		SelectedAdvicePoints: selectedAdvicePoints,
		PatientVisitId:       encoding.NewObjectId(patientVisitId),
		TreatmentPlanId:      encoding.NewObjectId(requestData.TreatmentPlanId),
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, responseData)
}

func (d *DoctorAdviceHandler) updateAdvicePoints(w http.ResponseWriter, r *http.Request) {
	var requestData common.Advice
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating advice points: "+err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId.Int64(), GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(d.DataApi, requestData.PatientVisitId.Int64(), api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	treatmentPlanId, err := d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, requestData.PatientVisitId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	// first, ensure that all selected advice points are actually in the global list on the client side
	for _, selectedAdvicePoint := range requestData.SelectedAdvicePoints {
		advicePointFound := false
		for _, advicePoint := range requestData.AllAdvicePoints {
			if advicePoint.Id.Int64() == 0 {
				if advicePoint.Text == selectedAdvicePoint.Text {
					advicePointFound = true
					break
				}
			} else if advicePoint.Id.Int64() == selectedAdvicePoint.Id.Int64() {
				advicePointFound = true
				break
			}
		}
		if !advicePointFound {
			WriteDeveloperError(w, http.StatusBadRequest, "There is an advice point in the selected list that is not in the global list")
			return
		}
	}

	currentActiveAdvicePoints, err := d.DataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active advice points for the doctor")
		return
	}

	advicePointsToDelete := make([]*common.DoctorInstructionItem, 0)
	for _, currentAdvicePoint := range currentActiveAdvicePoints {
		// now, search for whether this particular item (based on the id) is present on the list coming from the client
		advicePointFound := false
		for _, advicePointFromClient := range requestData.AllAdvicePoints {
			if currentAdvicePoint.Id.Int64() == advicePointFromClient.Id.Int64() {
				advicePointFound = true
				break
			}
		}
		if !advicePointFound {
			advicePointsToDelete = append(advicePointsToDelete, currentAdvicePoint)
		}
	}

	// mark all advice points that are not present in the list coming from the client to be deleted
	err = d.DataApi.MarkAdvicePointsToBeDeleted(advicePointsToDelete, patientVisitReviewData.DoctorId)
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
			err = d.DataApi.AddOrUpdateAdvicePointForDoctor(advicePoint, patientVisitReviewData.DoctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newOrUpdatedPointToIdMapping[advicePoint.Text] = advicePoint.Id.Int64()
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_DELETED:
			err = d.DataApi.MarkAdvicePointToBeDeleted(advicePoint, patientVisitReviewData.DoctorId)
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
			advicePoint.Id = encoding.NewObjectId(updatedOrNewId)
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
