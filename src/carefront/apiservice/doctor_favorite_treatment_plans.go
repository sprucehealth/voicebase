package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorFavoriteTreatmentPlansHandler struct {
	DataApi api.DataAPI
}

type DoctorFavoriteTreatmentPlansRequestData struct {
	FavoriteTreatmentPlanId string                        `schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *common.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlans []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan  *common.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *DoctorFavoriteTreatmentPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from id: "+err.Error())
		return
	}

	switch r.Method {
	case HTTP_GET:
		d.getFavoriteTreatmentPlans(w, r, doctor)
	case HTTP_POST, HTTP_PUT:
		d.addOrUpdateFavoriteTreatmentPlan(w, r, doctor)
	case HTTP_DELETE:
		d.deleteFavoriteTreatmentPlan(w, r, doctor)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (d *DoctorFavoriteTreatmentPlansHandler) getFavoriteTreatmentPlans(w http.ResponseWriter, r *http.Request, doctor *common.Doctor) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := DoctorFavoriteTreatmentPlansRequestData{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// no favorite treatment plan id specified in which case return all
	if requestData.FavoriteTreatmentPlanId == "" {
		favoriteTreatmentPlans, err := d.DataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plans for doctor: "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
		return
	}

	favoriteTreatmentPlanId, err := strconv.ParseInt(requestData.FavoriteTreatmentPlanId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get specified favoriteTreatmentPlanId: "+err.Error())
		return
	}

	favoriteTreatmentPlan, err := d.DataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err != nil && err == api.NoRowsError {
		WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan with requested id does not exist")
		return
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: favoriteTreatmentPlan})
}

func (d *DoctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// ensure that favorite treatment plan has a name
	if requestData.FavoriteTreatmentPlan.Name == "" {
		WriteDeveloperError(w, http.StatusBadRequest, "A favorite treatment plan requires a name")
		return
	}

	// ensure that favorite treatment plan has treatments
	if len(requestData.FavoriteTreatmentPlan.Treatments) == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "A favorite treatment plan has to have treamtents")
		return
	}

	// prepare the favorite treatment plan to have a doctor id
	requestData.FavoriteTreatmentPlan.DoctorId = doctor.DoctorId.Int64()

	if err := d.DataApi.CreateOrUpdateFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlan); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update favorite treatment plan : "+err.Error())
		return
	}

	// echo back added favorite treatment plan

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: requestData.FavoriteTreatmentPlan})
}

func (d *DoctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := DoctorFavoriteTreatmentPlansRequestData{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	if requestData.FavoriteTreatmentPlanId == "" {
		WriteDeveloperError(w, http.StatusBadRequest, "FavoriteTreatmentPlanId required when attempting to delete a favorite treatment plan")
		return
	}

	favoriteTreatmentPlanId, err := strconv.ParseInt(requestData.FavoriteTreatmentPlanId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse specified favorite treatment plan id: "+err.Error())
		return
	}

	// ensure that the favorite treatment plan exists before attempting to delete it
	if _, err := d.DataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId); err != nil {
		if err == api.NoRowsError {
			WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan attempting to be deleted not found")
			return
		}

		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the favorite treamtent plan that is attempting to be deleted: "+err.Error())
		return
	}

	if err := d.DataApi.DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorite treatment plan: "+err.Error())
		return
	}

	// echo back updated list of favorite treatment plans
	favoriteTreatmentPlans, err := d.DataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plans for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
}
