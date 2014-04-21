package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

type TGView interface {
	Validate() error
}

type TGSmallDividerView struct {
	Type string `json:"type"`
}

func (v *TGSmallDividerView) Validate() error {
	v.Type = "view:small_divider"
	return nil
}

type TGLargeDividerView struct {
	Type string `json:"type"`
}

func (v *TGLargeDividerView) Validate() error {
	v.Type = "view:large_divider"
	return nil
}

type TGImageView struct {
	Type        string `json:"type"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ImageURL    string `json:"image_url"`
	// TODO insets
}

type TGIconTitleSubtitleView struct {
	Type     string `json:"type"`
	IconURL  string `json:"icon_url"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

func (v *TGIconTitleSubtitleView) Validate() error {
	v.Type = "view:icon_title_subtitle_view"
	return nil
}

type TGTextView struct {
	Type  string `json:"type"`
	Style string `json:"style,omitempty"`
	Text  string `json:"text"`
}

func (v *TGTextView) Validate() error {
	v.Type = "view:text"
	return nil
}

type TGIconTextView struct {
	Type       string `json:"type"`
	IconURL    string `json:"icon_url"`
	IconWidth  int    `json:"icon_width"`
	IconHeight int    `json:"icon_height"`
	Style      string `json:"style"`
	Text       string `json:"text"`
}

func (v *TGIconTextView) Validate() error {
	v.Type = "view:icon_text_view"
	return nil
}

type TGSnippetDetailsView struct {
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Details string `json:"details"`
}

func (v *TGSnippetDetailsView) Validate() error {
	v.Type = "view:snippet_details"
	return nil
}

type TGListElementView struct {
	Type         string `json:"type"`
	ElementStyle string `json:"element_style"` // numbered, dont
	Number       int    `json:"number,omitempty"`
	Text         string `json:"text"`
}

func (v *TGListElementView) Validate() error {
	if v.ElementStyle != "numbered" && v.ElementStyle != "dont" {
		return errors.New("ListElementView expects ElementStyle of numbered or dont, not " + v.ElementStyle)
	}
	v.Type = "view:list_element"
	return nil
}

type TGPlainButtonView struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	TapURL string `json:"tap_url"`
}

func (v *TGPlainButtonView) Validate() error {
	v.Type = "view:plain_button"
	return nil
}

type TGButtonView struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	TapURL  string `json:"tap_url"`
	IconURL string `json:"icon_url"`
}

func (v *TGButtonView) Validate() error {
	v.Type = "view:button"
	return nil
}

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
}

type PatientTreatmentGuideHandler struct {
	DataAPI api.DataAPI
}

type DoctorTreatmentGuideHandler struct {
	DataAPI api.DataAPI
}

func NewPatientTreatmentGuideHandler(dataAPI api.DataAPI) *PatientTreatmentGuideHandler {
	return &PatientTreatmentGuideHandler{DataAPI: dataAPI}
}

func NewDoctorTreatmentGuideHandler(dataAPI api.DataAPI) *DoctorTreatmentGuideHandler {
	return &DoctorTreatmentGuideHandler{DataAPI: dataAPI}
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

	patientID, err := h.DataAPI.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	}

	treatment, err := h.DataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get treatment: "+err.Error())
		return
	} else if treatment == nil {
		WriteUserError(w, http.StatusNotFound, "Unknown treatment")
		return
	}

	if treatment.PatientId.Int64() != patientID {
		WriteUserError(w, http.StatusForbidden, "Patient does not have access to the given treatment")
		return
	}

	treatmentGuideResponse(h.DataAPI, w, treatment)
}

func (h *DoctorTreatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	doctorID, err := h.DataAPI.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get doctor: "+err.Error())
		return
	}

	treatment, err := h.DataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get treatment: "+err.Error())
		return
	} else if treatment == nil {
		WriteUserError(w, http.StatusNotFound, "Unknown treatment")
		return
	}

	if err := verifyDoctorPatientRelationship(h.DataAPI, treatment.Doctor, treatment.Patient); err != nil {
		golog.Warningf("Doctor %d does not have access to treatment %d: %s", doctorID, treatment.Id.Int64(), err.Error())
		WriteUserError(w, http.StatusForbidden, "Doctor does not have access to the given treatment")
		return
	}

	treatmentGuideResponse(h.DataAPI, w, treatment)
}

func treatmentGuideResponse(dataAPI api.DataAPI, w http.ResponseWriter, treatment *common.Treatment) {
	ndc := treatment.DrugDBIds[erx.NDC]
	if ndc == "" {
		WriteUserError(w, http.StatusNotFound, "NDC unknown")
		return
	}

	details, err := dataAPI.DrugDetails(ndc)
	if err == api.NoRowsError {
		WriteUserError(w, http.StatusNotFound, "No details available")
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get drug details: "+err.Error())
		return
	}

	// Format drug details into views

	views := []TGView{
		&TGIconTitleSubtitleView{
			IconURL:  "spruce:///images/icon_rx",
			Title:    details.Name,
			Subtitle: details.Subtitle,
		},
		&TGSmallDividerView{},
		&TGTextView{
			Style: "small_gray",
			Text:  details.Description,
		},
		&TGLargeDividerView{},
		&TGIconTextView{
			// TODO: This icon info isn't robust or likely accurate
			IconURL:    fmt.Sprintf("spruce:///images/doctor_photo_%s_%s", treatment.Doctor.FirstName, treatment.Doctor.LastName),
			IconWidth:  32,
			IconHeight: 32,
			Text:       fmt.Sprintf("Dr. %s's Instructions", treatment.Doctor.LastName),
			Style:      "section_header",
		},
		&TGSmallDividerView{},
		&TGTextView{
			Text: treatment.PatientInstructions,
		},
		&TGLargeDividerView{},
		&TGTextView{
			Text:  "What to Know",
			Style: "section_header",
		},
		&TGSmallDividerView{},
	}

	if len(details.Warnings) != 0 {
		views = append(views, &TGTextView{
			Text:  "Warnings",
			Style: "subheader",
		})
		for _, s := range details.Warnings {
			views = append(views, &TGTextView{
				Text:  s,
				Style: "warning",
			})
		}
	}

	if len(details.Precautions) != 0 {
		views = append(views, &TGTextView{
			Text:  "Precautions",
			Style: "subheader",
		})
		for _, p := range details.Precautions {
			views = append(views, &TGSnippetDetailsView{
				Snippet: p.Snippet,
				Details: p.Details,
			})
		}
	}

	views = append(views,
		&TGLargeDividerView{},
		&TGTextView{
			Text:  "How to Use " + details.Name,
			Style: "section_header",
		},
		&TGSmallDividerView{},
	)

	for i, s := range details.HowToUse {
		views = append(views, &TGListElementView{
			ElementStyle: "numbered",
			Number:       i + 1,
			Text:         s,
		})
	}

	if len(details.DoNots) != 0 {
		views = append(views, &TGSmallDividerView{})
		for _, s := range details.DoNots {
			views = append(views, &TGListElementView{
				ElementStyle: "dont",
				Text:         s,
			})
		}
	}

	views = append(views,
		&TGLargeDividerView{},
		&TGTextView{
			Text:  "Message Your Doctor If\u2026",
			Style: "section_header",
		},
		&TGSmallDividerView{},
	)

	for _, s := range details.MessageDoctorIf {
		views = append(views, &TGTextView{
			Text: s,
		})
	}

	views = append(views,
		&TGPlainButtonView{
			Text:   fmt.Sprintf("View all %s side effects", details.Name),
			TapURL: "spruce:///action/view_side_effects",
		},
		&TGButtonView{
			Text:    "Message Dr. " + treatment.Doctor.LastName,
			IconURL: "spruce:///images/icon_message",
			TapURL:  "spruce:///action/message_doctor",
		},
	)

	for _, v := range views {
		if err := v.Validate(); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Failed to render views: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]TGView{"views": views})
}
