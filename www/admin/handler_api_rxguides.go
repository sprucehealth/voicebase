package admin

import (
	"bytes"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
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
	ndc := mux.Vars(r)["ndc"]

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetRXGuide", map[string]interface{}{"ndc": ndc})

	details, err := h.dataAPI.DrugDetails(ndc)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var html string

	if r.FormValue("with_html") != "" {
		treatment := &common.Treatment{
			DrugName:            details.Name,
			PatientInstructions: "Apply a pea-sized amount to the area affected by acne in the morning and at night.",
			Doctor: &common.Doctor{
				ShortTitle: "Dr. Kohen",
			},
		}

		b := &bytes.Buffer{}
		if err := treatment_plan.RenderRXGuide(b, details, treatment, nil); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		html = b.String()
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Guide *common.DrugDetails `json:"guide"`
		HTML  string              `json:"html"`
	}{
		Guide: details,
		HTML:  html,
	})
}
