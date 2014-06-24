package doctor_treatment_plan

import (
	"carefront/accessmgmt"
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"errors"
	"net/http"
)

type adviceHandler struct {
	dataAPI api.DataAPI
}

func NewAdviceHandler(dataAPI api.DataAPI) *adviceHandler {
	return &adviceHandler{
		dataAPI: dataAPI,
	}
}

func (d *adviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_POST:
		d.updateAdvicePoints(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *adviceHandler) updateAdvicePoints(w http.ResponseWriter, r *http.Request) {
	var requestData common.Advice
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating advice points: "+err.Error())
		return
	} else if requestData.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment plan id needs to be specified to know which treatment plan to add advice to", w, r)
		return
	}

	patientId, err := d.dataAPI.GetPatientIdFromTreatmentPlanId(requestData.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorId, err := d.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	statusCode, err := accessmgmt.ValidateDoctorAccessToPatientFile(doctorId, patientId, d.dataAPI)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// can only add regimen for a treatment that is a draft
	treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId.Int64(), doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if treatmentPlan.Status != api.STATUS_DRAFT {
		apiservice.WriteValidationError("treatment plan must be in draft mode", w, r)
		return
	}

	// ensure that all selected advice points are actually in the global list on the client side
	for _, selectedAdvicePoint := range requestData.SelectedAdvicePoints {
		if httpStatusCode, err := d.ensureLinkedAdvicePointExistsInMasterList(selectedAdvicePoint, &requestData, doctorId); err != nil {
			apiservice.WriteDeveloperError(w, httpStatusCode, err.Error())
			return
		}
	}

	currentActiveAdvicePoints, err := d.dataAPI.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active advice points for the doctor")
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
	err = d.dataAPI.MarkAdvicePointsToBeDeleted(advicePointsToDelete, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete advice points: "+err.Error())
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
			err = d.dataAPI.AddAdvicePointForDoctor(advicePoint, doctorId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newPointToIdMapping[advicePoint.Text] = append(newPointToIdMapping[advicePoint.Text], advicePoint.Id.Int64())
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_MODIFIED:
			previousAdvicePointId := advicePoint.Id.Int64()
			err = d.dataAPI.UpdateAdvicePointForDoctor(advicePoint, doctorId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
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
			// move the id that was just used to the back of the queue
			// so as to assign a different id to the same text that could appear again
			newPointToIdMapping[advicePoint.Text] = append(newIds[1:], newIds[0])
		} else if updatedId, ok := updatedPointToIdMapping[advicePoint.ParentId.Int64()]; ok {
			// update the parentId to point to the new updated item
			advicePoint.ParentId = encoding.NewObjectId(updatedId)
		} else if advicePoint.State == common.STATE_MODIFIED || advicePoint.State == common.STATE_ADDED {
			// break any existing linkage given that the text has been modified and is no longer the same as
			// the parent step
			advicePoint.ParentId = encoding.ObjectId{}
		}
	}

	err = d.dataAPI.CreateAdviceForTreatmentPlan(requestData.SelectedAdvicePoints, requestData.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add advice for patient visit: "+err.Error())
		return
	}

	// fetch all advice points in the treatment plan and the global advice poitns to
	// return an updated view of the world to the client
	advicePoints, err := d.dataAPI.GetAdvicePointsForTreatmentPlan(requestData.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the advice points that were just created "+err.Error())
		return
	}

	allAdvicePoints, err := d.dataAPI.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get all advice points for doctor: "+err.Error())
		return
	}

	advice := &common.Advice{
		AllAdvicePoints:      allAdvicePoints,
		SelectedAdvicePoints: advicePoints,
		Status:               api.STATUS_COMMITTED,
	}

	dispatch.Default.PublishAsync(&AdviceAddedEvent{
		TreatmentPlanId: requestData.TreatmentPlanId.Int64(),
		Advice:          &requestData,
		DoctorId:        doctorId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, advice)
}

func (d *adviceHandler) ensureLinkedAdvicePointExistsInMasterList(selectedAdvicePoint *common.DoctorInstructionItem, advice *common.Advice, doctorId int64) (int, error) {
	// nothing to do if the advice point does not exist in the master list
	if !selectedAdvicePoint.ParentId.IsValid {
		return 0, nil
	}

	for _, advicePoint := range advice.AllAdvicePoints {
		if !advicePoint.Id.IsValid {
			continue
		}

		if advicePoint.Id.Int64() == selectedAdvicePoint.ParentId.Int64() {
			// ensure that text matches up
			if advicePoint.Text != selectedAdvicePoint.Text {
				return http.StatusBadRequest, errors.New("Text of an item in the selected list that is linked to an item in the global list has to match up")
			}
			break
		} else {
			parentAdvicePoint, err := d.dataAPI.GetAdvicePointForDoctor(selectedAdvicePoint.ParentId.Int64(), doctorId)
			if err == api.NoRowsError {
				return http.StatusBadRequest, errors.New("No parent advice point found for advice point in the selected list")
			} else if err != nil {
				return http.StatusInternalServerError, errors.New("Unable to fetch the parent advice point for an advice point in the selected list: " + err.Error())
			}

			if parentAdvicePoint.Text != selectedAdvicePoint.Text && selectedAdvicePoint.State != common.STATE_MODIFIED {
				return http.StatusBadRequest, errors.New("Cannot modify the text for a selected item linked to a parent advice point without indicating the intent to modify with STATE=MODIFIED")
			}
			break
		}
	}

	return 0, nil
}
