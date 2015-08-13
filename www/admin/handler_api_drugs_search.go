package admin

import (
	"net/http"
	"strings"
	"sync"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

const maxDrugSearchResults = 10

type drugSearchAPIHandler struct {
	dataAPI api.DataAPI
	eRxAPI  erx.ERxAPI
}

type drugStrength struct {
	ParsedGenericName string                        `json:"parsed_generic_name"`
	Strength          string                        `json:"strength"`
	Error             string                        `json:"error,omitempty"`
	GuideID           int64                         `json:"guide_id,string"`
	Medication        *erx.MedicationSelectResponse `json:"medication"`
}

type drugSearchResult struct {
	Name      string          `json:"name"`
	Error     string          `json:"error,omitempty"`
	Strengths []*drugStrength `json:"strengths"`
}

func newDrugSearchAPIHandler(dataAPI api.DataAPI, eRxAPI erx.ERxAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&drugSearchAPIHandler{
		dataAPI: dataAPI,
		eRxAPI:  eRxAPI,
	}, httputil.Get)
}

func (h *drugSearchAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var results []*drugSearchResult

	query := r.FormValue("q")

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "SearchDrugs", map[string]interface{}{"query": query})

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

		ch := make(chan *drugSearchResult)
		var wg sync.WaitGroup
		wg.Add(len(names))

		for _, name := range names {
			go func(name string) {
				defer wg.Done()
				strengths, err := h.eRxAPI.SearchForMedicationStrength(0, name)
				if err != nil {
					golog.Warningf(err.Error())
					ch <- &drugSearchResult{
						Name:  name,
						Error: "Failed to fetch medication strengths",
					}
					return
				}
				res := &drugSearchResult{
					Name:      name,
					Strengths: make([]*drugStrength, 0, len(strengths)),
				}
				for _, strength := range strengths {
					s := &drugStrength{
						Strength: strength,
					}
					res.Strengths = append(res.Strengths, s)

					med, err := h.eRxAPI.SelectMedication(0, name, strength)
					if err != nil {
						golog.Warningf(err.Error())
						s.Error = "Failed to fetch"
					} else {
						s.Medication = med
						s.ParsedGenericName, err = erx.ParseGenericName(med)
						if err != nil {
							s.ParsedGenericName = "ERROR: " + err.Error()
						}
					}
				}
				ch <- res
			}(name)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		for res := range ch {
			results = append(results, res)
		}

		// Check for RX guides

		var guideQueries []*api.DrugDetailsQuery
		for _, r := range results {
			for _, s := range r.Strengths {
				if !strings.HasPrefix(s.ParsedGenericName, "ERROR") {
					guideQueries = append(guideQueries, &api.DrugDetailsQuery{
						NDC:         s.Medication.RepresentativeNDC,
						GenericName: s.ParsedGenericName,
						Route:       s.Medication.RouteDescription,
						Form:        s.Medication.DoseFormDescription,
					})
				}
			}
		}
		guideIDs, err := h.dataAPI.MultiQueryDrugDetailIDs(guideQueries)
		if err != nil {
			golog.Errorf("Failed to fetch rx guides: %s", err.Error())
		} else {
			i := 0
			for _, r := range results {
				for _, s := range r.Strengths {
					if !strings.HasPrefix(s.ParsedGenericName, "ERROR") {
						s.GuideID = guideIDs[i]
						i++
					}
				}
			}
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Results []*drugSearchResult `json:"results"`
	}{
		Results: results,
	})
}
