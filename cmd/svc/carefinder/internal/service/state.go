package service

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
)

type stateService struct {
	cityDAL    dal.CityDAL
	doctorDAL  dal.DoctorDAL
	stateDAL   dal.StateDAL
	webURL     string
	contentURL string
}

const (
	stateLongDescriptionParagraph1 = `We’ve curated shortlists of the top dermatologists available in %s by analyzing thousands of medical interactions, patient reviews and ratings. Unlike other dermatologist directories, the doctors you see shortlisted include both dermatologists located in your city and dermatologists treating patients online through dermatology apps like Spruce. Getting treated by a dermatologist online means you can be treated faster (often within 24 hours) and more conveniently than the traditional in-person visit. It’s as simple as selecting your dermatologist, taking pictures and answering questions. Your doctor will review your case, diagnose and treat you within 24 hours. Any prescriptions written will be sent direct to your pharmacy. You skip the average dermatology appointment wait time of 28 days and the hassle of travelling to a dermatology office.`
	stateLongDescriptionParagraph2 = `We’ve selected dermatologists who treat a range of general, surgical and cosmetic conditions for adult and pediatric patients including acne, anti-aging, bed bugs, cold sores, athlete's foot and ringworm, dry or itchy skin, eczema, excessive sweating, eyelash thinning, hives, insect bites or stings, lice and scabies, male hair loss, poison oak and ivy, psoriasis, shaving bumps and ingrown hair, rashes, rosacea, skin discoloration, tick bites.`
	stateSEODescription            = `We’ve selected top dermatologists for a range of cities in %s based on patient reviews and medical peer referrals, including those with same-day availability. Read patient reviews, insurance accepted, practice location and contact information.`
)

func NewForState(cityDAL dal.CityDAL, doctorDAL dal.DoctorDAL, stateDAL dal.StateDAL, webURL, contentURL string) PageContentBuilder {
	return &stateService{
		cityDAL:    cityDAL,
		doctorDAL:  doctorDAL,
		stateDAL:   stateDAL,
		webURL:     webURL,
		contentURL: contentURL,
	}
}

type StatePageContext struct {
	StateKey string
}

func (s *stateService) PageContentForID(ctx interface{}, r *http.Request) (interface{}, error) {
	spc := ctx.(*StatePageContext)
	stateKey := spc.StateKey

	exists, err := s.stateDAL.IsStateShortListed(stateKey)
	if err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, nil
	}

	state, err := s.stateDAL.State(stateKey)
	if err != nil {
		return nil, errors.Trace(err)
	}

	p := conc.NewParallel()

	// get a list of featured cities
	var featuredCitiesAboveFold []*response.City
	var featuredCitiesBelowFold []*response.City
	p.Go(func() error {
		cities, err := s.cityDAL.CitiesForState(state.Key)
		if err != nil {
			return errors.Trace(err)
		}

		sort.Sort(byFeatured(cities))

		n := 8
		if len(cities) < n {
			n = len(cities)
		}

		featuredCitiesAboveFold = make([]*response.City, n)
		for i := 0; i < n; i++ {
			featuredCitiesAboveFold[i] = &response.City{
				Name:  cities[i].Name,
				State: cities[i].StateAbbreviation,
				Link:  response.CityPageURL(cities[i], s.webURL),
			}
		}

		remaining := len(cities) - n
		if remaining <= 0 {
			return nil
		}

		featuredCitiesBelowFold = make([]*response.City, 0, remaining)
		for j := n; j < len(cities); j++ {
			featuredCitiesBelowFold = append(featuredCitiesBelowFold, &response.City{
				Name:  cities[j].Name,
				State: cities[j].StateAbbreviation,
				Link:  response.CityPageURL(cities[j], s.webURL),
			})
		}

		return nil
	})

	// get a list of spruce doctors
	var spruceDoctors []*response.Doctor
	p.Go(func() error {
		spruceDoctorIDs, err := s.stateDAL.SpruceDoctorIDsForState(state.Abbreviation)
		if err != nil {
			return errors.Trace(err)
		}

		// get all of them and then sort by review count
		doctors, err := s.doctorDAL.Doctors(spruceDoctorIDs)
		if err != nil {
			return errors.Trace(err)
		}

		sort.Sort(sort.Reverse(byReviewCount(doctors)))

		n := 3
		if len(doctors) < n {
			n = len(doctors)
		}

		spruceDoctors = make([]*response.Doctor, n)
		for i, item := range doctors[:n] {
			spruceDoctors[i], err = response.TransformModel(item, "", s.contentURL, s.webURL)
			if err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})

	var bannerImageURL string
	p.Go(func() error {
		imageIDs, err := s.stateDAL.BannerImageIDsForState(state.Abbreviation)
		if err != nil {
			return errors.Trace(err)
		}

		bannerImageURL, err = response.URLForImageID(imageIDs[0], s.contentURL)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	return &response.StatePage{
		IsMobile:                isMobile(r),
		HTMLTitle:               fmt.Sprintf("Find Dermatologists in %s", state.FullName),
		Title:                   fmt.Sprintf("Top Dermatologists in %s", state.FullName),
		SEODescription:          fmt.Sprintf(stateSEODescription, state.FullName),
		BannerImageURL:          bannerImageURL,
		Description:             fmt.Sprintf("We've picked the top dermatologists accepting new patients in %s for a range of cities based on patient ratings, years of experience and education background.", state.FullName),
		FeaturedCitiesAboveFold: featuredCitiesAboveFold,
		FeaturedCitiesBelowFold: featuredCitiesBelowFold,
		FeaturedDoctors:         spruceDoctors,
		LongDescriptionParagraphs: []string{
			fmt.Sprintf(stateLongDescriptionParagraph1, state.FullName),
			fmt.Sprintf(stateLongDescriptionParagraph2),
		},
	}, nil
}

//byFeatured first sorts by surfacing cities that are featured, and then lexicographically
//sorts the cities that are not featured.
type byFeatured []*models.City

func (c byFeatured) Len() int      { return len(c) }
func (c byFeatured) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byFeatured) Less(i, j int) bool {
	if c[i].Featured {
		return true
	}
	if c[j].Featured {
		return false
	}
	if strings.Compare(c[i].Name, c[j].Name) == -1 {
		return true
	}
	return false
}
