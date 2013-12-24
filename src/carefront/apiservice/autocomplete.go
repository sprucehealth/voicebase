package apiservice

import (
	"carefront/api"
	"carefront/libs/erx"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
	"unicode"
)

type AutocompleteHandler struct {
	ERxApi erx.ERxAPI
	Role   string
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
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle,omitempty"`
	InternalId string `json:"internal_name"`
}

func (s *AutocompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(AutocompleteRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input paramaters: "+err.Error())
		return
	}

	var searchResults []string
	if s.Role == api.DOCTOR_ROLE {
		searchResults, err = s.ERxApi.GetDrugNamesForDoctor(requestData.SearchString)
	} else {
		searchResults, err = s.ERxApi.GetDrugNamesForPatient(requestData.SearchString)
	}
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get search results for drugs: "+err.Error())
		return
	}

	// populate suggestions
	autocompleteResponse := &AutocompleteResponse{}
	autocompleteResponse.Suggestions = make([]Suggestion, len(searchResults))
	for i, searchResult := range searchResults {
		// move anything within brackets to become the subtitle
		// TODO: Cache the results so that we don't have to constantly
		// parse the suggestions to break them up into title and subtitle,
		// and also so that suggestions are quicker to return
		openBracket := strings.Index(searchResult, "(")
		if openBracket != -1 {
			subtitle := searchResult[openBracket+1 : len(searchResult)-1]

			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult[:openBracket], Subtitle: SpecialTitle(subtitle), InternalId: searchResult}
		} else {
			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult}
		}
	}
	autocompleteResponse.Title = "Common Treatments"
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
