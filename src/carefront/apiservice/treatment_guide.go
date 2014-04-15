package apiservice

import (
	"carefront/api"
	"net/http"

	"github.com/gorilla/schema"
)

type SimpleView struct {
	Types []string `json:"types"` // view:small_divider, view:large_divider
}

type ImageView struct {
	Types       []string `json:"types"` // view:image
	ImageWidth  int      `json:"image_width"`
	ImageHeight int      `json:"image_height"`
	ImageURL    string   `json:"image_url"`
	// TODO insets
}

type IconTitleSubtitleView struct {
	Types    []string `json:"types"` // view:icon_title_subtitle_view
	IconURL  string   `json:"icon_url"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
}

type TextView struct {
	Types []string `json:"types"` // view:text
	Style string   `json:"style,omitempty"`
	Text  string   `json:"text"`
}

type IconTextView struct {
	Types      []string `json:"types"` // view:icon_text_view
	IconURL    string   `json:"icon_url"`
	IconWidth  int      `json:"icon_width"`
	IconHeight int      `json:"icon_height"`
	Style      string   `json:"style"`
	Text       string   `json:"text"`
}

type SnippetDetailsView struct {
	Types   []string `json:"types"` // view:snippet_details
	Snippet string   `json:"snippet"`
	Details string   `json:"details"`
}

type ListElementView struct {
	Types        []string `json:"types"`         // view:list_element
	ElementStyle string   `json:"element_style"` // numbered, dont
	Number       int      `json:"number,omitempty"`
	Text         string   `json:"text"`
}

type PlainButtonView struct {
	Types  []string `json:"types"` // view:plain_button
	Text   string   `json:"text"`
	TapURL string   `json:"tap_url"`
}

type ButtonView struct {
	Types   []string `json:"types"` // view:button
	Text    string   `json:"text"`
	TapURL  string   `json:"tap_url"`
	IconURL string   `json:"icon_url"`
}

type VisitHeaderView struct {
	Types    []string `json:"types"` // view:visit_header
	ImageURL string   `json:"image_url"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
}

type PatientTreatmentGuideHandler struct {
	DataApi api.DataAPI
	AuthApi thriftapi.Auth
}

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
}

func NewPatientTreatmentGuideHandler(dataApi api.DataAPI, authApi thriftapi.Auth) *PatientTreatmentGuideHandler {
	return &PatientTreatmentGuideHandler{DataApi: dataApi, AuthApi: authApi}
}

func (h *PatientTreatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(TreatmentGuideRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// patient :=
}
