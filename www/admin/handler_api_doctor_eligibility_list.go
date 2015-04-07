package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/encoding"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type doctorEligibilityListAPIHandler struct {
	dataAPI api.DataAPI
}

type statePathwayMapping struct {
	StateCode   string `json:"state_code"`
	PathwayTag  string `json:"pathway_tag"`
	Notify      bool   `json:"notify"`
	Unavailable bool   `json:"unavailable"`
}

type statePathwayMappingUpdate struct {
	ID          int64 `json:"id,string"`
	Notify      *bool `json:"notify"`
	Unavailable *bool `json:"unavailable"`
}

type doctorEligibilityUpdateRequest struct {
	Delete []encoding.ObjectID          `json:"delete,omitempty"`
	Create []*statePathwayMapping       `json:"create,omitempty"`
	Update []*statePathwayMappingUpdate `json:"update,omitempty"`
}

func NewDoctorEligibilityListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&doctorEligibilityListAPIHandler{
		dataAPI: dataAPI,
	}, []string{httputil.Get, httputil.Patch})
}

func (h *doctorEligibilityListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)

	switch r.Method {
	case httputil.Get:
		audit.LogAction(account.ID, "AdminAPI", "ListDoctorEligibility", map[string]interface{}{"doctor_id": doctorID})
		h.get(w, r, doctorID)
	case httputil.Patch:
		audit.LogAction(account.ID, "AdminAPI", "UpdateDoctorEligibility", map[string]interface{}{"doctor_id": doctorID})
		h.patch(w, r, doctorID)
	}
}

func (h *doctorEligibilityListAPIHandler) get(w http.ResponseWriter, r *http.Request, doctorID int64) {
	mappings, err := h.dataAPI.CareProviderStatePathwayMappings(&api.CareProviderStatePathwayMappingQuery{
		Provider: api.Provider{
			Role: api.RoleDoctor,
			ID:   doctorID,
		},
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &cpMappingsResponse{
		Mappings: mappings,
	})
}

func (h *doctorEligibilityListAPIHandler) patch(w http.ResponseWriter, r *http.Request, doctorID int64) {
	var req doctorEligibilityUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "unable to parse body")
		return
	}

	patch := &api.CareProviderStatePathwayMappingPatch{}
	for _, id := range req.Delete {
		patch.Delete = append(patch.Delete, id.Int64())
	}
	for _, c := range req.Create {
		patch.Create = append(patch.Create, &api.CareProviderStatePathway{
			Provider: api.Provider{
				Role: api.RoleDoctor,
				ID:   doctorID,
			},
			StateCode:   c.StateCode,
			PathwayTag:  c.PathwayTag,
			Notify:      c.Notify,
			Unavailable: c.Unavailable,
		})
	}
	for _, u := range req.Update {
		patch.Update = append(patch.Update, &api.CareProviderStatePathwayMappingUpdate{
			ID:          u.ID,
			Notify:      u.Notify,
			Unavailable: u.Unavailable,
		})
	}

	if err := h.dataAPI.UpdateCareProviderStatePathwayMapping(patch); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	h.get(w, r, doctorID)
}
