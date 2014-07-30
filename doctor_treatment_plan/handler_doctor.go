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
	dataApi        api.DataAPI
	erxAPI         erx.ERxAPI
	erxStatusQueue *common.SQSQueue
	routeErx       bool
}

func NewDoctorTreatmentPlanHandler(dataApi api.DataAPI, erxAPI erx.ERxAPI, erxStatusQueue *common.SQSQueue, routeErx bool) *doctorTreatmentPlanHandler {
	return &doctorTreatmentPlanHandler{
		dataApi:        dataApi,
		erxAPI:         erxAPI,
		erxStatusQueue: erxStatusQueue,
		routeErx:       routeErx,
	}
}

type TreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId int64                              `json:"dr_favorite_treatment_plan_id,string" schema:"dr_favorite_treatment_plan_id"`
	TreatmentPlanId               int64                              `json:"treatment_plan_id,string" schema:"treatment_plan_id" `
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

	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &TreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorId] = doctorId

	switch r.Method {
	case apiservice.HTTP_GET:
		if requestData.TreatmentPlanId == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
		}

		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctorId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateReadAccessToPatientCase(doctorId, treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}

		// if we are dealing with a draft, and the owner of the treatment plan does not match the doctor requesting it,
		// return an error because this should never be the case
		if treatmentPlan.Status == api.STATUS_DRAFT && treatmentPlan.DoctorId.Int64() != doctorId {
			return false, apiservice.NewAccessForbiddenError()
		}

	case apiservice.HTTP_PUT, apiservice.HTTP_DELETE:
		if requestData.TreatmentPlanId == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
		}

		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctorId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err := apiservice.ValidateWriteAccessToPatientCase(doctorId, treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), d.dataApi); err != nil {
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
		ctxt.RequestCache[apiservice.PatientVisitId] = patientVisitId

		patientCase, err := d.dataApi.GetPatientCaseFromPatientVisitId(patientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientCase] = patientCase

		if err := apiservice.ValidateWriteAccessToPatientCase(doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), d.dataApi); err != nil {
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
	if treatmentPlan.Status != api.STATUS_DRAFT {
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
		apiservice.WriteValidationError("message must not be empty", w, r)
		return
	}

	doctor, err := d.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var patientVisitId int64
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
		patientVisitId, err = d.dataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TreatmentPlanId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// if the parent of the treatment plan is a previous version of a treatment plan, ensure that it is an ACTIVE
		// treatment plan
		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(treatmentPlan.Parent.ParentId.Int64(), doctor.DoctorId.Int64())
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

	// get patient from treatment plan id
	patient, err := d.dataApi.GetPatientFromId(treatmentPlan.PatientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// route treatments to patient pharmacy if any exist
	if err := routeRxInTreatmentPlanToPharmacy(requestData.TreatmentPlanId, patient, doctor, d.routeErx, d.dataApi, d.erxAPI, d.erxStatusQueue); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataApi.ActivateTreatmentPlan(requestData.TreatmentPlanId, doctor.DoctorId.Int64()); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	caseID, err := d.dataApi.GetPatientCaseIdFromPatientVisitId(patientVisitId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	msg := &common.CaseMessage{
		CaseID:   caseID,
		PersonID: doctor.PersonId,
		Body:     requestData.Message,
		Attachments: []*common.CaseMessageAttachment{
			&common.CaseMessageAttachment{
				ItemType: common.AttachmentTypeTreatmentPlan,
				ItemID:   treatmentPlan.Id.Int64(),
			},
		},
	}
	if _, err := d.dataApi.CreateCaseMessage(msg); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Publish event that treamtent plan was created
	dispatch.Default.Publish(&TreatmentPlanActivatedEvent{
		PatientId:     treatmentPlan.PatientId,
		DoctorId:      doctor.DoctorId.Int64(),
		VisitId:       patientVisitId,
		TreatmentPlan: treatmentPlan,
		Patient:       patient,
		Message:       msg,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (d *doctorTreatmentPlanHandler) getTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*TreatmentPlanRequestData)
	doctorId := ctxt.RequestCache[apiservice.DoctorId].(int64)
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
	doctorId := ctxt.RequestCache[apiservice.DoctorId].(int64)
	patientVisitId := ctxt.RequestCache[apiservice.PatientVisitId].(int64)
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

	dispatch.Default.Publish(&NewTreatmentPlanStartedEvent{
		DoctorId:        doctorId,
		PatientVisitId:  patientVisitId,
		TreatmentPlanId: treatmentPlanId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}
