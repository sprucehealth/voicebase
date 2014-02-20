package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/schema"
)

type AutocompleteHandler struct {
	DataApi api.DataAPI
	ERxApi  erx.ERxAPI
	Role    string
}

type AutocompleteRequestData struct {
	SearchString string `schema:"query,required"`
	QuestionId   string `schema:"question_id"`
}

type AutocompleteResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
	Title       string       `json:"title"`
}

type Suggestion struct {
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle,omitempty"`
	DrugInternalName string `json:"drug_internal_name,omitempty"`
}

func (s *AutocompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData AutocompleteRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input paramaters: "+err.Error())
		return
	}

	var searchResults []string
	var err error
	var doctor *common.Doctor
	if s.Role == api.DOCTOR_ROLE {
		doctor, err = s.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from accountId: "+err.Error())
			return
		}
		searchResults, err = s.ERxApi.GetDrugNamesForDoctor(doctor.DoseSpotClinicianId, requestData.SearchString)
	} else {

		patient, err := s.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id: "+err.Error())
			return
		}
		careTeam, err := s.DataApi.GetCareTeamForPatient(patient.PatientId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient: "+err.Error())
			return
		}
		doctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
		doctor, err := s.DataApi.GetDoctorFromId(doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from id: "+err.Error())
			return
		}

		searchResults, err = s.ERxApi.GetDrugNamesForPatient(doctor.DoseSpotClinicianId, requestData.SearchString)
	}
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get search results for drugs: "+err.Error())
		return
	}

	// populate suggestions
	autocompleteResponse := &AutocompleteResponse{
		Title:       "Common Treatments",
		Suggestions: make([]Suggestion, len(searchResults)),
	}
	for i, searchResult := range searchResults {
		// move anything within brackets to become the subtitle
		// TODO: Cache the results so that we don't have to constantly
		// parse the suggestions to break them up into title and subtitle,
		// and also so that suggestions are quicker to return
		openBracket := strings.Index(searchResult, "(")
		if openBracket != -1 {
			subtitle := searchResult[openBracket+1 : len(searchResult)-1]

			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult[:openBracket], Subtitle: SpecialTitle(subtitle), DrugInternalName: searchResult}
		} else {
			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult}
		}
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, autocompleteResponse)
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
