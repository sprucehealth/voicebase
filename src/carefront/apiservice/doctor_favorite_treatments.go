package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"net/http"
)

type DoctorFavoriteTreatmentsHandler struct {
	DataApi api.DataAPI
}

type DoctorFavoriteTreatmentsRequest struct {
	TreatmentPlanId    int64                             `json:"treamtent_plan_id,string"`
	PatientVisitId     int64                             `json:"patient_visit_id,string"`
	FavoriteTreatments []*common.DoctorFavoriteTreatment `json:"favorite_treatments"`
}

type DoctorFavoriteTreatmentsResponse struct {
	FavoritedTreatments []*common.DoctorFavoriteTreatment `json:"favorite_treatments"`
	Treatments          []*common.Treatment               `json:"treatments,omitempty"`
}

func (t *DoctorFavoriteTreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t.getFavoriteTreatments(w, r)
	case "POST":
		t.addFavoriteTreatments(w, r)
	case "DELETE":
		t.deleteFavoriteTreatments(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (t *DoctorFavoriteTreatmentsHandler) getFavoriteTreatments(w http.ResponseWriter, r *http.Request) {
	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	favoriteTreatments, err := t.DataApi.GetFavoriteTreatments(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentsResponse{FavoritedTreatments: favoriteTreatments})
}

func (t *DoctorFavoriteTreatmentsHandler) deleteFavoriteTreatments(w http.ResponseWriter, r *http.Request) {
	var favoriteTreatmentRequest DoctorFavoriteTreatmentsRequest

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&favoriteTreatmentRequest); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	for _, favoriteTreatment := range favoriteTreatmentRequest.FavoriteTreatments {
		if favoriteTreatment.Id == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to delete a treatment that does not have an id associated with it")
			return
		}
	}

	err = t.DataApi.DeleteFavoriteTreatments(favoriteTreatmentRequest.FavoriteTreatments, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorited treatment: "+err.Error())
		return
	}

	favoriteTreatments, err := t.DataApi.GetFavoriteTreatments(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	treatmentPlanId := favoriteTreatmentRequest.TreatmentPlanId
	patientVisitId := favoriteTreatmentRequest.PatientVisitId
	var treatmentsInTreatmentPlan []*common.Treatment
	if patientVisitId != 0 {
		if treatmentPlanId == 0 {
			treatmentPlanId, err = t.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan from patient visit: "+err.Error())
				return
			}
		}

		treatmentsInTreatmentPlan, err = t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, treatmentPlanId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments based on treatment plan id: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentsResponse{
		FavoritedTreatments: favoriteTreatments,
		Treatments:          treatmentsInTreatmentPlan,
	})
}

func (t *DoctorFavoriteTreatmentsHandler) addFavoriteTreatments(w http.ResponseWriter, r *http.Request) {
	var favoriteTreatmentRequest DoctorFavoriteTreatmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&favoriteTreatmentRequest); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	for _, favoriteTreatment := range favoriteTreatmentRequest.FavoriteTreatments {
		err = validateTreatment(favoriteTreatment.FavoritedTreatment)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(favoriteTreatment.FavoritedTreatment.DrugInternalName)
		favoriteTreatment.FavoritedTreatment.DrugName = drugName
		// only break down name into route and form if the route and form are non-empty strings
		if drugForm != "" && drugRoute != "" {
			favoriteTreatment.FavoritedTreatment.DrugForm = drugForm
			favoriteTreatment.FavoritedTreatment.DrugRoute = drugRoute
		}
	}

	err = t.DataApi.AddFavoriteTreatments(favoriteTreatmentRequest.FavoriteTreatments, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to favorite treatment: "+err.Error())
		return
	}

	favoriteTreatments, err := t.DataApi.GetFavoriteTreatments(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorited treatments for doctor: "+err.Error())
		return
	}

	treatmentPlanId := favoriteTreatmentRequest.TreatmentPlanId
	patientVisitId := favoriteTreatmentRequest.PatientVisitId
	var treatmentsInTreatmentPlan []*common.Treatment
	if patientVisitId != 0 {
		if treatmentPlanId == 0 {
			treatmentPlanId, err = t.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan from patient visit: "+err.Error())
				return
			}
		}

		treatmentsInTreatmentPlan, err = t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, treatmentPlanId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments based on treatment plan id: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentsResponse{
		FavoritedTreatments: favoriteTreatments,
		Treatments:          treatmentsInTreatmentPlan,
	})
}
