package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
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
	FavoriteTreatmentPlanId int64                         `schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *common.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
	TreatmentPlanId         int64                         `json:"treatment_plan_id,string"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlans []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan  *common.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *doctorFavoriteTreatmentPlansHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	doctor, err := d.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	requestData := &DoctorFavoriteTreatmentPlansRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	if requestData.FavoriteTreatmentPlanId > 0 {
		// ensure that the doctor is the owner of the favorite treatment plan
		favoriteTreatmentPlan, err := d.dataApi.GetFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.FavoriteTreatmentPlan] = favoriteTreatmentPlan

		if favoriteTreatmentPlan.DoctorId != doctor.DoctorId.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}
	}

	if requestData.TreatmentPlanId > 0 {
		// ensure that the doctor has access to the patient file
		treatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctor.DoctorId.Int64())
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		// ensure that the doctor owns the treatment plan
		if treatmentPlan.DoctorId.Int64() != doctor.DoctorId.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, doctor.DoctorId.Int64(), treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), d.dataApi); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *doctorFavoriteTreatmentPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*DoctorFavoriteTreatmentPlansRequestData)

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
	if requestData.FavoriteTreatmentPlanId == 0 {
		favoriteTreatmentPlans, err := d.dataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
		return
	}

	favoriteTreatmentPlan := apiservice.GetContext(r).RequestCache[apiservice.FavoriteTreatmentPlan].(*common.FavoriteTreatmentPlan)

	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: favoriteTreatmentPlan})
}

func (d *doctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {

	// ensure that favorite treatment plan has a name
	if requestData.FavoriteTreatmentPlan.Name == "" {
		apiservice.WriteValidationError("A favorite treatment plan requires a name", w, r)
		return
	}

	// ensure that favorite treatment plan has atleast one of the sections filled out
	if (requestData.FavoriteTreatmentPlan.TreatmentList == nil ||
		len(requestData.FavoriteTreatmentPlan.TreatmentList.Treatments) == 0) &&
		len(requestData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) == 0 &&
		len(requestData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) == 0 {
		apiservice.WriteValidationError("A favorite treatment plan must have either a set of treatments, a regimen plan or list of advice to be added", w, r)
		return
	}

	// this means that the favorite treatment plan was created
	// in the context of a treatment plan so associate the two
	if requestData.TreatmentPlanId != 0 {
		drTreatmentPlan := apiservice.GetContext(r).RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)

		if err := fillInTreatmentPlan(drTreatmentPlan, doctor.DoctorId.Int64(), d.dataApi); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if !requestData.FavoriteTreatmentPlan.EqualsDoctorTreatmentPlan(drTreatmentPlan) {
			apiservice.WriteValidationError("Cannot associate a favorite treatment plan with a treatment plan when the contents of the two don't match", w, r)
			return
		}
	}

	// prepare the favorite treatment plan to have a doctor id
	requestData.FavoriteTreatmentPlan.DoctorId = doctor.DoctorId.Int64()

	if err := d.dataApi.CreateOrUpdateFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlan, requestData.TreatmentPlanId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: requestData.FavoriteTreatmentPlan})
}

func (d *doctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {

	if requestData.FavoriteTreatmentPlanId == 0 {
		apiservice.WriteValidationError("favorite_treatment_plan_id must be specifeid", w, r)
		return
	}

	if err := d.dataApi.DeleteFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// echo back updated list of favorite treatment plans
	favoriteTreatmentPlans, err := d.dataApi.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
}
