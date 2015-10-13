package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type doctorPageHandler struct {
	refTemplate       *template.Template
	staticResourceURL string
}

type reviewItem struct {
	Text           string
	Source         string
	SourceImageURL string
	YelpPageURL    string
	Author         string
	Date           string
	RatingURL      string
}

type reviewsSection struct {
	Reviews             []reviewItem
	ShowMoreReviewsLink bool
	MoreReviewsURL      string
	Title               string
	AverageRatingURL    string
	SourceImageURL      string
}

type titleDescriptionItem struct {
	Title       string
	Description string
}

type officeHoursItem struct {
	Day   string
	Hours string
}

type imageTextItem struct {
	ImageURL string
	Text     string
}

type officeInfo struct {
	AddressLine1   string
	AddressLine2   string
	GoogleMapsLink string
	OfficeHours    []officeHoursItem
}

type doctorInfo struct {
	LongDisplayName           string
	ProfileImageURL           string
	BannerImageURL            string
	Phone                     string
	PhoneLink                 string
	IsSpruceDoctor            bool
	ReviewsSection            reviewsSection
	Specialties               []string
	Qualifications            []titleDescriptionItem
	PhysicalOfficeInformation *officeInfo `json:",omitempty"`
	StateCoverageText         string
	AcceptedInsurance         []string
	ConditionsTreated         []string
	AvailabilityItems         []imageTextItem
	OfficeSectionTitle        string
}

func staticDoctorContent(staticResourceURL string) map[string]doctorInfo {
	yelpImageURL := fmt.Sprintf("%s/img/yelp.svg", staticResourceURL)

	return map[string]doctorInfo{
		"jason-fung": doctorInfo{
			LongDisplayName:    "Dr. Jason Fung",
			ProfileImageURL:    fmt.Sprintf("%s/img/fung.jpg", staticResourceURL),
			BannerImageURL:     "http://test-spruce-storage.s3.amazonaws.com/sf.jpg",
			IsSpruceDoctor:     true,
			OfficeSectionTitle: "See Dr. Fung Online",
			ReviewsSection: reviewsSection{
				Title:               "24 REVIEWS ON YELP",
				AverageRatingURL:    "http://s3-media3.fl.yelpcdn.com/assets/2/www/img/22affc4e6c38/ico/stars/v1/stars_large_5.png",
				ShowMoreReviewsLink: true,
				Reviews: []reviewItem{
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
				},
				SourceImageURL: yelpImageURL,
				MoreReviewsURL: "http://m.yelp.com/biz/jason-fung-md-oakland",
			},
			Specialties: []string{
				"Psoriasis",
				"Eczema",
				"Bed Bugs",
			},
			Qualifications: []titleDescriptionItem{
				{
					Title:       "MEDICAL SCHOOL",
					Description: "University of Rochester School of Medicine and Dentistry",
				},
				{
					Title:       "RESIDENCY",
					Description: "University of Michigan, Ann Arbor",
				},
				{
					Title:       "CLINICAL EXPERIENCE",
					Description: "15 years",
				},
			},
			StateCoverageText: "Dr. Fung treats patients in California, Florida, and Ohio online through Spruce",
			ConditionsTreated: []string{"Acne", "Anti-Aging", "Bed Bugs", "Cold Sores", "Athlete's Foot & Ringworm", "Dry or Itchy Skin", "Eczema", "Excessive Sweating", "Hives", "Insect Bites or Stings", "Lice & Scabies", "Male Hair Loss", "Poison Oak & Ivy", "Psoriasis", "Shaving Bumps & Ingrown Hair", "Rashes", "Roscaea", "Skin Discoloration", "Tick Bites"},
			AvailabilityItems: []imageTextItem{
				{
					ImageURL: fmt.Sprintf("%s/new_patients.png", staticResourceURL),
					Text:     "Accepting new patients with Spruce",
				},
				{
					ImageURL: fmt.Sprintf("%s/img/24_hours.png", staticResourceURL),
					Text:     "Typically responds in 24 hours",
				},
			},
		},
		"lavanya-krishnan": doctorInfo{
			LongDisplayName:    "Dr. Lavanya Krishnan",
			ProfileImageURL:    fmt.Sprintf("%s/img/lavanya.jpg", staticResourceURL),
			BannerImageURL:     "http://test-spruce-storage.s3.amazonaws.com/sf.jpg",
			IsSpruceDoctor:     false,
			Phone:              "206-877-3590",
			PhoneLink:          "tel:206-877-3590",
			OfficeSectionTitle: "See Dr. Krishnan In Office",
			ReviewsSection: reviewsSection{
				Title:               "24 REVIEWS ON YELP",
				AverageRatingURL:    "http://s3-media3.fl.yelpcdn.com/assets/2/www/img/22affc4e6c38/ico/stars/v1/stars_large_5.png",
				ShowMoreReviewsLink: true,
				Reviews: []reviewItem{
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
					{
						Text:           "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Dugis vel ipsum quis diam tincidvunt lobortis. Pellentesque egestas lectus sapien.",
						Source:         "yelp",
						SourceImageURL: yelpImageURL,
						YelpPageURL:    "http://m.yelp.com/biz/jason-fung-md-oakland",
						Author:         "Kunal Jham",
						Date:           "02/12/15",
						RatingURL:      "http://s3-media4.fl.yelpcdn.com/assets/2/www/img/9f83790ff7f6/ico/stars/v1/stars_large_4_half.png",
					},
				},
				SourceImageURL: yelpImageURL,
				MoreReviewsURL: "http://m.yelp.com/biz/jason-fung-md-oakland",
			},
			Specialties: []string{
				"Psoriasis",
				"Eczema",
				"Bed Bugs",
			},
			Qualifications: []titleDescriptionItem{
				{
					Title:       "MEDICAL SCHOOL",
					Description: "University of Rochester School of Medicine and Dentistry",
				},
				{
					Title:       "RESIDENCY",
					Description: "University of Michigan, Ann Arbor",
				},
				{
					Title:       "CLINICAL EXPERIENCE",
					Description: "15 years",
				},
			},
			PhysicalOfficeInformation: &officeInfo{
				AddressLine1:   "1510 Eddy Street",
				AddressLine2:   "San Francisco, CA 94115",
				GoogleMapsLink: "http://maps.google.com/?q=1510 Eddy Street San Francisco CA 94115",
				OfficeHours: []officeHoursItem{
					{
						Day:   "Mon",
						Hours: "8:00am - 5:00pm",
					},
					{
						Day:   "Tue",
						Hours: "8:00am - 5:00pm",
					},
					{
						Day:   "Wed",
						Hours: "8:00am - 5:00pm",
					},
					{
						Day:   "Thu",
						Hours: "8:00am - 5:00pm",
					},
					{
						Day:   "Fri",
						Hours: "8:00am - 5:00pm",
					},
					{
						Day:   "Sat",
						Hours: "Closed",
					},
					{
						Day:   "Sun",
						Hours: "Closed",
					},
				},
			},
			AcceptedInsurance: []string{"Aetna", "Blue Cross Blue Shield of California", "Blue Cross Blue Shield of New York", "Total Health Plan", "QualCore", "MagnaCare", "Kaiser Permanente", "HealthLink", "First Choice", "Cofinity", "BridgeSpan", "Assurant"},
			AvailabilityItems: []imageTextItem{
				{
					ImageURL: fmt.Sprintf("%s/img/phone.png", staticResourceURL),
					Text:     "Contact Dr. Derm’s office for next available appointment",
				},
			},
		},
	}
}

func NewDoctorPageHandler(staticResourceURL string, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return &doctorPageHandler{
		refTemplate:       templateLoader.MustLoadTemplate("doctorpage.html", "base.html", nil),
		staticResourceURL: staticResourceURL,
	}
}

func (d *doctorPageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(ctx)
	doctors := staticDoctorContent(d.staticResourceURL)
	www.TemplateResponse(w, http.StatusOK, d.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Doctor Page"),
		SubContext:  doctors[vars["doctor"]],
	})
}
