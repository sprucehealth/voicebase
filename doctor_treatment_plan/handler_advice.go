package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

type adviceHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

func NewAdviceHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return &adviceHandler{
		dataAPI:    dataAPI,
		dispatcher: dispatcher,
	}
}

func (d *adviceHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	ctxt := apiservice.GetContext(r)

	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	var requestData common.Advice
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	} else if requestData.TreatmentPlanID.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientId, err := d.dataAPI.GetPatientIdFromTreatmentPlanId(requestData.TreatmentPlanID.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientID] = patientId

	doctorId, err := d.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	// can only add regimen for a treatment that is a draft
	treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID.Int64(), doctorId)
	if err != nil {
		return false, err
	} else if !treatmentPlan.InDraftMode() {
		return false, apiservice.NewValidationError("treatment plan must be in draft mode", r)
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientId, treatmentPlan.PatientCaseId.Int64(), d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *adviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	requestData := ctxt.RequestCache[apiservice.RequestData].(common.Advice)

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
			if currentAdvicePoint.ID.Int64() == advicePointFromClient.ID.Int64() {
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
			newPointToIdMapping[advicePoint.Text] = append(newPointToIdMapping[advicePoint.Text], advicePoint.ID.Int64())
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_MODIFIED:
			previousAdvicePointId := advicePoint.ID.Int64()
			err = d.dataAPI.UpdateAdvicePointForDoctor(advicePoint, doctorId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			updatedPointToIdMapping[previousAdvicePointId] = advicePoint.ID.Int64()
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		default:
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		}
	}

	// go through advice points to assign ids to the new points that dont have them
	for _, advicePoint := range requestData.SelectedAdvicePoints {
		if newIds, ok := newPointToIdMapping[advicePoint.Text]; ok {
			advicePoint.ParentID = encoding.NewObjectId(newIds[0])
			// move the id that was just used to the back of the queue
			// so as to assign a different id to the same text that could appear again
			newPointToIdMapping[advicePoint.Text] = append(newIds[1:], newIds[0])
		} else if updatedId, ok := updatedPointToIdMapping[advicePoint.ParentID.Int64()]; ok {
			// update the parentId to point to the new updated item
			advicePoint.ParentID = encoding.NewObjectId(updatedId)
		} else if advicePoint.State == common.STATE_MODIFIED || advicePoint.State == common.STATE_ADDED {
			// break any existing linkage given that the text has been modified and is no longer the same as
			// the parent step
			advicePoint.ParentID = encoding.ObjectId{}
		}
	}

	err = d.dataAPI.CreateAdviceForTreatmentPlan(requestData.SelectedAdvicePoints, requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add advice for patient visit: "+err.Error())
		return
	}

	// fetch all advice points in the treatment plan and the global advice poitns to
	// return an updated view of the world to the client
	advicePoints, err := d.dataAPI.GetAdvicePointsForTreatmentPlan(requestData.TreatmentPlanID.Int64())
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

	d.dispatcher.PublishAsync(&AdviceAddedEvent{
		TreatmentPlanID: requestData.TreatmentPlanID.Int64(),
		Advice:          &requestData,
		DoctorId:        doctorId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, advice)
}

func (d *adviceHandler) ensureLinkedAdvicePointExistsInMasterList(selectedAdvicePoint *common.DoctorInstructionItem, advice *common.Advice, doctorId int64) (int, error) {
	// nothing to do if the advice point does not exist in the master list
	if !selectedAdvicePoint.ParentID.IsValid {
		return http.StatusOK, nil
	}

	for _, advicePoint := range advice.AllAdvicePoints {
		if !advicePoint.ID.IsValid {
			continue
		}

		if advicePoint.ID.Int64() == selectedAdvicePoint.ParentID.Int64() {
			// ensure that text matches up, and if not, break the linkage
			if advicePoint.Text != selectedAdvicePoint.Text {
				selectedAdvicePoint.ParentID = encoding.ObjectId{}
			}
			return http.StatusOK, nil
		}
	}

	// if the advice point was not found in the current master advice list, then its probably an older advice step
	parentAdvicePoint, err := d.dataAPI.GetAdvicePointForDoctor(selectedAdvicePoint.ParentID.Int64(), doctorId)
	// break the linkage if there was an error getting the parent step
	if err != nil {
		selectedAdvicePoint.ParentID = encoding.ObjectId{}
		golog.Warningf("Unable to get parent advice step: %s", err)
	}

	// break the linkage if the text is modified but the state of the step does not indicate so
	if parentAdvicePoint.Text != selectedAdvicePoint.Text && selectedAdvicePoint.State != common.STATE_MODIFIED {
		selectedAdvicePoint.ParentID = encoding.ObjectId{}
	}

	return http.StatusOK, nil
}
