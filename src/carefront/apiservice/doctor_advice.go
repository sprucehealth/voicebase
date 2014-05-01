package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"encoding/json"
	"errors"
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

	// ensure that all selected advice points are actually in the global list on the client side
	for _, selectedAdvicePoint := range requestData.SelectedAdvicePoints {
		if httpStatusCode, err := d.ensureAdvicePointExistsInMasterList(selectedAdvicePoint, &requestData, patientVisitReviewData.DoctorId); err != nil {
			WriteDeveloperError(w, httpStatusCode, err.Error())
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
	// its possible for multiple items with the exact same text to be added, which is why we maintain a mapping of
	// text to a slice of int64s
	newPointToIdMapping := make(map[string][]int64)
	updatedPointToIdMapping := make(map[int64]int64)
	updatedAdvicePoints := make([]*common.DoctorInstructionItem, 0)
	for _, advicePoint := range requestData.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_ADDED:
			err = d.DataApi.AddAdvicePointForDoctor(advicePoint, patientVisitReviewData.DoctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newPointToIdMapping[advicePoint.Text] = append(newPointToIdMapping[advicePoint.Text], advicePoint.Id.Int64())
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_MODIFIED:
			previousAdvicePointId := advicePoint.Id.Int64()
			err = d.DataApi.UpdateAdvicePointForDoctor(advicePoint, patientVisitReviewData.DoctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			updatedPointToIdMapping[previousAdvicePointId] = advicePoint.Id.Int64()
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		default:
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		}
	}

	// go through advice points to assign ids to the new points that dont have them
	for _, advicePoint := range requestData.SelectedAdvicePoints {
		if newIds, ok := newPointToIdMapping[advicePoint.Text]; ok {
			advicePoint.ParentId = encoding.NewObjectId(newIds[0])
			// remove the id that was just used so as to assign a different id to the same text
			// that could appear again
			newPointToIdMapping[advicePoint.Text] = newIds[1:]
		}
		if updatedId, ok := updatedPointToIdMapping[advicePoint.ParentId.Int64()]; ok {
			// update the parentId to point to the new updated item
			advicePoint.ParentId = encoding.NewObjectId(updatedId)
		}
	}

	err = d.DataApi.CreateAdviceForPatientVisit(requestData.SelectedAdvicePoints, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add advice for patient visit: "+err.Error())
		return
	}

	// fetch all advice points in the treatment plan and the global advice poitns to
	// return an updated view of the world to the client
	advicePoints, err := d.DataApi.GetAdvicePointsForTreatmentPlan(treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the advice points that were just created "+err.Error())
		return
	}

	allAdvicePoints, err := d.DataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get all advice points for doctor: "+err.Error())
		return
	}

	advice := &common.Advice{
		TreatmentPlanId:      encoding.NewObjectId(treatmentPlanId),
		PatientVisitId:       requestData.PatientVisitId,
		AllAdvicePoints:      allAdvicePoints,
		SelectedAdvicePoints: advicePoints,
	}

	dispatch.Default.Publish(&AdviceAddedEvent{
		TreatmentPlanId: treatmentPlanId,
		Advice:          &requestData,
	})

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, advice)
}

func (d *DoctorAdviceHandler) ensureAdvicePointExistsInMasterList(selectedAdvicePoint *common.DoctorInstructionItem, advice *common.Advice, doctorId int64) (int, error) {
	advicePointFound := false
	for _, advicePoint := range advice.AllAdvicePoints {
		if advicePoint.Id.Int64() == 0 && selectedAdvicePoint.ParentId.Int64() == 0 {
			// compare text if this is a new item
			if advicePoint.Text == selectedAdvicePoint.Text {
				advicePointFound = true
				break
			}
		} else if advicePoint.Id.Int64() == selectedAdvicePoint.ParentId.Int64() {
			// ensure that text matches up
			if advicePoint.Text != selectedAdvicePoint.Text {
				return http.StatusBadRequest, errors.New("Text of an item in the selected list that is linked to an item in the global list has to match up")
			}
			advicePointFound = true
			break
		} else if selectedAdvicePoint.ParentId.Int64() != 0 {

			parentAdvicePoint, err := d.DataApi.GetAdvicePointForDoctor(selectedAdvicePoint.ParentId.Int64(), doctorId)
			if err == api.NoRowsError {
				return http.StatusBadRequest, errors.New("No parent advice point found for advice point in the selected list")
			} else if err != nil {
				return http.StatusInternalServerError, errors.New("Unable to fetch the parent advice point for an advice point in the selected list: " + err.Error())
			}

			if parentAdvicePoint.Text != selectedAdvicePoint.Text && selectedAdvicePoint.State != common.STATE_MODIFIED {
				return http.StatusBadRequest, errors.New("Cannot modify the text for a selected item linked to a parent advice point without indicating the intent to modify with STATE=MODIFIED")
			}
			advicePointFound = true
			break
		}
	}

	if !advicePointFound {
		return http.StatusBadRequest, errors.New("There is an advice point in the selected list that is not in the global list")
	}
	return 0, nil
}
