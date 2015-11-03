package response

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

type CityPage struct {
	HTMLTitle                 string
	Title                     string
	Description               string
	SEODescription            string
	LongDescriptionParagraphs []string
	BannerImageURL            string
	Doctors                   []*Doctor
	UVRatingSection           *SpruceScoreSection
	CareRatingSection         *SpruceScoreSection
	TopSkinConditionsSection  *DescriptionItemsSection
	NearbyCitiesSection       *DescriptionItemsSection
}

type City struct {
	Name  string
	State string
	Link  string
}

type State struct {
	Name     string
	Link     string
	ImageURL string
}

type StatePage struct {
	IsMobile                  bool
	HTMLTitle                 string
	Title                     string
	Description               string
	FeaturedCitiesAboveFold   []*City
	FeaturedCitiesBelowFold   []*City
	SEODescription            string
	LongDescriptionParagraphs []string
	BannerImageURL            string
	FeaturedDoctors           []*Doctor
}

type AllStatesPage struct {
	IsMobile                  bool
	HTMLTitle                 string
	Title                     string
	Description               string
	BannerImageURL            string
	SEODescription            string
	States                    []*State
	LongDescriptionParagraphs []string
}

type Doctor struct {
	IsSpruceDoctor      bool
	Description         string
	LongDisplayName     string
	ShortDisplayName    string
	ProfileImageURL     string
	Experience          string
	Specialties         []string `json:",omitempty"`
	InsuranceAccepted   []string
	ProfileURL          string
	StartOnlineVisitURL string
	StarRatingImg       string
	ReviewCount         int
	AverageRating       float64
}

type SpruceScoreSection struct {
	Score       string
	Description string
	Bullets     []string
}

type DescriptionItemsSection struct {
	Description string
	Items       []*LinkableItem
}

type LinkableItem struct {
	Text string
	Link string
}

type TitleDescriptionItem struct {
	Title       string
	Description string
}

type ImageTextItem struct {
	ImageName string
	Text      string
}

type OfficeHoursItem struct {
	Day   string
	Hours string
}

type Address struct {
	AddressLine1         string
	AddressLine2         string
	City                 string
	State                string
	Zipcode              string
	Latitude             float64
	Phone                string
	Longitude            float64
	GoogleMapsLink       string
	GoogleMapsImageURL   string
	OfficeHours          []*OfficeHoursItem
	CondensedOfficeHours string
}

type Review struct {
	Text            string
	Source          string
	SourceImageName string
	YelpPageURL     string
	Author          string
	Date            string
	RatingImageURL  string
	Rating          float64
	Citation        string
}

type ReviewsSection struct {
	Reviews               []*Review
	MoreReviewsURL        string
	Title                 string
	AverageRatingImageURL string
	SourceImageName       string
	AverageRating         float64
	ReviewCount           int
}

type Container struct {
	Items []string
}

type BreadcrumbItem struct {
	Label string
	Link  string
}

type BreadcrumbList struct {
	Items []*BreadcrumbItem
}

type DoctorPage struct {
	HTMLTitle                 string
	LongDisplayName           string
	ProfileImageURL           string
	ProfileURL                string
	Description               string
	SEODescription            string
	BannerImageURL            string
	StartOnlineVisitURL       string
	PhoneLink                 string
	IsSpruceDoctor            bool
	ReviewsSection            *ReviewsSection
	Specialties               []string
	Qualifications            []*TitleDescriptionItem
	PhysicalOfficeInformation *Address
	StateCoverageText         string
	AcceptedInsurance         []*Container
	ConditionsTreated         []*Container
	AvailabilityItems         []*ImageTextItem
	OfficeSectionTitle        string
	SpruceDoctors             []*Doctor
	Breadcrumb                *BreadcrumbList
	OtherBreadcrumbs          []*BreadcrumbList
	IsMobile                  bool
}

type StartOnlineVisitPage struct {
	DoctorID               string
	HTMLTitle              string
	DoctorShortDisplayName string
	ReferralLink           string
	ProfileImageURL        string
	IsMobile               bool
}

func TransformModel(doctor *models.Doctor, cityID, contentURL, webURL string) (*Doctor, error) {

	var experience string
	if doctor.GraduationYear != "" {
		graduationYearInt, err := strconv.Atoi(doctor.GraduationYear)
		if err != nil {
			return nil, errors.Trace(err)
		}

		difference := time.Now().Year() - graduationYearInt
		switch {
		case difference < 5:
			experience = "< 5 years"
		case difference >= 5 && difference < 10:
			experience = "5-10 years"
		case difference >= 10:
			experience = "10+ years"

		}
	}

	var profileImageURL string
	if doctor.ProfileImageID != "" {
		var err error
		profileImageURL, err = URLForImageID(doctor.ProfileImageID, contentURL)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	var profileURL string
	if cityID != "" {
		profileURL = fmt.Sprintf("%s/%s?city_id=%s", webURL, doctor.ID, cityID)
	} else {
		profileURL = fmt.Sprintf("%s/%s", webURL, doctor.ID)
	}

	return &Doctor{
		IsSpruceDoctor:      doctor.IsSpruceDoctor,
		Description:         doctor.Description,
		LongDisplayName:     fmt.Sprintf("Dr. %s %s", doctor.FirstName, doctor.LastName),
		ShortDisplayName:    fmt.Sprintf("Dr. %s", doctor.LastName),
		ProfileImageURL:     profileImageURL,
		Experience:          experience,
		Specialties:         doctor.Specialties,
		InsuranceAccepted:   doctor.InsurancesAccepted,
		ProfileURL:          profileURL,
		StartOnlineVisitURL: fmt.Sprintf("%s/%s/start-online-visit", webURL, doctor.ID),
		StarRatingImg:       determineImageNameForRating(roundToClosestHalve(doctor.AverageRating)),
		AverageRating:       doctor.AverageRating,
		ReviewCount:         doctor.ReviewCount,
	}, nil
}
