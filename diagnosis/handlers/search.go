package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

var (
	defaultMaxResults = 50
)

type searchHandler struct {
	dataAPI      api.DataAPI
	diagnosisAPI diagnosis.API
}

type DiagnosisSearchResult struct {
	Sections []*ResultSection `json:"result_sections"`
}

type ResultSection struct {
	Title string        `json:"title"`
	Items []*ResultItem `json:"items"`
}

type ResultItem struct {
	Title     string             `json:"title"`
	Subtitle  string             `json:"subtitle,omitempty"`
	Diagnosis *AbridgedDiagnosis `json:"abridged_diagnosis"`
}

type AbridgedDiagnosis struct {
	CodeID     string `json:"code_id"`
	Code       string `json:"display_diagnosis_code"`
	Title      string `json:"title"`
	Synonyms   string `json:"synonyms,omitempty"`
	HasDetails bool   `json:"has_details"`
}

func NewSearchHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API) http.Handler {
	return apiservice.SupportedRoles(
		httputil.SupportedMethods(
			apiservice.NoAuthorizationRequired(&searchHandler{
				dataAPI:      dataAPI,
				diagnosisAPI: diagnosisAPI,
			}), httputil.Get), api.RoleDoctor)
}

func (s *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	query := r.FormValue("query")
	pathwayTag := s.determinePathwayTag(r)

	// if the query is empty, return the common diagnoses set
	// pertaining to the pathway
	if len(query) == 0 {
		title, diagnosisCodeIDs, err := s.dataAPI.CommonDiagnosisSet(pathwayTag)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(err, w, r)
			return
		}

		diagnosesMap, err := s.diagnosisAPI.DiagnosisForCodeIDs(diagnosisCodeIDs)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		diagnosesList := make([]*diagnosis.Diagnosis, len(diagnosesMap))
		for i, codeID := range diagnosisCodeIDs {
			diagnosesList[i] = diagnosesMap[codeID]
		}

		response, err := s.createResponseFromDiagnoses(diagnosesList, true, title)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, response)
		return
	}

	var maxResults int
	var err error

	// parse user specified numResults
	if numResults := r.FormValue("max_results"); numResults == "" {
		maxResults = defaultMaxResults
	} else if maxResults, err = strconv.Atoi(numResults); err != nil {
		apiservice.WriteValidationError(
			fmt.Sprintf("Invalid max_results parameter: %s", err.Error()), w, r)
		return
	}

	var diagnoses []*diagnosis.Diagnosis
	var queriedUsingDiagnosisCode bool

	// search for diagnoses by code if the query resembles a diagnosis code
	if resemblesCode(query) {
		diagnoses, err = s.diagnosisAPI.SearchDiagnosesByCode(query, maxResults)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		queriedUsingDiagnosisCode = (len(diagnoses) > 0)
	} else if len(query) < 3 {
		httputil.JSONResponse(w, http.StatusOK, &DiagnosisSearchResult{})
		return
	}

	// if no diagnoses found, then do a general search for diagnoses
	if len(diagnoses) == 0 {
		diagnoses, err = s.diagnosisAPI.SearchDiagnoses(query, maxResults)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// if no diagnoses found yet, then fall back to fuzzy string matching
	if len(diagnoses) == 0 {
		diagnoses, err = s.diagnosisAPI.FuzzyTextSearchDiagnoses(query, maxResults)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var title string
	switch l := len(diagnoses); {
	case l == 0:
		title = "0 Results"
	case l > 1:
		title = strconv.Itoa(l) + " Results"
	case l == 1:
		title = "1 Result"
	}

	response, err := s.createResponseFromDiagnoses(diagnoses, queriedUsingDiagnosisCode, title)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

func (s *searchHandler) createResponseFromDiagnoses(
	diagnoses []*diagnosis.Diagnosis,
	queriedUsingDignosisCode bool,
	sectionTitle string) (*DiagnosisSearchResult, error) {
	codeIDs := make([]string, len(diagnoses))
	for i, diagnosis := range diagnoses {
		codeIDs[i] = diagnosis.ID
	}

	// make requests in parallel to get indicators for any codeIDS
	// with additional details and synonyms for diagnoses
	p := conc.NewParallel()

	var codeIDsWithDetails map[string]bool
	p.Go(func() error {
		var err error
		codeIDsWithDetails, err = s.dataAPI.DiagnosesThatHaveDetails(codeIDs)
		return errors.Trace(err)
	})

	var synonymMap map[string][]string
	p.Go(func() error {
		var err error
		synonymMap, err = s.diagnosisAPI.SynonymsForDiagnoses(codeIDs)
		return errors.Trace(err)
	})

	if err := p.Wait(); err != nil {
		return nil, err
	}

	items := make([]*ResultItem, len(diagnoses))
	for i, diagnosis := range diagnoses {

		synonyms := strings.Join(synonymMap[diagnosis.ID], ", ")
		items[i] = &ResultItem{
			Title: diagnosis.Description,
			Diagnosis: &AbridgedDiagnosis{
				CodeID:     diagnosis.ID,
				Code:       diagnosis.Code,
				Title:      diagnosis.Description,
				Synonyms:   synonyms,
				HasDetails: codeIDsWithDetails[diagnosis.ID],
			},
		}

		// appropriately return the subtitle based on whether the
		// user queried using a diagnosis code or synonyms
		if queriedUsingDignosisCode {
			items[i].Subtitle = diagnosis.Code
		} else {
			items[i].Subtitle = synonyms
		}
	}

	return &DiagnosisSearchResult{
		Sections: []*ResultSection{
			{
				Title: sectionTitle,
				Items: items,
			},
		},
	}, nil
}

// determinePathwayTag returns the pathwayTag if directly found in the caller.
// If not, it falls back to the patient_visit_id to lookup the pathwayTag.
// Reason for this is that the clients consuming this API today don't send
// the pathway_id but do send the patient_visit_id.The goal is for all consuming
// clients to send the pathway_id if possible.
func (s *searchHandler) determinePathwayTag(r *http.Request) string {
	pathwayTag := r.FormValue("pathway_id")

	// return pathway tag immediately if specified
	if pathwayTag != "" {
		return pathwayTag
	}

	// if pathway tag is not specified then fall back to the patient_visit_id
	// (if specified) to pull out the pathwayTag
	if patientVisitIDStr := r.FormValue("patient_visit_id"); patientVisitIDStr != "" {
		patientVisitID, err := strconv.ParseInt(patientVisitIDStr, 10, 64)
		if err != nil {
			golog.Warningf("Unable to parse patient_visit_id: %s", err.Error())
			return api.AcnePathwayTag
		}

		patientVisit, err := s.dataAPI.GetPatientVisitFromID(patientVisitID)
		if err != nil {
			golog.Warningf("Unable to get patient_visit from id: %s", err.Error())
			return api.AcnePathwayTag
		}

		return patientVisit.PathwayTag
	}

	return api.AcnePathwayTag
}
