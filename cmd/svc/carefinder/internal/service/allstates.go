package service

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/libs/errors"
)

type allStatesService struct {
	cityDAL    dal.CityDAL
	doctorDAL  dal.DoctorDAL
	stateDAL   dal.StateDAL
	webURL     string
	contentURL string
}

const (
	allStatesLongDescriptionParagraph1 = `We’ve curated shortlists of the top dermatologists available across a range of US cities by analyzing thousands of medical interactions, patient reviews and ratings. Unlike other dermatologist directories, the doctors you see shortlisted include both dermatologists located in each city and dermatologists treating patients online through dermatology apps like Spruce. Getting treated by a dermatologist online means you can be treated faster (often within 24 hours) and more conveniently than the traditional in-person visit. It’s as simple as selecting your dermatologist, taking pictures and answering questions. Your doctor will review your case, diagnose and treat you within 24 hours. Any prescriptions written will be sent direct to your pharmacy. You skip the average dermatology appointment wait time of 28 days and the hassle of travelling to a dermatology office.`
	allStatesLongDescriptionParagraph2 = `We’ve selected dermatologists who treat a range of general, surgical and cosmetic conditions for adult and pediatric patients including acne, anti-aging, bed bugs, cold sores, athlete's foot and ringworm, dry or itchy skin, eczema, excessive sweating, eyelash thinning, hives, insect bites or stings, lice and scabies, male hair loss, poison oak and ivy, psoriasis, shaving bumps and ingrown hair, rashes, rosacea, skin discoloration, tick bites.`
	allStatesSEODescription            = `We’ve selected top dermatologists for a range of cities in %s based on patient reviews and medical peer referrals, including those with same-day availability. Read patient reviews, insurance accepted, practice location and contact information.`
	topBannerImageID                   = `s3://us-east-1/carefinder/bannerimages/manhattan-ny-2`
)

func NewForAllStates(cityDAL dal.CityDAL, doctorDAL dal.DoctorDAL, stateDAL dal.StateDAL, webURL, contentURL string) PageContentBuilder {
	return &allStatesService{
		cityDAL:    cityDAL,
		doctorDAL:  doctorDAL,
		stateDAL:   stateDAL,
		webURL:     webURL,
		contentURL: contentURL,
	}
}

func (a *allStatesService) PageContentForID(ctx interface{}, r *http.Request) (interface{}, error) {
	states, err := a.stateDAL.StateShortList()
	if err != nil {
		return nil, errors.Trace(err)
	}

	stateAbbreviations := make([]string, len(states))
	for i, s := range states {
		stateAbbreviations[i] = s.Abbreviation
	}

	bannerImageIDsByState, err := a.stateDAL.BannerImageIDsForStates(stateAbbreviations)
	if err != nil {
		return nil, errors.Trace(err)
	}

	responseStates := make([]*response.State, len(states))
	for i, s := range states {
		bannerImageID, err := response.URLForImageID(bannerImageIDsByState[s.Abbreviation][0], a.contentURL)
		if err != nil {
			return nil, errors.Trace(err)
		}

		responseStates[i] = &response.State{
			Name:     s.FullName,
			Link:     response.StatePageURL(s.Key, a.webURL),
			ImageURL: bannerImageID,
		}
	}

	topBannerImageURL, err := response.URLForImageID(topBannerImageID, a.contentURL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &response.AllStatesPage{
		IsMobile:       isMobile(r),
		HTMLTitle:      "Find a Dermatologist | Spruce Health",
		Title:          "Find a Dermatologist",
		BannerImageURL: topBannerImageURL,
		Description:    "We've hand picked some of the top dermatologists for US cities based on patient ratings, years of experience, and educational background.",
		SEODescription: "We've hand picked some of the top dermatologists for US cities based on patient ratings, years of experience, and educational background.",
		States:         responseStates,
		LongDescriptionParagraphs: []string{
			fmt.Sprintf(allStatesLongDescriptionParagraph1),
			fmt.Sprintf(allStatesLongDescriptionParagraph2),
		},
	}, nil
}
