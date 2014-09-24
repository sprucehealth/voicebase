package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/www"
)

const maxDrugSearchResults = 10

type drugSearchAPIHandler struct {
	dataAPI api.DataAPI
	eRxAPI  erx.ERxAPI
}

type drugSearchResult struct {
	Name      string                       `json:"name"`
	Strengths map[string]*common.Treatment `json:"strengths"`
}

func NewDrugSearchAPIHandler(dataAPI api.DataAPI, eRxAPI erx.ERxAPI) http.Handler {
	return httputil.SupportedMethods(&drugSearchAPIHandler{
		dataAPI: dataAPI,
		eRxAPI:  eRxAPI,
	}, []string{"GET"})
}

func (h *drugSearchAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var results []*drugSearchResult

	query := r.FormValue("q")

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "SearchDrugs", map[string]interface{}{"query": query})

	details, err := h.dataAPI.ListDrugDetails()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	detailsMap := make(map[string]*common.DrugDetails)
	for _, d := range details {
		detailsMap[d.NDC] = d
	}

	if query != "" {
		var err error
		names, err := h.eRxAPI.GetDrugNamesForDoctor(0, query)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if len(names) > maxDrugSearchResults {
			names = names[:maxDrugSearchResults]
		}

		for _, name := range names {
			strengths, err := h.eRxAPI.SearchForMedicationStrength(0, name)
			if err != nil {
				www.APIInternalError(w, r, err)
				return
			}
			res := &drugSearchResult{
				Name:      name,
				Strengths: make(map[string]*common.Treatment),
			}
			for _, strength := range strengths {
				treatment, err := h.eRxAPI.SelectMedication(0, name, strength)
				if err != nil {
					www.APIInternalError(w, r, err)
					return
				}
				res.Strengths[strength] = treatment
			}
			results = append(results, res)
		}
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Results []*drugSearchResult            `json:"results"`
		Details map[string]*common.DrugDetails `json:"details"`
	}{
		Results: results,
		Details: detailsMap,
	})
}
