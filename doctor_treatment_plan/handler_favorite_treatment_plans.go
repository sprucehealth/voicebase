package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type doctorFavoriteTreatmentPlansHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorFavoriteTreatmentPlansHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&doctorFavoriteTreatmentPlansHandler{
			dataAPI: dataAPI,
		}), []string{"GET", "POST", "DELETE", "PUT"})
}

type DoctorFavoriteTreatmentPlansRequestData struct {
	FavoriteTreatmentPlanID int64                         `json:"favorite_treatment_plan_id" schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *common.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
	TreatmentPlanID         int64                         `json:"treatment_plan_id,omitempty,string"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlans []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan  *common.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *doctorFavoriteTreatmentPlansHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctor, err := d.dataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	requestData := &DoctorFavoriteTreatmentPlansRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	if requestData.FavoriteTreatmentPlanID > 0 {
		// ensure that the doctor is the owner of the favorite treatment plan
		favoriteTreatmentPlan, err := d.dataAPI.GetFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.FavoriteTreatmentPlan] = favoriteTreatmentPlan

		if favoriteTreatmentPlan.DoctorID != doctor.DoctorID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}
	}

	if requestData.TreatmentPlanID > 0 {
		// ensure that the doctor has access to the patient file
		treatmentPlan, err := d.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID, doctor.DoctorID.Int64())
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		// ensure that the doctor owns the treatment plan
		if treatmentPlan.DoctorID.Int64() != doctor.DoctorID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctor.DoctorID.Int64(), treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), d.dataAPI); err != nil {
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
	}
}

func (d *doctorFavoriteTreatmentPlansHandler) getFavoriteTreatmentPlans(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, requestData *DoctorFavoriteTreatmentPlansRequestData) {
	// no favorite treatment plan id specified in which case return all
	if requestData.FavoriteTreatmentPlanID == 0 {
		favoriteTreatmentPlans, err := d.dataAPI.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
		return
	}

	ftp := apiservice.GetContext(r).RequestCache[apiservice.FavoriteTreatmentPlan].(*common.FavoriteTreatmentPlan)
	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: ftp})
}

func (d *doctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, req *DoctorFavoriteTreatmentPlansRequestData) {
	// ensure that favorite treatment plan has a name
	if err := req.FavoriteTreatmentPlan.Validate(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// this means that the favorite treatment plan was created
	// in the context of a treatment plan so associate the two
	if req.TreatmentPlanID != 0 {
		drTreatmentPlan := apiservice.GetContext(r).RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

		if err := fillInTreatmentPlan(drTreatmentPlan, doctor.DoctorID.Int64(), d.dataAPI, AllSections); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if !req.FavoriteTreatmentPlan.EqualsTreatmentPlan(drTreatmentPlan) {
			apiservice.WriteValidationError("Cannot associate a favorite treatment plan with a treatment plan when the contents of the two don't match", w, r)
			return
		}
	}

	// prepare the favorite treatment plan to have a doctor id
	req.FavoriteTreatmentPlan.DoctorID = doctor.DoctorID.Int64()

	if err := d.dataAPI.CreateOrUpdateFavoriteTreatmentPlan(req.FavoriteTreatmentPlan, req.TreatmentPlanID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataAPI.SetFavoriteTreatmentPlanScheduledMessages(req.FavoriteTreatmentPlan.ID.Int64(), req.FavoriteTreatmentPlan.ScheduledMessages); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: req.FavoriteTreatmentPlan})
}

func (d *doctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(w http.ResponseWriter, r *http.Request, doctor *common.Doctor, req *DoctorFavoriteTreatmentPlansRequestData) {
	if req.FavoriteTreatmentPlanID == 0 {
		apiservice.WriteValidationError("favorite_treatment_plan_id must be specifeid", w, r)
		return
	}

	if err := d.dataAPI.DeleteFavoriteTreatmentPlanScheduledMessages(req.FavoriteTreatmentPlanID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := d.dataAPI.DeleteFavoriteTreatmentPlan(req.FavoriteTreatmentPlanID, doctor.DoctorID.Int64()); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// echo back updated list of favorite treatment plans
	favoriteTreatmentPlans, err := d.dataAPI.GetFavoriteTreatmentPlansForDoctor(doctor.DoctorID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: favoriteTreatmentPlans})
}
