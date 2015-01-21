package patient_visit

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/sku"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type pathwayDetailsHandler struct {
	dataAPI api.DataAPI
}

type pathwayDetailsResponse struct {
	Pathways []*pathwayDetails `json:"pathway_details_screens"`
}

type pathwayDetails struct {
	ID     int64                 `json:"pathway_id,string"`
	Screen *pathwayDetailsScreen `json:"screen"`
}

type pathwayDetailsScreen struct {
	Type                   string                `json:"type"`
	Title                  string                `json:"title"`
	Views                  []pdView              `json:"views,omitempty"`
	RightHeaderButtonTitle string                `json:"right_header_button_title,omitempty"`
	BottomButtonTitle      string                `json:"bottom_button_title,omitempty"`
	BottomButtonTapURL     *app_url.SpruceAction `json:"bottom_button_tap_url,omitempty"`
	// If type == "generic_message"
	ContentText    string `json:"content_text,omitempty"`
	ContentSubtext string `json:"contenxt_subtext,omitempty"`
	PhotoURL       string `json:"photo_url,omitempty"`
}

type pdView interface {
	TypeName() string
	Validate() error
}

type pdCardView struct {
	Type  string   `json:"type"`
	Title string   `json:"title"`
	Views []pdView `json:"views"`
}

type pdCheckboxTextListView struct {
	Type   string   `json:"type"`
	Titles []string `json:"titles"`
}

type pdFilledButtonView struct {
	Type   string                `json:"type"`
	Title  string                `json:"title"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type pdDoctorProfilePhotosView struct {
	Type      string   `json:"type"`
	PhotoURLs []string `json:"photo_urls"`
}

type pdOutlinedButtonView struct {
	Type   string                `json:"type"`
	Title  string                `json:"title"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type pdBodyTextView struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (v *pdCardView) TypeName() string {
	return "pathway_details:card_view"
}

func (v *pdCardView) Validate() error {
	v.Type = v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("card_view.tile required")
	}
	for _, v := range v.Views {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (v *pdCheckboxTextListView) TypeName() string {
	return "pathway_details:checkbox_text_list_view"
}

func (v *pdCheckboxTextListView) Validate() error {
	v.Type = v.TypeName()
	if len(v.Titles) == 0 {
		return fmt.Errorf("checkbox_text_list_view.titled required and must not be empty")
	}
	return nil
}

func (v *pdFilledButtonView) TypeName() string {
	return "pathway_details:filled_button_view"
}

func (v *pdFilledButtonView) Validate() error {
	v.Type = v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("filled_button_view.title required")
	}
	if v.TapURL == nil {
		return fmt.Errorf("filled_button_view.tap_url required")
	}
	return nil
}

func (v *pdOutlinedButtonView) TypeName() string {
	return "pathway_details:outlined_button_view"
}

func (v *pdOutlinedButtonView) Validate() error {
	v.Type = v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("outlined_button_view.title required")
	}
	if v.TapURL == nil {
		return fmt.Errorf("outlined_button_view.tap_url required")
	}
	return nil
}

func (v *pdDoctorProfilePhotosView) TypeName() string {
	return "pathway_details:doctor_profile_photos_view"
}

func (v *pdDoctorProfilePhotosView) Validate() error {
	v.Type = v.TypeName()
	if len(v.PhotoURLs) == 0 {
		return fmt.Errorf("doctor_profile_photos_view.photo_urls required and may not be empty")
	}
	return nil
}

func (v *pdBodyTextView) TypeName() string {
	return "pathway_details:body_text_view"
}

func (v *pdBodyTextView) Validate() error {
	v.Type = v.TypeName()
	if v.Text == "" {
		return fmt.Errorf("body_text_view.text required")
	}
	return nil
}

func NewPathwayDetailsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&pathwayDetailsHandler{
			dataAPI: dataAPI,
		}),
		[]string{"GET"})
}

func (h *pathwayDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathwayIDs, err := parseIDList(r.FormValue("pathway_id"))
	if err != nil {
		apiservice.WriteBadRequestError(errors.New("invalid format for pathway_id param"), w, r)
		return
	}
	if len(pathwayIDs) == 0 {
		// empty response for an empty request (eye for an eye)
		apiservice.WriteJSON(w, &pathwayDetailsResponse{
			Pathways: []*pathwayDetails{},
		})
		return
	}
	pathways, err := h.dataAPI.Pathways(pathwayIDs, api.POWithDetails|api.POActiveOnly)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var patientID int64
	var activeCases map[int64]int64

	ctx := apiservice.GetContext(r)
	if ctx.AccountID != 0 && ctx.Role == api.PATIENT_ROLE {
		patientID, err = h.dataAPI.GetPatientIDFromAccountID(ctx.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		activeCases, err = h.dataAPI.ActiveCaseIDsForPathways(patientID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var fetchedCareTeams bool
	var careTeams map[int64]*common.PatientCareTeam

	res := &pathwayDetailsResponse{}
	for _, p := range pathways {
		doctors, err := h.dataAPI.DoctorsForPathway(p.ID, 4)
		if err != nil {
			golog.Errorf("Failed to lookup doctors for pathway %d '%s': %s", p.ID, p.Name, err)
		}
		// TODO: for now grabbing acne visit cost but this should be specific to the pathway
		cost, err := h.dataAPI.GetActiveItemCost(sku.AcneVisit)
		if err != nil {
			golog.Errorf("Failed to get cost for pathway %d '%s': %s", p.ID, p.Name, err)
		}

		var screen *pathwayDetailsScreen
		if caseID := activeCases[p.ID]; caseID != 0 {
			if !fetchedCareTeams {
				careTeams, err = h.dataAPI.GetCareTeamsForPatientByCase(patientID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				fetchedCareTeams = true
			}
			screen = activeCaseScreen(careTeams[caseID], caseID, p)
		} else if p.Details == nil {
			golog.Errorf("Details missing for pathway %d '%s'", p.ID, p.Name)
			screen = detailsMissingScreen(p)
		} else {
			screen = merchandisingScreen(p, doctors, cost)
		}
		for _, v := range screen.Views {
			if err := v.Validate(); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
		res.Pathways = append(res.Pathways, &pathwayDetails{
			ID:     p.ID,
			Screen: screen,
		})
	}

	if res.Pathways == nil {
		// Return an empty list instead of null if no pathways found
		res.Pathways = []*pathwayDetails{}
	}

	apiservice.WriteJSON(w, res)
}

func merchandisingScreen(pathway *common.Pathway, doctors []*common.Doctor, cost *common.ItemCost) *pathwayDetailsScreen {
	if pathway.Details.WhoWillTreatMe == "" {
		golog.Errorf("Field WhoWillTreatMe missing for pathway %d '%s'", pathway.ID, pathway.Name)
	}
	if pathway.Details.RightForMe == "" {
		golog.Errorf("Field RightForMe missing for pathway %d '%s'", pathway.ID, pathway.Name)
	}
	var didYouKnow string
	if len(pathway.Details.DidYouKnow) != 0 {
		didYouKnow = pathway.Details.DidYouKnow[rand.Intn(len(pathway.Details.DidYouKnow))]
	} else {
		golog.Errorf("Field DidYouKnow missing for pathway %d '%s'", pathway.ID, pathway.Name)
	}

	doctorImageURLs := make([]string, len(doctors))
	for i, d := range doctors {
		doctorImageURLs[i] = d.LargeThumbnailURL
	}

	views := []pdView{
		&pdCardView{
			Title: "What's included?",
			Views: []pdView{
				&pdCheckboxTextListView{
					Titles: pathway.Details.WhatIsIncluded,
				},
				&pdFilledButtonView{
					Title:  "Sample Treatment Plan",
					TapURL: app_url.ViewSampleTreatmentPlanAction(pathway.ID),
				},
			},
		},
		&pdCardView{
			Title: "Who will treat me?",
			Views: []pdView{
				&pdDoctorProfilePhotosView{
					PhotoURLs: doctorImageURLs,
				},
				&pdBodyTextView{
					Text: pathway.Details.WhoWillTreatMe,
				},
			},
		},
		&pdCardView{
			Title: "Is this right for me?",
			Views: []pdView{
				&pdBodyTextView{
					Text: pathway.Details.RightForMe,
				},
				&pdOutlinedButtonView{
					Title:  "Read More",
					TapURL: app_url.ViewPathwayFAQ(pathway.ID),
				},
			},
		},
		&pdCardView{
			Title: "Did you know?",
			Views: []pdView{
				&pdBodyTextView{
					Text: didYouKnow,
				},
			},
		},
	}
	return &pathwayDetailsScreen{
		Type:  "merchandising",
		Title: fmt.Sprintf("%s Visit", pathway.Name),
		Views: views,
		RightHeaderButtonTitle: cost.TotalCost().String(),
		BottomButtonTitle:      "Choose Your Doctor",
		BottomButtonTapURL:     app_url.ViewChooseDoctorScreen(),
	}
}

func activeCaseScreen(careTeam *common.PatientCareTeam, caseID int64, pathway *common.Pathway) *pathwayDetailsScreen {
	var doctorName string
	var doctorThumbnailURL string
	if careTeam != nil {
		for _, a := range careTeam.Assignments {
			if a.ProviderRole == api.DOCTOR_ROLE {
				doctorName = a.ShortDisplayName
				doctorThumbnailURL = a.LargeThumbnailURL
				break
			}
		}
	}
	if doctorName == "" {
		golog.Errorf("Doctor not found in care team for case %d", caseID)
	}
	return &pathwayDetailsScreen{
		Type:               "generic_message",
		Title:              fmt.Sprintf("%s Visit", pathway.Name),
		BottomButtonTitle:  "Okay",
		BottomButtonTapURL: app_url.ViewHomeAction(),
		ContentText:        fmt.Sprintf("You have an existing %s case with %s.", pathway.Name, doctorName),
		ContentSubtext:     "Message your care team to ask about a follow up visit.",
		PhotoURL:           doctorThumbnailURL,
	}
}

func detailsMissingScreen(pathway *common.Pathway) *pathwayDetailsScreen {
	return &pathwayDetailsScreen{
		Type:           "generic_message",
		Title:          fmt.Sprintf("%s Visit", pathway.Name),
		ContentText:    "Sorry, but there seems to be a problem with the service.",
		ContentSubtext: "Please try to start a visit later.",
	}
}

func parseIDList(s string) ([]int64, error) {
	if len(s) == 0 {
		return nil, nil
	}
	// Counter the number of commas to preallocate the correct sized slice
	n := 1
	for _, r := range s {
		if r == ',' {
			n++
		}
	}
	ids := make([]int64, 0, n)
	for len(s) != 0 {
		sid := s
		if i := strings.IndexByte(s, ','); i > 0 {
			sid = s[:i]
			s = s[i+1:]
		} else {
			s = s[:0]
		}
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
