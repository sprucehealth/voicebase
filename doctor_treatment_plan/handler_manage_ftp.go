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
	DoctorID                int64                           `json:"doctor_id,string" schema:"doctor_id"`
	FavoriteTreatmentPlans  []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
	FavoriteTreatmentPlanID int64                           `schema:"favorite_treatment_plan_id"`
	PathwayTag              string                          `json:"pathway_id" schema:"pathway_id"`
}

func NewManageFTPHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.AuthorizationRequired(&manageFTPHandler{
		dataAPI: dataAPI,
	})
}

func (h *manageFTPHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *manageFTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		h.getFTPsForDoctor(w, r)
	case apiservice.HTTP_POST:
		h.createOrUpdateFTP(w, r)
	case apiservice.HTTP_DELETE:
		h.deleteFTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *manageFTPHandler) getFTPsForDoctor(w http.ResponseWriter, r *http.Request) {
	requestData := &manageFTPRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.DoctorID == 0 {
		apiservice.WriteValidationError("doctor_id is required", w, r)
		return
	}

	// TODO: for now default to acne
	if requestData.PathwayTag == "" {
		requestData.PathwayTag = api.AcnePathwayTag
	}

	favoriteTreatmentPlans, err := h.dataAPI.FavoriteTreatmentPlansForDoctor(requestData.DoctorID, requestData.PathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}

func (h *manageFTPHandler) createOrUpdateFTP(w http.ResponseWriter, r *http.Request) {
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

	pathwayTag := requestData.PathwayTag

	// validate all ftps being added
	for _, ftpItem := range requestData.FavoriteTreatmentPlans {
		// TODO: for now default to acne
		if ftpItem.PathwayTag == "" {
			ftpItem.PathwayTag = api.AcnePathwayTag
		}
		if pathwayTag == "" {
			pathwayTag = ftpItem.PathwayTag
		}
		if err := ftpItem.Validate(); err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		ftpItem.DoctorID = requestData.DoctorID
	}

	// add all ftps to the doctor account
	for _, ftpItem := range requestData.FavoriteTreatmentPlans {
		if err := h.dataAPI.UpsertFavoriteTreatmentPlan(ftpItem, 0); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	favoriteTreatmentPlans, err := h.dataAPI.FavoriteTreatmentPlansForDoctor(requestData.DoctorID, pathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}

func (h *manageFTPHandler) deleteFTP(w http.ResponseWriter, r *http.Request) {
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

	if err := h.dataAPI.DeleteFavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanID, requestData.DoctorID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// TODO: for now default to acne
	if requestData.PathwayTag == "" {
		requestData.PathwayTag = api.AcnePathwayTag
	}

	favoriteTreatmentPlans, err := h.dataAPI.FavoriteTreatmentPlansForDoctor(requestData.DoctorID, requestData.PathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"favorite_treatment_plans": favoriteTreatmentPlans,
	})
}
