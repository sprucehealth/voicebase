package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
)

type autocompleteHandler struct {
	dataAPI                       api.DataAPI
	erxAPI                        erx.ERxAPI
	allergicMedicationsQuestionId int64
}

const allergicMedicationsQuestionTag = "q_allergic_medication_entry"

func NewAutocompleteHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {

	a := &autocompleteHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	}

	// cache the allergic medications question id at startup so that we can return allergy related medications when the patient
	// is on the question where we ask if the patient is allergic to any medications
	questionInfos, err := dataAPI.GetQuestionInfoForTags([]string{allergicMedicationsQuestionTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err)
	} else if len(questionInfos) != 1 {
		panic(fmt.Sprintf("expected 1 question to be returned with tag %s instead got %d", allergicMedicationsQuestionTag, len(questionInfos)))
	}
	a.allergicMedicationsQuestionId = questionInfos[0].QuestionId

	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(a,
				[]string{api.PATIENT_ROLE, api.DOCTOR_ROLE})),
		[]string{"GET"})
}

type AutocompleteRequestData struct {
	SearchString string `schema:"query,required"`
	QuestionId   int64  `schema:"question_id"`
}

type AutocompleteResponse struct {
	Suggestions []*Suggestion `json:"suggestions"`
}

type Suggestion struct {
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle,omitempty"`
	DrugInternalName string `json:"drug_internal_name,omitempty"`
}

func (s *autocompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &AutocompleteRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if requestData.QuestionId == s.allergicMedicationsQuestionId {
		s.handleAutocompleteForAllergicMedications(requestData, w, r)
		return
	}

	s.handleAutocompleteForDrugs(requestData, w, r)
}

func (s *autocompleteHandler) handleAutocompleteForAllergicMedications(requestData *AutocompleteRequestData, w http.ResponseWriter, r *http.Request) {
	searchResults, err := s.erxAPI.SearchForAllergyRelatedMedications(requestData.SearchString)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	autocompleteResponse := &AutocompleteResponse{
		Suggestions: make([]*Suggestion, len(searchResults)),
	}

	// format the results as they are returned in lowercase form
	for i, searchResultItem := range searchResults {
		autocompleteResponse.Suggestions[i] = &Suggestion{Title: strings.Title(searchResultItem)}
	}

	apiservice.WriteJSON(w, autocompleteResponse)
}

func (s *autocompleteHandler) handleAutocompleteForDrugs(requestData *AutocompleteRequestData, w http.ResponseWriter, r *http.Request) {
	var searchResults []string
	var err error
	switch apiservice.GetContext(r).Role {
	case api.DOCTOR_ROLE:
		doctor, err := s.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		searchResults, err = s.erxAPI.GetDrugNamesForDoctor(doctor.DoseSpotClinicianId, requestData.SearchString)
	case api.PATIENT_ROLE:
		searchResults, err = s.erxAPI.GetDrugNamesForPatient(requestData.SearchString)
	default:
		apiservice.WriteAccessNotAllowedError(w, r)
	}
	if err != nil {
		apiservice.WriteError(err, w, r)
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
			autocompleteResponse.Suggestions[i] = &Suggestion{Title: searchResult[:openBracket], Subtitle: SpecialTitle(subtitle), DrugInternalName: searchResult}
		} else {
			autocompleteResponse.Suggestions[i] = &Suggestion{Title: searchResult}
		}
	}

	apiservice.WriteJSON(w, autocompleteResponse)
}

// Content in the paranthesis of a drug name is returned as Oral - powder for reconstitution
// This function attempts to convert the subtitle to Oral - Powder for reconstitution
func SpecialTitle(s string) string {
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
