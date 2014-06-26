package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"net/http"
	"strconv"
)

type doctorFavoriteTreatmentPlansHandler struct {
	dataApi api.DataAPI
}

func NewDoctorFavoriteTreatmentPlansHandler(dataApi api.DataAPI) *doctorFavoriteTreatmentPlansHandler {
	return &doctorFavoriteTreatmentPlansHandler{
		dataApi: dataApi,
	}
}

type DoctorFavoriteTreatmentPlansRequestData struct {
	FavoriteTreatmentPlanId string                        `schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *common.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
	TreatmentPlanId         int64                         `json:"treatment_plan_id,string"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlans []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan  *common.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *doctorFavoriteTreatmentPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctor, err := d.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from id: "+err.Error())
		return
	}

	requestData := &DoctorFavoriteTreatmentPlansRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		d.getFavoriteTreatmentPlans(w, r, doctor, requestData)
	case apiservice.HTTP_POST, apiservice.HTTP_PUT:
		d.addOrUpdateFavoriteTreatmentPlan(w, r, doctor, requestData)
	case apiservice.HTTP_DELETE:
		d.deleteFavoriteTreatmentPlan(w, r, doctor, requestData)
	default:
		http.NotFound(w, r)
		return
	}
}

func (d *doctorFavoriteTreatmentPlansHandler) getFavoriteTreatmentPlans(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {

	// no favorite treatment plan id specified in which case return all
	if requestData.FavoriteTreatmentPlanId == "" {
		favoriteTreatmentPlans, err := d.dataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plans for doctor: "+err.Error())
			return
		}
		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
		return
	}

	favoriteTreatmentPlanId, err := strconv.ParseInt(requestData.FavoriteTreatmentPlanId, 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get specified favoriteTreatmentPlanId: "+err.Error())
		return
	}

	favoriteTreatmentPlan, err := d.dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan with requested id does not exist")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan: "+err.Error())
		return
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: favoriteTreatmentPlan})
}

func (d *doctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {

	// ensure that favorite treatment plan has a name
	if requestData.FavoriteTreatmentPlan.Name == "" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "A favorite treatment plan requires a name")
		return
	}

	// ensure that favorite treatment plan has atleast one of the sections filled out
	if (requestData.FavoriteTreatmentPlan.TreatmentList == nil ||
		len(requestData.FavoriteTreatmentPlan.TreatmentList.Treatments) == 0) &&
		len(requestData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) == 0 &&
		len(requestData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) == 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "A favorite treatment plan must have either a set of treatments, a regimen plan or list of advice to be added")
		return
	}

	// this means that the favorite treatment plan was created
	// in the context of a treatment plan so associate the two
	if requestData.TreatmentPlanId != 0 {
		drTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctor.DoctorId.Int64())
		if err == api.NoRowsError {
			apiservice.WriteDeveloperError(w, http.StatusNotFound, "No treatment plan exists for patient visit")
			return
		} else if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
			return
		}

		if err := fillInTreatmentPlan(drTreatmentPlan, doctor.DoctorId.Int64(), d.dataApi); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if !requestData.FavoriteTreatmentPlan.EqualsDoctorTreatmentPlan(drTreatmentPlan) {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Cannot associate a favorite treatment plan with a treatment plan when the contents of the two don't match")
			return
		}
	}

	// prepare the favorite treatment plan to have a doctor id
	requestData.FavoriteTreatmentPlan.DoctorId = doctor.DoctorId.Int64()

	if err := d.dataApi.CreateOrUpdateFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlan, requestData.TreatmentPlanId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update favorite treatment plan : "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: requestData.FavoriteTreatmentPlan})
}

func (d *doctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {

	if requestData.FavoriteTreatmentPlanId == "" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "FavoriteTreatmentPlanId required when attempting to delete a favorite treatment plan")
		return
	}

	favoriteTreatmentPlanId, err := strconv.ParseInt(requestData.FavoriteTreatmentPlanId, 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse specified favorite treatment plan id: "+err.Error())
		return
	}

	// ensure that the favorite treatment plan exists before attempting to delete it
	if _, err := d.dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId); err != nil {
		if err == api.NoRowsError {
			apiservice.WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan attempting to be deleted not found")
			return
		}

		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the favorite treamtent plan that is attempting to be deleted: "+err.Error())
		return
	}

	if err := d.dataApi.DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorite treatment plan: "+err.Error())
		return
	}

	// echo back updated list of favorite treatment plans
	favoriteTreatmentPlans, err := d.dataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plans for doctor: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
}
