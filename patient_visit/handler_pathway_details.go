package patient_visit

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/careprovider"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
)

type pathwayDetailsHandler struct {
	dataAPI              api.DataAPI
	apiDomain            string
	launchPromoStartDate *time.Time
	cfgStore             cfg.Store
}

type pathwayDetailsResponse struct {
	Pathways []*pathwayDetails `json:"pathway_details_screens"`
}

type pathwayDetails struct {
	PathwayTag      string                   `json:"pathway_id"`
	Screen          *pathwayDetailsScreen    `json:"screen"`
	FAQ             *pathwayFAQ              `json:"faq,omitempty"`
	AgeRestrictions []*pathwayAgeRestriction `json:"age_restrictions,omitempty"`
}

type pathwayAgeRestriction struct {
	MaxAgeOfRange *int          `json:"max_age_of_range"`
	VisitAllowed  bool          `json:"visit_allowed"`
	Alert         *pathwayAlert `json:"alert,omitempty"`
}

type pathwayAlert struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Message     string `json:"message"`
	ButtonTitle string `json:"button_title"`
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
	ContentSubtext string `json:"content_subtext,omitempty"`
	PhotoURL       string `json:"photo_url,omitempty"`
}

type pathwayFAQ struct {
	Title string       `json:"title"`
	Views []views.View `json:"views"`
}

// NewPathwayDetailsHandler returns an initialized instance of pathwayDetailsHandler
func NewPathwayDetailsHandler(dataAPI api.DataAPI, apiDomain string, cfgStore cfg.Store) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&pathwayDetailsHandler{
			dataAPI:   dataAPI,
			apiDomain: apiDomain,
			cfgStore:  cfgStore,
		}),
		httputil.Get)
}

func (h *pathwayDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathwayTags := strings.Split(r.FormValue("pathway_id"), ",")
	if len(pathwayTags) == 0 {
		// empty response for an empty request (eye for an eye)
		httputil.JSONResponse(w, http.StatusOK, &pathwayDetailsResponse{
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
	activeCases := make(map[string]*common.PatientCase)
	var activeCaseIDs []int64

	ctx := apiservice.GetContext(r)
	if ctx.AccountID != 0 && ctx.Role == api.RolePatient {
		patientID, err = h.dataAPI.GetPatientIDFromAccountID(ctx.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		cases, err := h.dataAPI.GetCasesForPatient(patientID, []string{common.PCStatusActive.String(), common.PCStatusOpen.String()})
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		activeCaseIDs = make([]int64, len(cases))
		for i, pc := range cases {
			activeCases[pc.PathwayTag] = pc
			activeCaseIDs[i] = pc.ID.Int64()
		}
	}

	var fetchedCareTeams bool
	var careTeams map[int64]*common.PatientCareTeam

	res := &pathwayDetailsResponse{}
	for _, p := range pathways {
		sku, err := h.dataAPI.SKUForPathway(p.Tag, common.SCVisit)
		if err != nil {
			golog.Errorf("Failed to lookup sku for pathway %s: %s", p.Name, err)
		}

		cost, err := h.dataAPI.GetActiveItemCost(sku.Type)
		if err != nil {
			golog.Errorf("Failed to get cost for pathway %d '%s': %s", p.ID, p.Name, err)
		}

		var screen *pathwayDetailsScreen
		var faq *pathwayFAQ
		var ageRestrictions []*pathwayAgeRestriction
		if pcase := activeCases[p.Tag]; pcase != nil {
			switch {
			case pcase.Status == common.PCStatusOpen:
				screen = openCaseScreen(pcase, p, h.apiDomain)
			case !pcase.Claimed:
				screen = pendingReviewCaseScreen(pcase, p)
			default:
				if !fetchedCareTeams {
					careTeams, err = h.dataAPI.CaseCareTeams(activeCaseIDs)
					if err != nil {
						apiservice.WriteError(err, w, r)
						return
					}
					fetchedCareTeams = true
				}
				screen = activeCaseScreen(careTeams[pcase.ID.Int64()], pcase.ID.Int64(), p, h.apiDomain)
			}
		} else if p.Details == nil {
			golog.Errorf("Details missing for pathway %d '%s'", p.ID, p.Name)
			screen = detailsMissingScreen(p)
		} else {
			imageURLs, err := careprovider.RandomDoctorURLs(4, h.dataAPI, h.apiDomain, nil)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			screen, err = merchandisingScreen(p, imageURLs, cost, h.apiDomain, patientID, h.launchPromoStartDate, h.dataAPI, h.cfgStore)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
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

			if len(p.Details.AgeRestrictions) != 0 {
				ageRestrictions = make([]*pathwayAgeRestriction, len(p.Details.AgeRestrictions))
				for i, ar := range p.Details.AgeRestrictions {
					var alert *pathwayAlert
					if ar.Alert != nil {
						alert = &pathwayAlert{
							Type:        ar.Alert.Type,
							Title:       ar.Alert.Title,
							Message:     ar.Alert.Message,
							ButtonTitle: ar.Alert.ButtonTitle,
						}
					}
					ageRestrictions[i] = &pathwayAgeRestriction{
						MaxAgeOfRange: ar.MaxAgeOfRange,
						VisitAllowed:  ar.VisitAllowed,
						Alert:         alert,
					}
				}
			}
		}
		if err := views.Validate(screen.Views, "pathway_details"); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		res.Pathways = append(res.Pathways, &pathwayDetails{
			PathwayTag:      p.Tag,
			Screen:          screen,
			FAQ:             faq,
			AgeRestrictions: ageRestrictions,
		})
	}

	if res.Pathways == nil {
		// Return an empty list instead of null if no pathways found
		res.Pathways = []*pathwayDetails{}
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}

func merchandisingScreen(pathway *common.Pathway, doctorImageURLs []string, itemCost *common.ItemCost, apiDomain string, patientID int64, launchPromoStartDate *time.Time, dataAPI api.DataAPI, cfgStore cfg.Store) (*pathwayDetailsScreen, error) {
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

	cardViews := []views.View{
		&views.Card{
			Title: "What's included?",
			Views: []views.View{
				&views.CheckboxTextList{
					Titles: pathway.Details.WhatIsIncluded,
				},
				&views.OutlinedButton{
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
					TapURL: app_url.ViewPathwayFAQ(pathway.Tag),
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

	if cfgStore.Snapshot().Bool(cost.GlobalFirstVisitFreeEnabled.Name) {
		card, err := limitedTimeFirstVisitFreeCard(patientID, dataAPI)
		if err != nil {
			return nil, err
		}

		if card != nil {
			newCardViews := []views.View{card}
			cardViews = append(newCardViews, cardViews...)
		}
	}

	var headerButtonTitle string
	if itemCost != nil {
		headerButtonTitle = itemCost.TotalCost().String()
	}

	return &pathwayDetailsScreen{
		Type:  "merchandising",
		Title: fmt.Sprintf("%s Visit", pathway.Name),
		Views: cardViews,
		RightHeaderButtonTitle: headerButtonTitle,
		BottomButtonTitle:      "Continue",
		BottomButtonTapURL:     app_url.ViewChooseDoctorScreen(),
	}, nil
}

// limitedTimeFirstVisitFreeCard returns the card for the first visit free promotion. If the patient is not eligible it will return nil
func limitedTimeFirstVisitFreeCard(patientID int64, dataAPI api.DataAPI) (views.View, error) {
	limitedTimeOfferCard := &views.Card{
		Title: "Limited time offer",
		Views: []views.View{
			&views.BodyText{
				Text: "Your first visit on Spruce is free.",
			},
		},
	}

	// always add limited time offer card for unauthenticated case.
	if patientID == 0 {
		return limitedTimeOfferCard, nil
	}

	// check if the patient has any submitted visits
	visits, err := dataAPI.VisitsSubmittedForPatientSince(patientID, time.Unix(1, 0))
	if err != nil {
		return nil, err
	}

	// don't return card if the user is logged in
	// and has already submitted a visit since launch that was free
	if len(visits) > 0 {
		return nil, nil
	}

	return limitedTimeOfferCard, nil
}

func openCaseScreen(pcase *common.PatientCase, pathway *common.Pathway, apiDomain string) *pathwayDetailsScreen {
	name := pathway.Name
	lowerName := strings.ToLower(name)
	article := "a"
	switch name[0] {
	case 'a', 'e', 'i', 'o', 'u':
		article = "an"
	}
	return &pathwayDetailsScreen{
		Type:               "generic_message",
		Title:              name,
		BottomButtonTitle:  "Okay",
		BottomButtonTapURL: app_url.ViewHomeAction(),
		ContentText:        fmt.Sprintf("You have %s %s visit in progress.", article, lowerName),
		ContentSubtext:     "Complete your visit and get a personalized treatment plan from your doctor.",
		PhotoURL:           app_url.IconWhiteCase.String(),
	}
}

func pendingReviewCaseScreen(pcase *common.PatientCase, pathway *common.Pathway) *pathwayDetailsScreen {
	return &pathwayDetailsScreen{
		Type:               "generic_message",
		Title:              pathway.Name,
		BottomButtonTitle:  "Okay",
		BottomButtonTapURL: app_url.ViewHomeAction(),
		ContentText:        fmt.Sprintf("You have an existing %s visit that is pending review.", strings.ToLower(pathway.Name)),
		ContentSubtext:     "Message your care team with any questions you may have.",
		PhotoURL:           app_url.IconWhiteCase.String(),
	}
}

func activeCaseScreen(careTeam *common.PatientCareTeam, caseID int64, pathway *common.Pathway, apiDomain string) *pathwayDetailsScreen {
	var doctorName string
	var doctorThumbnailURL string
	if careTeam != nil {
		for _, a := range careTeam.Assignments {
			if a.ProviderRole == api.RoleDoctor {
				doctorName = a.ShortDisplayName
				doctorThumbnailURL = app_url.ThumbnailURL(apiDomain, a.ProviderRole, a.ProviderID)
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
