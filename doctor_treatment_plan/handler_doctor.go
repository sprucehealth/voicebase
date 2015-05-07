package doctor_treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
)

type doctorTreatmentPlanHandler struct {
	dataAPI         api.DataAPI
	mediaStore      *media.Store
	erxAPI          erx.ERxAPI
	dispatcher      *dispatch.Dispatcher
	erxRoutingQueue *common.SQSQueue
	erxStatusQueue  *common.SQSQueue
	routeErx        bool
}

func NewDoctorTreatmentPlanHandler(
	dataAPI api.DataAPI,
	erxAPI erx.ERxAPI,
	mediaStore *media.Store,
	dispatcher *dispatch.Dispatcher,
	erxRoutingQueue *common.SQSQueue,
	erxStatusQueue *common.SQSQueue,
	routeErx bool,
) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&doctorTreatmentPlanHandler{
				dataAPI:         dataAPI,
				erxAPI:          erxAPI,
				mediaStore:      mediaStore,
				dispatcher:      dispatcher,
				erxRoutingQueue: erxRoutingQueue,
				erxStatusQueue:  erxStatusQueue,
				routeErx:        routeErx,
			}), []string{"GET", "PUT", "POST", "DELETE"})
}

type TreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanID int64                              `json:"dr_favorite_treatment_plan_id,string" schema:"dr_favorite_treatment_plan_id"`
	TreatmentPlanID               int64                              `json:"treatment_plan_id,string" schema:"treatment_plan_id" `
	PatientVisitID                int64                              `json:"patient_visit_id,string" schema:"patient_visit_id" `
	Abridged                      bool                               `json:"abridged" schema:"abridged"`
	TPContentSource               *common.TreatmentPlanContentSource `json:"content_source"`
	TPParent                      *common.TreatmentPlanParent        `json:"parent"`
	Message                       string                             `json:"message"`
	Sections                      string                             `json:"sections,omitempty"`
}

type DoctorTreatmentPlanResponse struct {
	TreatmentPlan *responses.TreatmentPlan `json:"treatment_plan"`
}

func (d *doctorTreatmentPlanHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &TreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	switch r.Method {
	case httputil.Get:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified")
		}

		treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID, doctorID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			ctxt.Role,
			doctorID,
			treatmentPlan.PatientID,
			treatmentPlan.PatientCaseID.Int64(),
			d.dataAPI); err != nil {
			return false, err
		}

		// if we are dealing with a draft, and the owner of the treatment plan does not match the doctor requesting it,
		// return an error because this should never be the case
		if treatmentPlan.InDraftMode() && treatmentPlan.DoctorID.Int64() != doctorID {
			return false, apiservice.NewAccessForbiddenError()
		}

	case httputil.Put, httputil.Delete:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified")
		}

		treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID, doctorID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			ctxt.Role,
			doctorID,
			treatmentPlan.PatientID,
			treatmentPlan.PatientCaseID.Int64(),
			d.dataAPI); err != nil {
			return false, err
		}

		// ensure that doctor is owner of the treatment plan
		// and that the treatment plan is in draft mode
		if doctorID != treatmentPlan.DoctorID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

	case httputil.Post:
		if requestData.TPParent == nil || requestData.TPParent.ParentID.Int64() == 0 {
			return false, apiservice.NewValidationError("parent_id must be specified")
		}

		patientVisitID := requestData.TPParent.ParentID.Int64()
		switch requestData.TPParent.ParentType {
		case common.TPParentTypeTreatmentPlan:
			// ensure that parent treatment plan is ACTIVE
			parentTreatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TPParent.ParentID.Int64(), doctorID)
			if err != nil {
				return false, err
			} else if parentTreatmentPlan.Status != api.StatusActive {
				return false, apiservice.NewValidationError("parent treatment plan has to be ACTIVE")
			}

			patientVisitID, err = d.dataAPI.GetPatientVisitIDFromTreatmentPlanID(requestData.TPParent.ParentID.Int64())
			if err != nil {
				return false, err
			}
		case common.TPParentTypePatientVisit:
		default:
			return false, apiservice.NewValidationError("Expected the parent type to either by PATIENT_VISIT or TREATMENT_PLAN")
		}
		ctxt.RequestCache[apiservice.PatientVisitID] = patientVisitID

		patientCase, err := d.dataAPI.GetPatientCaseFromPatientVisitID(patientVisitID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientCase] = patientCase

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			ctxt.Role,
			doctorID,
			patientCase.PatientID.Int64(),
			patientCase.ID.Int64(),
			d.dataAPI); err != nil {
			return false, err
		}

	default:
		return false, nil
	}

	return true, nil
}

func (d *doctorTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		d.getTreatmentPlan(w, r)
	case httputil.Post:
		d.pickATreatmentPlan(w, r)
	case httputil.Put:
		d.submitTreatmentPlan(w, r)
	case httputil.Delete:
		d.deleteTreatmentPlan(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *doctorTreatmentPlanHandler) deleteTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	// Ensure treatment plan is a draft
	if !treatmentPlan.InDraftMode() {
		apiservice.WriteValidationError("only draft treatment plan can be deleted", w, r)
		return
	}

	// Delete treatment plan
	if err := d.dataAPI.DeleteTreatmentPlan(treatmentPlan.ID.Int64()); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *doctorTreatmentPlanHandler) submitTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	// First check request to support older apps
	// FIXME: remove this when no longer needed
	note := requestData.Message
	if note == "" {
		var err error
		note, err = d.dataAPI.GetTreatmentPlanNote(requestData.TreatmentPlanID)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(err, w, r)
			return
		}
		if note == "" {
			apiservice.WriteValidationError("Please include a personal note to the patient before submitting the treatment plan.", w, r)
			return
		}
	}

	var patientVisitID int64
	switch treatmentPlan.Parent.ParentType {
	case common.TPParentTypePatientVisit:
		// if the parent of this treatment plan is a patient visit, this means that this is the first
		// treatment plan. In this case we expect the patient visit to be in the REVIEWING state.
		patientVisitID = treatmentPlan.Parent.ParentID.Int64()
		if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataAPI, patientVisitID, common.PVStatusReviewing); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case common.TPParentTypeTreatmentPlan:
		var err error
		patientVisitID, err = d.dataAPI.GetPatientVisitIDFromTreatmentPlanID(requestData.TreatmentPlanID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// if the parent of the treatment plan is a previous version of a treatment plan, ensure that it is an ACTIVE
		// treatment plan
		treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(treatmentPlan.Parent.ParentID.Int64(), treatmentPlan.DoctorID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if treatmentPlan.Status != api.StatusActive {
			apiservice.WriteValidationError(fmt.Sprintf("Expected the parent treatment plan to be in the active state but its in %s state", treatmentPlan.Status), w, r)
			return
		}

	default:
		apiservice.WriteValidationError(fmt.Sprintf("Parent of treatment plan is unexpected parent of type %s", treatmentPlan.Parent.ParentType), w, r)
		return
	}

	// mark the treatment plan as submitted
	status := common.TPStatusSubmitted
	if err := d.dataAPI.UpdateTreatmentPlan(treatmentPlan.ID.Int64(), &api.TreatmentPlanUpdate{
		Status: &status,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	d.dispatcher.Publish(&TreatmentPlanSubmittedEvent{
		VisitID:       patientVisitID,
		TreatmentPlan: treatmentPlan,
	})

	if d.routeErx {
		apiservice.QueueUpJob(d.erxRoutingQueue, &erxRouteMessage{
			TreatmentPlanID: requestData.TreatmentPlanID,
			PatientID:       treatmentPlan.PatientID,
			DoctorID:        treatmentPlan.DoctorID.Int64(),
			Message:         note,
		})
	} else {
		if err := d.dataAPI.ActivateTreatmentPlan(treatmentPlan.ID.Int64(), treatmentPlan.DoctorID.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		doctor, err := d.dataAPI.GetDoctorFromID(treatmentPlan.DoctorID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := sendCaseMessageAndPublishTPActivatedEvent(d.dataAPI, d.dispatcher, treatmentPlan, doctor, note); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *doctorTreatmentPlanHandler) getTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	// only return the small amount of information retreived about the treatment plan
	if requestData.Abridged {
		tpRes, err := responses.TransformTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, treatmentPlan)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: tpRes})
		return
	}

	if err := populateTreatmentPlan(treatmentPlan, doctorID, d.dataAPI, parseSections(requestData.Sections)); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tpRes, err := responses.TransformTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, treatmentPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK,
		&DoctorTreatmentPlanResponse{TreatmentPlan: tpRes})
}

func (d *doctorTreatmentPlanHandler) pickATreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	patientVisitID := ctxt.RequestCache[apiservice.PatientVisitID].(int64)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)

	if requestData.TPContentSource != nil {
		switch requestData.TPContentSource.Type {
		case common.TPContentSourceTypeFTP, common.TPContentSourceTypeTreatmentPlan:
		default:
			apiservice.WriteValidationError("Invalid content source for treatment plan", w, r)
			return
		}
	}

	tp := &common.TreatmentPlan{
		PatientID:     patientCase.PatientID.Int64(),
		PatientCaseID: patientCase.ID,
		DoctorID:      encoding.NewObjectID(doctorID),
		Parent:        requestData.TPParent,
		ContentSource: requestData.TPContentSource,
	}

	if err := copyContentSourceIntoTreatmentPlan(tp, d.dataAPI, doctorID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlanID, err := d.dataAPI.StartNewTreatmentPlan(patientVisitID, tp)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start new treatment plan for patient visit: "+err.Error())
		return
	}

	// get the treatment plan just created so that it populates it with all the necessary metadata
	tp, err = d.dataAPI.GetAbridgedTreatmentPlan(treatmentPlanID, doctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := populateTreatmentPlan(tp, doctorID, d.dataAPI, AllSections); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	d.dispatcher.Publish(&NewTreatmentPlanStartedEvent{
		PatientID:       tp.PatientID,
		DoctorID:        doctorID,
		Case:            patientCase,
		CaseID:          tp.PatientCaseID.Int64(),
		VisitID:         patientVisitID,
		TreatmentPlanID: treatmentPlanID,
	})

	tpRes, err := responses.TransformTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, tp)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK,
		&DoctorTreatmentPlanResponse{TreatmentPlan: tpRes})
}
