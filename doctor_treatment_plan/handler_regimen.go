package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type regimenHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type DoctorRegimenRequestResponse struct {
	RegimenSteps     []*common.DoctorInstructionItem `json:"regimen_steps"`
	DrugInternalName string                          `json:"drug_internal_name,omitempty"`
}

func NewRegimenHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return &regimenHandler{
		dataAPI:    dataAPI,
		dispatcher: dispatcher,
	}
}

func (d *regimenHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	requestData := &common.RegimenPlan{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	} else if requestData.TreatmentPlanId.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctorId, err := d.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	patientId, err := d.dataAPI.GetPatientIdFromTreatmentPlanId(requestData.TreatmentPlanId.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientID] = patientId

	// can only add regimen for a treatment that is a draft
	treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId.Int64(), doctorId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientId, treatmentPlan.PatientCaseId.Int64(), d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *regimenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*common.RegimenPlan)

	if !treatmentPlan.InDraftMode() {
		apiservice.WriteValidationError("treatment plan must be in draft mode", w, r)
		return
	}

	// ensure that all regimen steps in the regimen sections actually exist in the client global list
	for _, regimenSection := range requestData.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			if httpStatusCode, err := d.ensureLinkedRegimenStepExistsInMasterList(regimenStep, requestData, doctorId); err != nil {
				apiservice.WriteDeveloperError(w, httpStatusCode, err.Error())
				return
			}
		}
	}

	// compare the master list of regimen steps from the client with the active list
	// that we have stored on the server
	currentActiveRegimenSteps, err := d.dataAPI.GetRegimenStepsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	regimenStepsToDelete := make([]*common.DoctorInstructionItem, 0, len(currentActiveRegimenSteps))
	for _, currentRegimenStep := range currentActiveRegimenSteps {
		regimenStepFound := false
		for _, regimenStep := range requestData.AllRegimenSteps {
			if regimenStep.Id.Int64() == currentRegimenStep.Id.Int64() {
				regimenStepFound = true
				break
			}
		}
		if !regimenStepFound {
			regimenStepsToDelete = append(regimenStepsToDelete, currentRegimenStep)
		}
	}
	err = d.dataAPI.MarkRegimenStepsToBeDeleted(regimenStepsToDelete, doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Go through regimen steps to add and update regimen steps before creating the regimen plan
	// for the user
	newStepToIdMapping := make(map[string][]int64)
	// keep track of the multiple items that could have the exact same text associated with it
	updatedStepToIdMapping := make(map[int64]int64)
	updatedAllRegimenSteps := make([]*common.DoctorInstructionItem, 0)
	for _, regimenStep := range requestData.AllRegimenSteps {
		switch regimenStep.State {
		case common.STATE_ADDED:
			err = d.dataAPI.AddRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add reigmen step to doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newStepToIdMapping[regimenStep.Text] = append(newStepToIdMapping[regimenStep.Text], regimenStep.Id.Int64())
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		case common.STATE_MODIFIED:
			previousRegimenStepId := regimenStep.Id.Int64()
			err = d.dataAPI.UpdateRegimenStepForDoctor(regimenStep, doctorId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update regimen step for doctor: "+err.Error())
				return
			}
			// keep track of the new id for updated regimen steps so that we can update the regimen step in the
			// regimen section
			updatedStepToIdMapping[previousRegimenStepId] = regimenStep.Id.Int64()
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		default:
			updatedAllRegimenSteps = append(updatedAllRegimenSteps, regimenStep)
		}
	}

	// go through regimen steps within the regimen sections to assign ids to the new steps that dont have them
	for _, regimenSection := range requestData.RegimenSections {

		for _, regimenStep := range regimenSection.RegimenSteps {

			if newIds, ok := newStepToIdMapping[regimenStep.Text]; ok {
				regimenStep.ParentId = encoding.NewObjectId(newIds[0])
				// update the list to move the item just used to the back of the queue
				newStepToIdMapping[regimenStep.Text] = append(newIds[1:], newIds[0])
			} else if updatedId, ok := updatedStepToIdMapping[regimenStep.ParentId.Int64()]; ok {
				// update the parentId to point to the new updated regimen step
				regimenStep.ParentId = encoding.NewObjectId(updatedId)
			} else if regimenStep.State == common.STATE_MODIFIED || regimenStep.State == common.STATE_ADDED {
				// break any linkage to the parent step because the text is no longer the same and the regimen step does
				// not exist in the master list
				regimenStep.ParentId = encoding.ObjectId{}
			}
		}
	}

	err = d.dataAPI.CreateRegimenPlanForTreatmentPlan(requestData)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create regimen plan for patient visit: "+err.Error())
		return
	}

	// fetch all regimen steps in the treatment plan and the global regimen steps to
	// return an updated view of the world to the client
	regimenPlan, err := d.dataAPI.GetRegimenPlanForTreatmentPlan(requestData.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the regimen plan for treatment plan: "+err.Error())
		return
	}

	allRegimenSteps, err := d.dataAPI.GetRegimenStepsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the list of regimen steps for doctor: "+err.Error())
		return
	}

	regimenPlan = &common.RegimenPlan{
		RegimenSections: regimenPlan.RegimenSections,
		AllRegimenSteps: allRegimenSteps,
		TreatmentPlanId: requestData.TreatmentPlanId,
		Status:          api.STATUS_COMMITTED,
	}

	d.dispatcher.PublishAsync(&RegimenPlanAddedEvent{
		TreatmentPlanId: requestData.TreatmentPlanId.Int64(),
		RegimenPlan:     requestData,
		DoctorId:        doctorId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, regimenPlan)
}

func (d *regimenHandler) ensureLinkedRegimenStepExistsInMasterList(regimenStep *common.DoctorInstructionItem, regimenPlan *common.RegimenPlan, doctorId int64) (int, error) {
	// no need to check if the regimen step does not indicate that it exists in the master list
	if !regimenStep.ParentId.IsValid {
		return http.StatusOK, nil
	}

	// search for the regimen step against the current master list returned from the client
	for _, globalRegimenStep := range regimenPlan.AllRegimenSteps {

		if !globalRegimenStep.Id.IsValid {
			continue
		}

		// break the linkage if the text doesn't match
		if globalRegimenStep.Id.Int64() == regimenStep.ParentId.Int64() {
			if globalRegimenStep.Text != regimenStep.Text {
				regimenStep.ParentId = encoding.ObjectId{}
			}
			return http.StatusOK, nil
		}
	}

	// its possible that the step is not present in the active global list but exists as a
	// step from the past
	parentRegimenStep, err := d.dataAPI.GetRegimenStepForDoctor(regimenStep.ParentId.Int64(), doctorId)
	if err != nil {
		regimenStep.ParentId = encoding.ObjectId{}
	}

	// if the parent regimen step does exist, ensure that the text matches up, and if not break the linkage
	if parentRegimenStep.Text != regimenStep.Text && regimenStep.State != common.STATE_MODIFIED {
		regimenStep.ParentId = encoding.ObjectId{}
	}

	return http.StatusOK, nil
}
