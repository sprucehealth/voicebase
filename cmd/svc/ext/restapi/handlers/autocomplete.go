package handlers

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type autocompleteHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

const allergicMedicationsQuestionTag = "q_allergic_medication_entry"

func NewAutocompleteHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) httputil.ContextHandler {
	a := &autocompleteHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	}
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(a,
				api.RolePatient, api.RoleDoctor)),
		httputil.Get)
}

type autocompleteRequestData struct {
	SearchString string `schema:"query,required"`
	QuestionID   int64  `schema:"question_id"`
}

type AutocompleteResponse struct {
	Suggestions []*Suggestion `json:"suggestions"`
}

type Suggestion struct {
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle,omitempty"`
	DrugInternalName string `json:"drug_internal_name,omitempty"`
}

func (s *autocompleteHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestData := &autocompleteRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	vq, err := s.dataAPI.VersionedQuestionFromID(requestData.QuestionID)
	if !api.IsErrNotFound(err) && err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if vq != nil && vq.QuestionTag == allergicMedicationsQuestionTag {
		s.handleAutocompleteForAllergicMedications(ctx, requestData, w, r)
		return
	}

	s.handleAutocompleteForDrugs(ctx, requestData, w, r)
}

func (s *autocompleteHandler) handleAutocompleteForAllergicMedications(ctx context.Context, requestData *autocompleteRequestData, w http.ResponseWriter, r *http.Request) {
	searchResults, err := s.erxAPI.SearchForAllergyRelatedMedications(requestData.SearchString)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	autocompleteResponse := &AutocompleteResponse{
		Suggestions: make([]*Suggestion, len(searchResults)),
	}

	// format the results as they are returned in lowercase form
	for i, searchResultItem := range searchResults {
		autocompleteResponse.Suggestions[i] = &Suggestion{Title: strings.Title(searchResultItem)}
	}

	httputil.JSONResponse(w, http.StatusOK, autocompleteResponse)
}

func (s *autocompleteHandler) handleAutocompleteForDrugs(ctx context.Context, requestData *autocompleteRequestData, w http.ResponseWriter, r *http.Request) {
	var searchResults []string
	var err error
	account := apiservice.MustCtxAccount(ctx)
	switch account.Role {
	case api.RoleDoctor:
		doctor, e := s.dataAPI.GetDoctorFromAccountID(account.ID)
		if e != nil {
			apiservice.WriteError(ctx, e, w, r)
			return
		}
		searchResults, err = s.erxAPI.GetDrugNamesForDoctor(doctor.DoseSpotClinicianID, requestData.SearchString)
	case api.RolePatient:
		searchResults, err = s.erxAPI.GetDrugNamesForPatient(requestData.SearchString)
	default:
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// populate suggestions
	autocompleteResponse := &AutocompleteResponse{
		Suggestions: make([]*Suggestion, len(searchResults)),
	}
	for i, searchResult := range searchResults {
		// move anything within brackets to become the subtitle
		// TODO: Cache the results so that we don't have to constantly
		// parse the suggestions to break them up into title and subtitle,
		// and also so that suggestions are quicker to return
		openBracket := strings.Index(searchResult, "(")
		if openBracket != -1 {
			subtitle := searchResult[openBracket+1 : len(searchResult)-1]
			autocompleteResponse.Suggestions[i] = &Suggestion{Title: searchResult[:openBracket], Subtitle: specialTitle(subtitle), DrugInternalName: searchResult}
		} else {
			autocompleteResponse.Suggestions[i] = &Suggestion{Title: searchResult}
		}
	}

	httputil.JSONResponse(w, http.StatusOK, autocompleteResponse)
}

// Content in the paranthesis of a drug name is returned as Oral - powder for reconstitution
// This function attempts to convert the subtitle to Oral - Powder for reconstitution
func specialTitle(s string) string {
	// Use a closure here to remember state.
	// Hackish but effective. Depends on Map scanning in order and calling
	// the closure once per rune.
	firstLetter := false
	hyphenFound := false
	spaceAfterHyphenFound := false
	letterAfterSpaceAfterHyphenFound := false
	return strings.Map(
		func(r rune) rune {
			if !firstLetter {
				firstLetter = true
				return unicode.ToTitle(r)
			}

			if hyphenFound {
				if !spaceAfterHyphenFound {
					spaceAfterHyphenFound = true
					if r != ' ' {
						return unicode.ToTitle(r)
					}
				} else if !letterAfterSpaceAfterHyphenFound {
					letterAfterSpaceAfterHyphenFound = true
					return unicode.ToTitle(r)
				}
			}

			if r == '-' {
				hyphenFound = true
			}

			return r

		},
		s)
}
