package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"carefront/libs/erx"
	"fmt"
	"net/http"
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

type DoctorTreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId int64 `schema:"dr_favorite_treatment_plan_id" json:"dr_favorite_treatment_plan_id,string"`
	TreatmentPlanId               int64 `schema:"treatment_plan_id" json:"treatment_plan_id,string"`
	PatientVisitId                int64 `schema:"patient_visit_id" json:"patient_visit_id,string"`
	Abridged                      bool  `schema:"abridged" json:"abridged"`
}

type PickTreatmentPlanRequestData struct {
	TPContentSource *common.TreatmentPlanContentSource `json:"content_source"`
	TPParent        *common.TreatmentPlanParent        `json:"parent"`
}

type TreatmentPlanRequestData struct {
	TreatmentPlanId encoding.ObjectId `json:"treatment_plan_id"`
	Message         string            `json:"message"`
}

type DoctorTreatmentPlanResponse struct {
	TreatmentPlan *common.DoctorTreatmentPlan `json:"treatment_plan"`
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
	requestData := &TreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if requestData.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId.Int64(), doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Ensure treatment plan is owned by this doctor
	if doctorId != treatmentPlan.DoctorId.Int64() {
		apiservice.WriteValidationError("Cannot delete treatment plan not owned by doctor", w, r)
		return
	}

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
	var requestData TreatmentPlanRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if requestData.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	} else if requestData.Message == "" {
		apiservice.WriteValidationError("message must not be empty", w, r)
		return
	}

	doctor, err := d.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId.Int64(), doctor.DoctorId.Int64())
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
		if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataApi, patientVisitId, api.CASE_STATUS_REVIEWING); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case common.TPParentTypeTreatmentPlan:
		patientVisitId, err = d.dataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TreatmentPlanId.Int64())
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

	// Ensure that doctor is authorized to work on the case
	_, err = apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, apiservice.GetContext(r).AccountId, d.dataApi)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// get patient from treatment plan id
	patient, err := d.dataApi.GetPatientFromId(treatmentPlan.PatientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// route treatments to patient pharmacy if any exist
	if err := routeRxInTreatmentPlanToPharmacy(requestData.TreatmentPlanId.Int64(), patient, doctor, d.routeErx, d.dataApi, d.erxAPI, d.erxStatusQueue); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataApi.ActivateTreatmentPlan(requestData.TreatmentPlanId.Int64(), doctor.DoctorId.Int64()); err != nil {
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
	dispatch.Default.PublishAsync(&TreatmentPlanCreatedEvent{
		PatientId:       treatmentPlan.PatientId,
		DoctorId:        doctor.DoctorId.Int64(),
		VisitId:         patientVisitId,
		TreatmentPlanId: requestData.TreatmentPlanId.Int64(),
		Patient:         patient,
		Message:         msg,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (d *doctorTreatmentPlanHandler) getTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorTreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.TreatmentPlanId == 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "treatment_plan_id not specified")
		return
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	drTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctorId)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "No treatment plan exists for patient visit")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	// if we are dealing with a draft, and the owner of the treatment plan does not match the doctor requesting it,
	// return an error because this should never be the case
	if drTreatmentPlan.Status == api.STATUS_DRAFT && drTreatmentPlan.DoctorId.Int64() != doctorId {
		apiservice.WriteValidationError("Cannot retrieve draft treatment plan owned by different doctor", w, r)
		return
	}

	// only return the small amount of information retreived about the treatment plan
	if requestData.Abridged {
		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
		return
	}

	if err := fillInTreatmentPlan(drTreatmentPlan, doctorId, d.dataApi); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}

func (d *doctorTreatmentPlanHandler) pickATreatmentPlan(w http.ResponseWriter, r *http.Request) {
	requestData := &PickTreatmentPlanRequestData{}

	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.TPParent == nil || requestData.TPParent.ParentId.Int64() == 0 {
		apiservice.WriteValidationError("Expected the parent id to be specified for the treatment plan", w, r)
		return
	} else if requestData.TPParent.ParentType != common.TPParentTypePatientVisit && requestData.TPParent.ParentType != common.TPParentTypeTreatmentPlan {
		apiservice.WriteValidationError("Expected the parent type to either by PATIENT_VISIT or TREATMENT_PLAN", w, r)
		return
	} else if requestData.TPContentSource != nil {
		if requestData.TPContentSource.ContentSourceType != common.TPContentSourceTypeFTP && requestData.TPContentSource.ContentSourceType != common.TPContentSourceTypeTreatmentPlan {
			apiservice.WriteValidationError(fmt.Sprintf("Expected content source type be either FAVORITE_TREATMENT_PLAN or TREATMENT_PLAN but instead it was %s", requestData.TPContentSource.ContentSourceType), w, r)
			return
		}
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientVisitId := requestData.TPParent.ParentId.Int64()
	switch requestData.TPParent.ParentType {
	case common.TPParentTypeTreatmentPlan:
		// ensure that parent treatment plan is ACTIVE
		parentTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TPParent.ParentId.Int64(), doctorId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if parentTreatmentPlan.Status != api.STATUS_ACTIVE {
			apiservice.WriteValidationError("parent treatment plan has to be ACTIVE", w, r)
			return
		}

		patientVisitId, err = d.dataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TPParent.ParentId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	patientVisitReviewData, err := apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, apiservice.GetContext(r).AccountId, d.dataApi)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlanId, err := d.dataApi.StartNewTreatmentPlan(patientVisitReviewData.PatientVisit.PatientId.Int64(),
		patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId, requestData.TPParent, requestData.TPContentSource)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start new treatment plan for patient visit: "+err.Error())
		return
	}

	// get the treatment plan just created
	drTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(treatmentPlanId, patientVisitReviewData.DoctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := fillInTreatmentPlan(drTreatmentPlan, patientVisitReviewData.DoctorId, d.dataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	dispatch.Default.Publish(&NewTreatmentPlanStartedEvent{
		DoctorId:        patientVisitReviewData.DoctorId,
		PatientVisitId:  patientVisitId,
		TreatmentPlanId: treatmentPlanId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}
