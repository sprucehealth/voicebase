package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"net/http"
)

type DoctorTreatmentTemplatesHandler struct {
	DataApi api.DataAPI
}

type DoctorTreatmentTemplatesRequest struct {
	TreatmentPlanId    encoding.ObjectId                 `json:"treatment_plan_id"`
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
}

type DoctorTreatmentTemplatesResponse struct {
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
	Treatments         []*common.Treatment               `json:"treatments,omitempty"`
}

func (t *DoctorTreatmentTemplatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		t.getTreatmentTemplates(w, r)
	case HTTP_POST:
		t.addTreatmentTemplates(w, r)
	case HTTP_DELETE:
		t.deleteTreatmentTemplates(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (t *DoctorTreatmentTemplatesHandler) getTreatmentTemplates(w http.ResponseWriter, r *http.Request) {
	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	doctorTreatmentTemplates, err := t.DataApi.GetTreatmentTemplates(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{TreatmentTemplates: doctorTreatmentTemplates})
}

func (t *DoctorTreatmentTemplatesHandler) deleteTreatmentTemplates(w http.ResponseWriter, r *http.Request) {

	var treatmentTemplateRequest DoctorTreatmentTemplatesRequest
	if err := DecodeRequestData(&treatmentTemplateRequest, r); err != nil {
		WriteError(err, w, r)
		return
	} else if treatmentTemplateRequest.TreatmentPlanId.Int64() == 0 {
		WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	for _, favoriteTreatment := range treatmentTemplateRequest.TreatmentTemplates {
		if favoriteTreatment.Id.Int64() == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to delete a treatment that does not have an id associated with it")
			return
		}
	}

	err = t.DataApi.DeleteTreatmentTemplates(treatmentTemplateRequest.TreatmentTemplates, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorited treatment: "+err.Error())
		return
	}

	treatmentTemplates, err := t.DataApi.GetTreatmentTemplates(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	treatmentsInTreatmentPlan, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentTemplateRequest.TreatmentPlanId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments based on treatment plan id: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}

func (t *DoctorTreatmentTemplatesHandler) addTreatmentTemplates(w http.ResponseWriter, r *http.Request) {
	var treatmentTemplateRequest DoctorTreatmentTemplatesRequest
	if err := DecodeRequestData(&treatmentTemplateRequest, r); err != nil {
		WriteError(err, w, r)
		return
	} else if treatmentTemplateRequest.TreatmentPlanId.Int64() == 0 {
		WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	for _, treatmentTemplate := range treatmentTemplateRequest.TreatmentTemplates {
		err = validateTreatment(treatmentTemplate.Treatment)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		drugName, drugForm, drugRoute := BreakDrugInternalNameIntoComponents(treatmentTemplate.Treatment.DrugInternalName)
		treatmentTemplate.Treatment.DrugName = drugName
		// only break down name into route and form if the route and form are non-empty strings
		if drugForm != "" && drugRoute != "" {
			treatmentTemplate.Treatment.DrugForm = drugForm
			treatmentTemplate.Treatment.DrugRoute = drugRoute
		}
	}

	err = t.DataApi.AddTreatmentTemplates(treatmentTemplateRequest.TreatmentTemplates, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to favorite treatment: "+err.Error())
		return
	}

	treatmentTemplates, err := t.DataApi.GetTreatmentTemplates(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorited treatments for doctor: "+err.Error())
		return
	}

	treatmentsInTreatmentPlan, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentTemplateRequest.TreatmentPlanId.Int64())
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}
