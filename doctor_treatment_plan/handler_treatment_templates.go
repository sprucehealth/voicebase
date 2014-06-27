package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

type treatmentTemplatesHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentTemplatesHandler(dataApi api.DataAPI) *treatmentTemplatesHandler {
	return &treatmentTemplatesHandler{
		dataAPI: dataApi,
	}
}

type DoctorTreatmentTemplatesRequest struct {
	TreatmentPlanId    encoding.ObjectId                 `json:"treatment_plan_id"`
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
}

type DoctorTreatmentTemplatesResponse struct {
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
	Treatments         []*common.Treatment               `json:"treatments,omitempty"`
}

func (t *treatmentTemplatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		t.getTreatmentTemplates(w, r)
	case apiservice.HTTP_POST:
		t.addTreatmentTemplates(w, r)
	case apiservice.HTTP_DELETE:
		t.deleteTreatmentTemplates(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (t *treatmentTemplatesHandler) getTreatmentTemplates(w http.ResponseWriter, r *http.Request) {
	doctorId, err := t.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	doctorTreatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{TreatmentTemplates: doctorTreatmentTemplates})
}

func (t *treatmentTemplatesHandler) deleteTreatmentTemplates(w http.ResponseWriter, r *http.Request) {

	var treatmentTemplateRequest DoctorTreatmentTemplatesRequest
	if err := apiservice.DecodeRequestData(&treatmentTemplateRequest, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if treatmentTemplateRequest.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	doctorId, err := t.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	for _, favoriteTreatment := range treatmentTemplateRequest.TreatmentTemplates {
		if favoriteTreatment.Id.Int64() == 0 {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to delete a treatment that does not have an id associated with it")
			return
		}
	}

	err = t.dataAPI.DeleteTreatmentTemplates(treatmentTemplateRequest.TreatmentTemplates, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorited treatment: "+err.Error())
		return
	}

	treatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	treatmentsInTreatmentPlan, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentTemplateRequest.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments based on treatment plan id: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}

func (t *treatmentTemplatesHandler) addTreatmentTemplates(w http.ResponseWriter, r *http.Request) {
	var treatmentTemplateRequest DoctorTreatmentTemplatesRequest
	if err := apiservice.DecodeRequestData(&treatmentTemplateRequest, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if treatmentTemplateRequest.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	doctorId, err := t.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	for _, treatmentTemplate := range treatmentTemplateRequest.TreatmentTemplates {
		err = apiservice.ValidateTreatment(treatmentTemplate.Treatment)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		treatmentTemplate.Treatment.DrugName, treatmentTemplate.Treatment.DrugForm, treatmentTemplate.Treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(treatmentTemplate.Treatment.DrugInternalName)
	}

	err = t.dataAPI.AddTreatmentTemplates(treatmentTemplateRequest.TreatmentTemplates, doctorId, treatmentTemplateRequest.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to favorite treatment: "+err.Error())
		return
	}

	treatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorited treatments for doctor: "+err.Error())
		return
	}

	treatmentsInTreatmentPlan, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentTemplateRequest.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}
