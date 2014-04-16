package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

type View interface {
	Validate() error
}

type SmallDividerView struct {
	Types []string `json:"types"`
}

func (v *SmallDividerView) Validate() error {
	v.Types = []string{"view:small_divider"}
	return nil
}

type LargeDividerView struct {
	Types []string `json:"types"`
}

func (v *LargeDividerView) Validate() error {
	v.Types = []string{"view:large_divider"}
	return nil
}

type ImageView struct {
	Types       []string `json:"types"`
	ImageWidth  int      `json:"image_width"`
	ImageHeight int      `json:"image_height"`
	ImageURL    string   `json:"image_url"`
	// TODO insets
}

type IconTitleSubtitleView struct {
	Types    []string `json:"types"`
	IconURL  string   `json:"icon_url"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
}

func (v *IconTitleSubtitleView) Validate() error {
	v.Types = []string{"view:icon_title_subtitle_view"}
	return nil
}

type TextView struct {
	Types []string `json:"types"`
	Style string   `json:"style,omitempty"`
	Text  string   `json:"text"`
}

func (v *TextView) Validate() error {
	v.Types = []string{"view:text"}
	return nil
}

type IconTextView struct {
	Types      []string `json:"types"`
	IconURL    string   `json:"icon_url"`
	IconWidth  int      `json:"icon_width"`
	IconHeight int      `json:"icon_height"`
	Style      string   `json:"style"`
	Text       string   `json:"text"`
}

func (v *IconTextView) Validate() error {
	v.Types = []string{"view:icon_text_view"}
	return nil
}

type SnippetDetailsView struct {
	Types   []string `json:"types"`
	Snippet string   `json:"snippet"`
	Details string   `json:"details"`
}

func (v *SnippetDetailsView) Validate() error {
	v.Types = []string{"view:snippet_details"}
	return nil
}

type ListElementView struct {
	Types        []string `json:"types"`
	ElementStyle string   `json:"element_style"` // numbered, dont
	Number       int      `json:"number,omitempty"`
	Text         string   `json:"text"`
}

func (v *ListElementView) Validate() error {
	if v.ElementStyle != "numbered" && v.ElementStyle != "dont" {
		return errors.New("ListElementView expects ElementStyle of numbered or dont, not " + v.ElementStyle)
	}
	v.Types = []string{"view:list_element"}
	return nil
}

type PlainButtonView struct {
	Types  []string `json:"types"`
	Text   string   `json:"text"`
	TapURL string   `json:"tap_url"`
}

func (v *PlainButtonView) Validate() error {
	v.Types = []string{"view:plain_button"}
	return nil
}

type ButtonView struct {
	Types   []string `json:"types"`
	Text    string   `json:"text"`
	TapURL  string   `json:"tap_url"`
	IconURL string   `json:"icon_url"`
}

func (v *ButtonView) Validate() error {
	v.Types = []string{"view:button"}
	return nil
}

type PatientTreatmentGuideHandler struct {
	DataAPI api.DataAPI
}

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
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

	patient, err := h.DataAPI.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	} else if patient == nil {
		WriteUserError(w, http.StatusNotFound, "Unknown patient")
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

	if treatment.PatientId != patient.PatientId.Int64() {
		WriteUserError(w, http.StatusForbidden, "Patient does not have access to the given treatment")
		return
	}

	doctor, err := h.DataAPI.GetDoctorFromId(treatment.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get doctor: "+err.Error())
		return
	}

	treatment.Patient = patient
	treatment.Doctor = doctor

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

	doctor, err := h.DataAPI.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get doctor: "+err.Error())
		return
	} else if doctor == nil {
		WriteUserError(w, http.StatusNotFound, "Unknown doctor")
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

	if treatment.DoctorId != doctor.DoctorId.Int64() {
		WriteUserError(w, http.StatusForbidden, "Doctor does not have access to the given treatment")
		return
	}

	patient, err := h.DataAPI.GetPatientFromId(treatment.PatientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	}

	treatment.Patient = patient
	treatment.Doctor = doctor

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

	views := []View{
		&IconTitleSubtitleView{
			IconURL: "spruce:///images/icon_rx",
			Title:   details.Name,
			// TODO Subtitle: details.
		},
		&SmallDividerView{},
		&TextView{
			Style: "small_gray",
			Text:  details.Description,
		},
		&LargeDividerView{},
		&IconTextView{
			// TODO: This icon info isn't robust or likely accurate
			IconURL:    fmt.Sprintf("spruce:///images/doctor_photo_%s_%s", treatment.Doctor.FirstName, treatment.Doctor.LastName),
			IconWidth:  32,
			IconHeight: 32,
			Text:       fmt.Sprintf("Dr. %s's Instructions", treatment.Doctor.LastName),
			Style:      "section_header",
		},
		&SmallDividerView{},
		&TextView{
			Text: treatment.PatientInstructions,
		},
		&LargeDividerView{},
		&TextView{
			Text:  "What to Know",
			Style: "section_header",
		},
		&SmallDividerView{},
	}

	if len(details.Warnings) != 0 {
		views = append(views, &TextView{
			Text:  "Warnings",
			Style: "subheader",
		})
		for _, s := range details.Warnings {
			views = append(views, &TextView{
				Text:  s,
				Style: "warning",
			})
		}
	}

	if len(details.Precautions) != 0 {
		views = append(views, &TextView{
			Text:  "Precautions",
			Style: "subheader",
		})
		for _, s := range details.Precautions {
			views = append(views, &TextView{
				Text:  s,
				Style: "warning",
			})
		}
		// TODO: the spreadsheet doesn't have snippet and details for precautions
		// for _, s := range details.Precautions {
		// 	views = append(views, &SnippetDetailsView{
		// 		Snippet:  "",
		// 		Details: "",
		// 	})
		// }
	}

	views = append(views,
		&LargeDividerView{},
		&TextView{
			Text:  "How to Use " + details.Name,
			Style: "section_header",
		},
		&SmallDividerView{},
	)

	for i, s := range details.HowToUse {
		views = append(views, &ListElementView{
			ElementStyle: "numbered",
			Number:       i + 1,
			Text:         s,
		})
	}

	if len(details.DoNots) != 0 {
		views = append(views, &SmallDividerView{})
		for _, s := range details.DoNots {
			views = append(views, &ListElementView{
				ElementStyle: "dont",
				Text:         s,
			})
		}
	}

	views = append(views,
		&LargeDividerView{},
		&TextView{
			Text:  "Message Your Doctor If\u2026",
			Style: "section_header",
		},
		&SmallDividerView{},
	)

	for _, s := range details.MessageDoctorIf {
		views = append(views, &TextView{
			Text: s,
		})
	}

	views = append(views,
		&PlainButtonView{
			Text:   fmt.Sprintf("View all %s side effects", details.Name),
			TapURL: "spruce:///action/view_side_effects",
		},
		&ButtonView{
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

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]View{"views": views})
}
