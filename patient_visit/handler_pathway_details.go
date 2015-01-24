package patient_visit

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/views"
)

type pathwayDetailsHandler struct {
	dataAPI api.DataAPI
}

type pathwayDetailsResponse struct {
	Pathways []*pathwayDetails `json:"pathway_details_screens"`
}

type pathwayDetails struct {
	PathwayTag string                `json:"pathway_id"`
	Screen     *pathwayDetailsScreen `json:"screen"`
	FAQ        *pathwayFAQ           `json:"faq,omitempty"`
}

type pathwayDetailsScreen struct {
	Type                   string                `json:"type"`
	Title                  string                `json:"title"`
	Views                  []views.View          `json:"views,omitempty"`
	RightHeaderButtonTitle string                `json:"right_header_button_title,omitempty"`
	BottomButtonTitle      string                `json:"bottom_button_title,omitempty"`
	BottomButtonTapURL     *app_url.SpruceAction `json:"bottom_button_tap_url,omitempty"`
	// If type == "generic_message"
	ContentText    string `json:"content_text,omitempty"`
	ContentSubtext string `json:"contenxt_subtext,omitempty"`
	PhotoURL       string `json:"photo_url,omitempty"`
}

type pathwayFAQ struct {
	Title string       `json:"title"`
	Views []views.View `json:"views"`
}

func NewPathwayDetailsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&pathwayDetailsHandler{
			dataAPI: dataAPI,
		}),
		[]string{"GET"})
}

func (h *pathwayDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathwayTags := strings.Split(r.FormValue("pathway_id"), ",")
	if len(pathwayTags) == 0 {
		// empty response for an empty request (eye for an eye)
		apiservice.WriteJSON(w, &pathwayDetailsResponse{
			Pathways: []*pathwayDetails{},
		})
		return
	}
	pathways, err := h.dataAPI.PathwaysForTags(pathwayTags, api.POWithDetails|api.POActiveOnly)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var patientID int64
	var activeCases map[string]int64

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
		doctors, err := h.dataAPI.DoctorsForPathway(p.Tag, 4)
		if err != nil {
			golog.Errorf("Failed to lookup doctors for pathway %d '%s': %s", p.ID, p.Name, err)
		}
		// TODO: for now grabbing acne visit cost but this should be specific to the pathway
		cost, err := h.dataAPI.GetActiveItemCost(sku.AcneVisit)
		if err != nil {
			golog.Errorf("Failed to get cost for pathway %d '%s': %s", p.ID, p.Name, err)
		}

		var screen *pathwayDetailsScreen
		var faq *pathwayFAQ
		if caseID := activeCases[p.Tag]; caseID != 0 {
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
			faq = &pathwayFAQ{
				Title: "Is this right for me?",
			}
			for i, aq := range p.Details.FAQ {
				if i != 0 {
					faq.Views = append(faq.Views, &views.LargeDivider{})
				}
				faq.Views = append(faq.Views,
					&views.Text{Text: aq.Question, Style: views.SectionHeaderStyle},
					&views.SmallDivider{},
					&views.Text{Text: aq.Answer},
				)
			}
			if err := views.Validate(faq.Views, "faq"); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
		if err := views.Validate(screen.Views, "pathway_details"); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		res.Pathways = append(res.Pathways, &pathwayDetails{
			PathwayTag: p.Tag,
			Screen:     screen,
			FAQ:        faq,
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

	views := []views.View{
		&views.Card{
			Title: "What's included?",
			Views: []views.View{
				&views.CheckboxTextList{
					Titles: pathway.Details.WhatIsIncluded,
				},
				&views.FilledButton{
					Title:  "Sample Treatment Plan",
					TapURL: app_url.ViewSampleTreatmentPlanAction(pathway.Tag),
				},
			},
		},
		&views.Card{
			Title: "Who will treat me?",
			Views: []views.View{
				&views.DoctorProfilePhotos{
					PhotoURLs: doctorImageURLs,
				},
				&views.BodyText{
					Text: pathway.Details.WhoWillTreatMe,
				},
			},
		},
		&views.Card{
			Title: "Is this right for me?",
			Views: []views.View{
				&views.BodyText{
					Text: pathway.Details.RightForMe,
				},
				&views.OutlinedButton{
					Title:  "Read More",
					TapURL: app_url.ViewPathwayFAQ(pathway.ID),
				},
			},
		},
		&views.Card{
			Title: "Did you know?",
			Views: []views.View{
				&views.BodyText{
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
