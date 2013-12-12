package apiservice

import (
	"carefront/libs/erx"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
)

type AutocompleteHandler struct {
	ERxApi erx.ERxAPI
}

type AutocompleteRequestData struct {
	SearchString string `schema:"query,required"`
	QuestionId   string `schema:"question_id,required"`
}

type AutocompleteResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
	Title       string       `json:"title"`
}

type Suggestion struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
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

	searchResults, err := s.ERxApi.GetDrugNames(requestData.SearchString)
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
			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult[:openBracket], Subtitle: searchResult[openBracket:]}
		} else {
			autocompleteResponse.Suggestions[i] = Suggestion{Title: searchResult}
		}
	}
	autocompleteResponse.Title = "Common Treatments"
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, autocompleteResponse)
}
