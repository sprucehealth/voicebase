package service

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/uv"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type PageContentBuilder interface {
	PageContentForID(ctx interface{}, r *http.Request) (interface{}, error)
}

type cityService struct {
	cityDAL    dal.CityDAL
	doctorDAL  dal.DoctorDAL
	uvService  uv.Service
	webURL     string
	contentURL string
}

func NewForCity(cityDAL dal.CityDAL, doctorDAL dal.DoctorDAL, webURL, contentURL string) PageContentBuilder {
	return &cityService{
		cityDAL:    cityDAL,
		doctorDAL:  doctorDAL,
		webURL:     webURL,
		contentURL: contentURL,
		uvService:  uv.NewService(),
	}
}

type CityPageContext struct {
	CityID string
}

const (
	longDescriptionParagraph1 = `We’ve curated a shortlist of the top dermatologists available in %s by analyzing thousands of medical interactions, patient reviews and ratings. Unlike other dermatologist directories serving %s, the results you see on this page include both dermatologists near you and dermatologists treating patients online through dermatology apps like Spruce. Getting treated by a dermatologist online means you can be treated faster (often within 24 hours) and more conveniently than the traditional in-person visit. It’s as simple as selecting your dermatologist, taking pictures and answering questions. Your doctor will review your case, diagnose and treat you within 24 hours. Any prescriptions written will be sent direct to your pharmacy. You skip the average dermatology appointment wait time of 28 days and the hassle of travelling to a dermatology office.`
	longDescriptionParagraph2 = `We’ve selected dermatologists who treat a range of general, surgical and cosmetic conditions for adult and pediatric patients including acne, anti-aging, bed bugs, cold sores, athlete's foot and ringworm, dry or itchy skin, eczema, excessive sweating, eyelash thinning, hives, insect bites or stings, lice and scabies, male hair loss, poison oak and ivy, psoriasis, shaving bumps and ingrown hair, rashes, rosacea, skin discoloration, tick bites.`
	seoDescription            = `We’ve selected top dermatologists in %s, %s including those with same-day availability. Read patient reviews, insurance accepted, practice location and contact information.`
)

func (c *cityService) PageContentForID(ctx interface{}, r *http.Request) (interface{}, error) {
	cp := ctx.(*CityPageContext)
	cityID := cp.CityID

	// check if the city exists first
	exists, err := c.cityDAL.IsCityShortListed(cityID)
	if err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, nil
	}

	city, err := c.cityDAL.City(cityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	p := conc.NewParallel()

	var spruceDoctors, localDoctors []*models.Doctor

	// local doctors
	p.Go(func() error {
		localDoctorIDs, err := c.cityDAL.LocalDoctorIDsForCity(cityID)
		if err != nil {
			return errors.Trace(err)
		}

		// get all doctors
		doctors, err := c.doctorDAL.Doctors(localDoctorIDs)
		if err != nil {
			return errors.Trace(err)
		}

		// sort the doctors by yelp review count so as to pick those
		// with most reviews
		sort.Sort(sort.Reverse(byYelpReviewCount(doctors)))

		n := 5
		if len(doctors) < n {
			n = len(doctors)
		}
		// pick the top n doctors
		localDoctors = doctors[:n]

		return nil
	})

	// spruce doctors
	p.Go(func() error {
		spruceDoctorIDs, err := c.cityDAL.SpruceDoctorIDsForCity(cityID)
		if err != nil {
			return errors.Trace(err)
		}

		spruceDoctors, err = c.pickNRandomDoctors(3, spruceDoctorIDs)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})

	// populate nearby cities by itself because it can
	// be a more expensive call
	var nearbyCitiesList []*response.LinkableItem
	p.Go(func() error {
		nearbyCities, err := c.cityDAL.NearbyCitiesForCity(city.ID, 5)
		if err != nil {
			return errors.Trace(err)
		}

		nearbyCitiesList = make([]*response.LinkableItem, len(nearbyCities))
		for i, nc := range nearbyCities {
			nearbyCitiesList[i] = &response.LinkableItem{
				Text: fmt.Sprintf("%s, %s", nc.Name, nc.StateAbbreviation),
				Link: fmt.Sprintf("%s/%s", c.webURL, nc.ID),
			}
		}
		return nil
	})

	// populate top level info
	var t *topLevelInfo
	var careRatingSection *response.SpruceScoreSection
	var topSkinConditionsList []*response.LinkableItem
	p.Go(func() error {
		t, err = c.populateTopLevelInfoForCity(city)
		if err != nil {
			return errors.Trace(err)
		}

		topSkinConditions, err := c.cityDAL.TopSkinConditionsForCity(city.ID, 5)
		if err != nil {
			return errors.Trace(err)
		}

		topSkinConditionsList = make([]*response.LinkableItem, len(topSkinConditions))
		for i, sc := range topSkinConditions {
			topSkinConditionsList[i] = &response.LinkableItem{
				Text: sc,
			}
		}

		careRatingSection, err = c.populateCareRatingForCity(city)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})

	// populate uv index separately given it requires communicating with an external service
	var uvSection *response.SpruceScoreSection
	p.Go(func() error {
		var err error
		uvSection, err = c.populateUVIndexForCity(city)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	doctors := make([]*response.Doctor, 0, len(spruceDoctors)+len(localDoctors))

	// alternate between spruce and local doctors
	var i, j int

	for i < len(spruceDoctors) || j < len(localDoctors) {
		if j < len(localDoctors) {
			doctor, err := response.TransformModel(localDoctors[j], city.ID, c.contentURL, c.webURL)
			if err != nil {
				return nil, errors.Trace(err)
			}
			doctors = append(doctors, doctor)
			j++
		}

		if i < len(spruceDoctors) {
			doctor, err := response.TransformModel(spruceDoctors[i], city.ID, c.contentURL, c.webURL)
			if err != nil {
				return nil, errors.Trace(err)
			}
			doctors = append(doctors, doctor)
			i++
		}
	}

	return &response.CityPage{
		HTMLTitle:                 fmt.Sprintf("Find Dermatologists in %s, %s | Spruce Health", city.Name, city.State),
		Title:                     t.Title,
		SEODescription:            fmt.Sprintf(seoDescription, city.Name, city.StateAbbreviation),
		Description:               t.Description,
		LongDescriptionParagraphs: t.LongDescriptionParagraphs,
		Doctors:                   doctors,
		BannerImageURL:            t.BannerImageURL,
		UVRatingSection:           uvSection,
		CareRatingSection:         careRatingSection,
		TopSkinConditionsSection: &response.DescriptionItemsSection{
			Description: fmt.Sprintf("Top 5 skin conditions people seek treatment for in %s are:", city.Name),
			Items:       topSkinConditionsList,
		},
		NearbyCitiesSection: &response.DescriptionItemsSection{
			Items: nearbyCitiesList,
		},
	}, nil
}

type topLevelInfo struct {
	Title                     string
	Description               string
	BannerImageURL            string
	LongDescriptionParagraphs []string
}

func (c *cityService) populateTopLevelInfoForCity(city *models.City) (*topLevelInfo, error) {

	bannerImageIDs, err := c.cityDAL.BannerImageIDsForCity(city.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// pick first one to make banner image picking deterministic
	bannerImageID := bannerImageIDs[0]

	bannerImageURL, err := response.URLForImageID(bannerImageID, c.contentURL)
	if err != nil {
		return nil, err
	}

	return &topLevelInfo{
		Title:       fmt.Sprintf("Top Dermatologists in %s", city.Name),
		Description: fmt.Sprintf("We’ve picked the top dermatologists accepting new patients within 15 miles of %s based on patient ratings, years of experience, and education background.", city.Name),
		LongDescriptionParagraphs: []string{
			fmt.Sprintf(longDescriptionParagraph1, city.Name, city.Name),
			longDescriptionParagraph2,
		},
		BannerImageURL: bannerImageURL,
	}, nil
}

func (c *cityService) populateCareRatingForCity(city *models.City) (*response.SpruceScoreSection, error) {
	careRating, err := c.cityDAL.CareRatingForCity(city.ID)
	if errors.Cause(err) == dal.ErrNoCareRatingFound {
		golog.Warningf("Care rating not found for city %s", city.ID)
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	ss := &response.SpruceScoreSection{
		Score:       careRating.Rating,
		Description: careRating.Title,
		Bullets:     careRating.Bullets,
	}

	return ss, nil
}

func (c *cityService) populateUVIndexForCity(city *models.City) (*response.SpruceScoreSection, error) {
	uvIndex, err := c.uvService.DailyUVIndexByCityState(city.Name, city.StateAbbreviation)
	if err != nil {
		golog.Warningf("Unable to get uvIndex for city %d: %s", city.ID, err.Error())
		return nil, nil
	}

	if uvIndex == 0 {
		return nil, nil
	}

	var bullets []string
	switch {
	case uvIndex <= 2:
		bullets = []string{
			"UV intensity is lower than national average",
			"Apply sunscreen with SPF 30+ daily",
		}
	case uvIndex >= 3 && uvIndex <= 5:
		bullets = []string{
			"UV intensity is moderate compared to national average",
			"Apply sunscreen with SPF 30+ daily and re-apply every 2 hours",
			"Avoid sunlight around midday when UV exposure is highest",
		}
	case uvIndex >= 6 && uvIndex <= 7:
		bullets = []string{
			"UV intensity is higher than national average",
			"Apply susncreen with SPF 30+ daily and re-apply every 2 hours",
			"Reduce sun exposure 10am-4pm",
		}
	case uvIndex >= 8 && uvIndex < 10:
		bullets = []string{
			"UV intensity is higher than national average",
			"Apply susncreen with SPF 30+ daily and re-apply every 2 hours",
			"Avoid sun exposure 10am-4pm",
			"Seek shade if outdoors",
		}
	case uvIndex >= 11:
		bullets = []string{
			"UV intensity is much higher than national average",
			"Apply susncreen with SPF 30+ daily and re-apply every 2 hours",
			"Avoid sun exposure 10am-4pm",
			"Seek shade if outdoors",
		}
	}

	return &response.SpruceScoreSection{
		Score:       strconv.Itoa(uvIndex),
		Description: fmt.Sprintf("Based on the UV Index for %s:", city.Name),
		Bullets:     bullets,
	}, nil
}

func (c *cityService) pickNRandomDoctors(n int, ids []string) ([]*models.Doctor, error) {
	shuffle(ids)

	if len(ids) < n {
		n = len(ids)
	}

	if n == 0 {
		return nil, nil
	}

	doctors, err := c.doctorDAL.Doctors(ids[:n])
	if err != nil {
		return nil, errors.Trace(err)
	}

	return doctors, nil
}

type byYelpReviewCount []*models.Doctor

func (c byYelpReviewCount) Len() int      { return len(c) }
func (c byYelpReviewCount) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byYelpReviewCount) Less(i, j int) bool {
	return c[i].ReviewCount < c[j].ReviewCount
}
