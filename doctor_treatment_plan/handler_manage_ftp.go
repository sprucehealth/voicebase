package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type manageFTPHandler struct {
	dataAPI api.DataAPI
}

type manageFTPRequestData struct {
	DoctorID                int64                           `json:"doctor_id,string"`
	FavoriteTreatmentPlans  []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
	FavoriteTreatmentPlanID int64                           `json:"favorite_treatment_plan_id,string"`
}

func NewManageFTPHandler(dataAPI api.DataAPI) http.Handler {
	return &manageFTPHandler{
		dataAPI: dataAPI,
	}
}

func (i *manageFTPHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (i *manageFTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		i.getFTPsForDoctor(w, r)
	case apiservice.HTTP_POST:
		i.createOrUpdateFTP(w, r)
	case apiservice.HTTP_DELETE:
		i.deleteFTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (i *manageFTPHandler) getFTPsForDoctor(w http.ResponseWriter, r *http.Request) {
	requestData := &manageFTPRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.DoctorID == 0 {
		apiservice.WriteValidationError("doctor_id is required", w, r)
		return
	}

	favoriteTreatmentPlans, err := i.dataAPI.GetFavoriteTreatmentPlansForDoctor(requestData.DoctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}

func (i *manageFTPHandler) createOrUpdateFTP(w http.ResponseWriter, r *http.Request) {
	requestData := &manageFTPRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.DoctorID == 0 {
		apiservice.WriteValidationError("doctor_id must be specified", w, r)
		return
	} else if len(requestData.FavoriteTreatmentPlans) == 0 {
		apiservice.WriteValidationError("FTPs required", w, r)
		return
	}

	// validate all ftps being added
	for _, ftpItem := range requestData.FavoriteTreatmentPlans {
		if err := ftpItem.Validate(); err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		ftpItem.DoctorId = requestData.DoctorID
	}

	// add all ftps to the doctor account
	for _, ftpItem := range requestData.FavoriteTreatmentPlans {
		if err := i.dataAPI.CreateOrUpdateFavoriteTreatmentPlan(ftpItem, 0); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	favoriteTreatmentPlans, err := i.dataAPI.GetFavoriteTreatmentPlansForDoctor(requestData.DoctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}

func (i *manageFTPHandler) deleteFTP(w http.ResponseWriter, r *http.Request) {
	requestData := &manageFTPRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.DoctorID == 0 {
		apiservice.WriteValidationError("doctor_id required", w, r)
		return
	} else if requestData.FavoriteTreatmentPlanID == 0 {
		apiservice.WriteValidationError("favorite_treatment_plan_id is required", w, r)
		return
	}

	if err := i.dataAPI.DeleteFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanID, requestData.DoctorID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	favoriteTreatmentPlans, err := i.dataAPI.GetFavoriteTreatmentPlansForDoctor(requestData.DoctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}
