package admin

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/www"
)

type rxGuidesAPIHandler struct {
	dataAPI api.DataAPI
}

func NewRXGuideAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&rxGuidesAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *rxGuidesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetRXGuide", map[string]interface{}{"id": id})

	details, err := h.dataAPI.DrugDetails(id)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var html string

	if r.FormValue("with_html") != "" {
		// Fill in some sample content
		treatment := &common.Treatment{
			DrugName:            details.Name,
			PatientInstructions: "The doctors instructions will go here. This text is just to show what the RX guide will visually look like.",
			Doctor: &common.Doctor{
				ShortDisplayName: "Dr. Kohen",
			},
		}

		b := &bytes.Buffer{}
		if err := treatment_plan.RenderRXGuide(b, details, treatment, nil); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		html = b.String()
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Guide *common.DrugDetails `json:"guide"`
		HTML  string              `json:"html"`
	}{
		Guide: details,
		HTML:  html,
	})
}
