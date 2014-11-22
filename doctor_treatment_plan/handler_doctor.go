package doctor_treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
)

type doctorTreatmentPlanHandler struct {
	dataApi         api.DataAPI
	erxAPI          erx.ERxAPI
	dispatcher      *dispatch.Dispatcher
	erxRoutingQueue *common.SQSQueue
	erxStatusQueue  *common.SQSQueue
	routeErx        bool
}

func NewDoctorTreatmentPlanHandler(dataApi api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher, erxRoutingQueue *common.SQSQueue, erxStatusQueue *common.SQSQueue, routeErx bool) *doctorTreatmentPlanHandler {
	return &doctorTreatmentPlanHandler{
		dataApi:         dataApi,
		erxAPI:          erxAPI,
		dispatcher:      dispatcher,
		erxRoutingQueue: erxRoutingQueue,
		erxStatusQueue:  erxStatusQueue,
		routeErx:        routeErx,
	}
}

type TreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId int64                              `json:"dr_favorite_treatment_plan_id,string" schema:"dr_favorite_treatment_plan_id"`
	TreatmentPlanID               int64                              `json:"treatment_plan_id,string" schema:"treatment_plan_id" `
	PatientVisitId                int64                              `json:"patient_visit_id,string" schema:"patient_visit_id" `
	Abridged                      bool                               `json:"abridged" schema:"abridged"`
	TPContentSource               *common.TreatmentPlanContentSource `json:"content_source"`
	TPParent                      *common.TreatmentPlanParent        `json:"parent"`
	Message                       string                             `json:"message"`
}

type DoctorTreatmentPlanResponse struct {
	TreatmentPlan *common.DoctorTreatmentPlan `json:"treatment_plan"`
}

func (d *doctorTreatmentPlanHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &TreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	switch r.Method {
	case apiservice.HTTP_GET:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
		}

		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID, doctorId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}

		// if we are dealing with a draft, and the owner of the treatment plan does not match the doctor requesting it,
		// return an error because this should never be the case
		if treatmentPlan.InDraftMode() && treatmentPlan.DoctorId.Int64() != doctorId {
			return false, apiservice.NewAccessForbiddenError()
		}

	case apiservice.HTTP_PUT, apiservice.HTTP_DELETE:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
		}

		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID, doctorId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}

		// ensure that doctor is owner of the treatment plan
		if doctorId != treatmentPlan.DoctorId.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

	case apiservice.HTTP_POST:
		if requestData.TPParent == nil || requestData.TPParent.ParentId.Int64() == 0 {
			return false, apiservice.NewValidationError("parent_id must be specified", r)
		}

		patientVisitId := requestData.TPParent.ParentId.Int64()
		switch requestData.TPParent.ParentType {
		case common.TPParentTypeTreatmentPlan:
			// ensure that parent treatment plan is ACTIVE
			parentTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TPParent.ParentId.Int64(), doctorId)
			if err != nil {
				return false, err
			} else if parentTreatmentPlan.Status != api.STATUS_ACTIVE {
				return false, apiservice.NewValidationError("parent treatment plan has to be ACTIVE", r)
			}

			patientVisitId, err = d.dataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TPParent.ParentId.Int64())
			if err != nil {
				return false, err
			}
		case common.TPParentTypePatientVisit:
		default:
			return false, apiservice.NewValidationError("Expected the parent type to either by PATIENT_VISIT or TREATMENT_PLAN", r)
		}
		ctxt.RequestCache[apiservice.PatientVisitID] = patientVisitId

		patientCase, err := d.dataApi.GetPatientCaseFromPatientVisitId(patientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientCase] = patientCase

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), d.dataApi); err != nil {
			return false, err
		}

	default:
		return false, nil
	}

	return true, nil
}

func (d *doctorTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getTreatmentPlan(w, r)
	case apiservice.HTTP_POST:
		d.pickATreatmentPlan(w, r)
	case apiservice.HTTP_PUT:
		d.submitTreatmentPlan(w, r)
	case apiservice.HTTP_DELETE:
		d.deleteTreatmentPlan(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *doctorTreatmentPlanHandler) deleteTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)

	// Ensure treatment plan is a draft
	if !treatmentPlan.InDraftMode() {
		apiservice.WriteValidationError("only draft treatment plan can be deleted", w, r)
		return
	}

	// Delete treatment plan
	if err := d.dataApi.DeleteTreatmentPlan(treatmentPlan.Id.Int64()); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (d *doctorTreatmentPlanHandler) submitTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)

	if requestData.Message == "" {
		apiservice.WriteValidationError("Please include a Personal Note to the patient before submitting the Treatment Plan.", w, r)
		return
	}

	var patientVisitId int64
	var err error
	switch treatmentPlan.Parent.ParentType {
	case common.TPParentTypePatientVisit:
		// if the parent of this treatment plan is a patient visit, this means that this is the first
		// treatment plan. In this case we expect the patient visit to be in the REVIEWING state.
		patientVisitId = treatmentPlan.Parent.ParentId.Int64()
		if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataApi, patientVisitId, common.PVStatusReviewing); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case common.TPParentTypeTreatmentPlan:
		patientVisitId, err = d.dataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TreatmentPlanID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// if the parent of the treatment plan is a previous version of a treatment plan, ensure that it is an ACTIVE
		// treatment plan
		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(treatmentPlan.Parent.ParentId.Int64(), treatmentPlan.DoctorId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if treatmentPlan.Status != api.STATUS_ACTIVE {
			apiservice.WriteValidationError(fmt.Sprintf("Expected the parent treatment plan to be in the active state but its in %s state", treatmentPlan.Status), w, r)
			return
		}

	default:
		apiservice.WriteValidationError(fmt.Sprintf("Parent of treatment plan is unexpected parent of type %s", treatmentPlan.Parent.ParentType), w, r)
		return
	}

	// mark the treatment plan as submitted
	if err := d.dataApi.UpdateTreatmentPlanStatus(treatmentPlan.Id.Int64(), common.TPStatusSubmitted); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	d.dispatcher.Publish(&TreatmentPlanSubmittedEvent{
		VisitId:       patientVisitId,
		TreatmentPlan: treatmentPlan,
	})

	if d.routeErx {
		apiservice.QueueUpJob(d.erxRoutingQueue, &erxRouteMessage{
			TreatmentPlanID: requestData.TreatmentPlanID,
			PatientID:       treatmentPlan.PatientId,
			DoctorID:        treatmentPlan.DoctorId.Int64(),
			Message:         requestData.Message,
		})
	} else {
		if err := d.dataApi.ActivateTreatmentPlan(treatmentPlan.Id.Int64(), treatmentPlan.DoctorId.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		doctor, err := d.dataApi.GetDoctorFromId(treatmentPlan.DoctorId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := sendCaseMessageAndPublishTPActivatedEvent(d.dataApi, d.dispatcher, treatmentPlan, doctor, requestData.Message); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *doctorTreatmentPlanHandler) getTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)

	// only return the small amount of information retreived about the treatment plan
	if requestData.Abridged {
		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: treatmentPlan})
		return
	}

	if err := fillInTreatmentPlan(treatmentPlan, doctorId, d.dataApi); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: treatmentPlan})
}

func (d *doctorTreatmentPlanHandler) pickATreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	patientVisitId := ctxt.RequestCache[apiservice.PatientVisitID].(int64)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)

	if requestData.TPContentSource != nil {
		if requestData.TPContentSource.ContentSourceType != common.TPContentSourceTypeFTP && requestData.TPContentSource.ContentSourceType != common.TPContentSourceTypeTreatmentPlan {
			apiservice.WriteValidationError(fmt.Sprintf("Expected content source type be either FAVORITE_TREATMENT_PLAN or TREATMENT_PLAN but instead it was %s", requestData.TPContentSource.ContentSourceType), w, r)
			return
		}
	}

	treatmentPlanId, err := d.dataApi.StartNewTreatmentPlan(patientCase.PatientId.Int64(),
		patientVisitId, doctorId, requestData.TPParent, requestData.TPContentSource)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start new treatment plan for patient visit: "+err.Error())
		return
	}

	// get the treatment plan just created
	drTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(treatmentPlanId, doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := fillInTreatmentPlan(drTreatmentPlan, doctorId, d.dataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	d.dispatcher.Publish(&NewTreatmentPlanStartedEvent{
		PatientID:       drTreatmentPlan.PatientId,
		DoctorID:        doctorId,
		CaseID:          drTreatmentPlan.PatientCaseId.Int64(),
		VisitID:         patientVisitId,
		TreatmentPlanID: treatmentPlanId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}
