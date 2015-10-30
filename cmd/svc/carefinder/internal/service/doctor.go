package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/googlemaps"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/yelp"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	conditionsTreatedOnline = []string{
		"Acne",
		"Anti-Aging",
		"Athletes Foot or Ringworm",
		"Bed Bugs",
		"Cold Sores",
		"Dry or Itchy Skin",
		"Eczema",
		"Excessive Sweating",
		"Hives",
		"Lice or Scabies",
		"Male Hair Loss",
		"Nail Problems & Injuries",
		"Poison Oak or Ivy",
		"Psoriasis",
		"Rash",
		"Ingrown Hair",
		"Skin Discoloration",
		"Rosacea",
		"Dandruff",
		"Eyelash Thinning",
	}
)

type doctorService struct {
	cityDAL                 dal.CityDAL
	doctorDAL               dal.DoctorDAL
	yelpClient              yelp.Client
	webURL                  string
	contentURL              string
	staticResourceURL       string
	staticMapsKey           string
	staticMapsURLSigningKey string
}

func NewForDoctor(cityDAL dal.CityDAL, doctorDAL dal.DoctorDAL, yelpClient yelp.Client, webURL, contentURL, staticResourceURL, staticMapsKey, staticMapsURLSigningKey string) PageContentBuilder {
	return &doctorService{
		cityDAL:                 cityDAL,
		doctorDAL:               doctorDAL,
		webURL:                  webURL,
		contentURL:              contentURL,
		staticResourceURL:       staticResourceURL,
		yelpClient:              yelpClient,
		staticMapsKey:           staticMapsKey,
		staticMapsURLSigningKey: staticMapsURLSigningKey,
	}
}

type DoctorPageContext struct {
	DoctorID string
	CityID   string
}

func (d *doctorService) PageContentForID(ctx interface{}, r *http.Request) (interface{}, error) {
	dpc := ctx.(*DoctorPageContext)
	doctorID := dpc.DoctorID
	cityID := dpc.CityID
	citySpecified := cityID != ""

	// check if the doctor is shortlisted
	exists, err := d.doctorDAL.IsDoctorShortListed(doctorID)
	if err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, nil
	}

	p := conc.NewParallel()

	// get doctor information
	var doctor *models.Doctor
	var doctorResponse *response.Doctor
	p.Go(func() error {
		var err error
		doctor, err = d.doctorDAL.Doctor(doctorID)
		if err != nil {
			return errors.Trace(err)
		}

		// transform the doctor model as its easier to work with
		doctorResponse, err = response.TransformModel(doctor, "", d.contentURL, d.webURL)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})

	p.Go(func() error {
		// ensure that the doctor is indeed short listed
		// in the city if local doctor
		// and in the state the city belongs to if spruce doctor
		if citySpecified {
			if shortListedForCity, err := d.cityDAL.IsDoctorShortListedForCity(doctorID, cityID); err != nil {
				return errors.Trace(err)
			} else if !shortListedForCity {
				citySpecified = false
			}
		}

		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	var city *models.City
	if citySpecified {
		city, err = d.cityDAL.City(cityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else if !doctor.IsSpruceDoctor {
		city, err = d.doctorDAL.ShortListedCityClosestToPracticeLocation(doctorID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	p = conc.NewParallel()

	// get spruce doctors if local doctor
	var spruceDoctors []*response.Doctor
	if !doctor.IsSpruceDoctor {
		p.Go(func() error {

			spruceDoctorIDs, err := d.cityDAL.SpruceDoctorIDsForCity(city.ID)
			if err != nil {
				return errors.Trace(err)
			}

			if len(spruceDoctorIDs) == 0 {
				return nil
			}

			shuffle(spruceDoctorIDs)

			n := 3
			if len(spruceDoctorIDs) < n {
				n = len(spruceDoctorIDs)
			}

			doctors, err := d.doctorDAL.Doctors(spruceDoctorIDs[:n])
			if err != nil {
				return errors.Trace(err)
			}

			spruceDoctors = make([]*response.Doctor, len(doctors))
			for i, sd := range doctors {
				spruceDoctors[i], err = response.TransformModel(sd, city.ID, d.contentURL, d.webURL)
				if err != nil {
					return errors.Trace(err)
				}
			}

			return nil
		})
	}

	// build reviews section for doctor depending on whether we are dealing with local or
	// spruce doctor
	var reviewSection *response.ReviewsSection
	p.Go(func() error {
		var err error
		reviewSection, err = d.buildReviewsSection(doctor, doctorResponse)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})

	// build out state coverage text
	var stateCoverageText string
	if doctor.IsSpruceDoctor {
		p.Go(func() error {
			// get the states the doctor is registered in
			statesCovered, err := d.doctorDAL.StateCoverageForSpruceDoctor(doctorID)
			if err != nil {
				return errors.Trace(err)
			}
			stateCoverageText = buildStateCoverageText(doctorResponse, statesCovered)
			return nil
		})
	}

	// build out banner image and breadcrumbs
	var bc response.BreadcrumbList
	var otherBCs []*response.BreadcrumbList
	var bannerImageURL string
	p.Go(func() error {

		switch {
		case citySpecified:
			bc.Items = append(bc.Items, &response.BreadcrumbItem{
				Label: city.State,
			}, &response.BreadcrumbItem{
				Label: city.Name,
				Link:  fmt.Sprintf("%s/%s", d.webURL, city.ID),
			})

			imageIDs, err := d.cityDAL.BannerImageIDsForCity(city.ID)
			if err != nil {
				return errors.Trace(err)
			}

			bannerImageURL, err = response.URLForImageID(imageIDs[0], d.contentURL)
			if err != nil {
				return errors.Trace(err)
			}

		case doctor.IsSpruceDoctor:
			// if city not specified, and we are working with spruce doctor,
			// show the banner image of a state
			// where the doctor is registered, and include breadcrumbs
			// at the state level

			states, err := d.doctorDAL.ShortListedStatesForSpruceDoctor(doctorID)
			if err != nil {
				return errors.Trace(err)
			}
			imageIDs, err := d.cityDAL.BannerImageIDsForState(states[0].Abbreviation)
			if err != nil {
				return errors.Trace(err)
			}

			bannerImageURL, err = response.URLForImageID(imageIDs[0], d.contentURL)
			if err != nil {
				return errors.Trace(err)
			}

			bc.Items = append(bc.Items, &response.BreadcrumbItem{
				Label: states[0].FullName,
			})

			// include all the other breadcrumbs for a spruce
			// doctor to help with indexing
			if len(states) > 1 {
				for _, s := range states[1:] {
					otherBCs = append(otherBCs, &response.BreadcrumbList{
						Items: []*response.BreadcrumbItem{
							{
								Label: s.FullName,
							},
						},
					})
				}
			}
		default:
			// if working with local doctor where city not specified,
			// show the banner image and breacrumbs for the city shortlisted
			// and closest practice location of doctor

			imageIDs, err := d.cityDAL.BannerImageIDsForCity(city.ID)
			if err != nil {
				return errors.Trace(err)
			}

			bannerImageURL, err = response.URLForImageID(imageIDs[0], d.contentURL)
			if err != nil {
				return errors.Trace(err)
			}

			bc.Items = append(bc.Items, &response.BreadcrumbItem{
				Label: city.State,
			}, &response.BreadcrumbItem{
				Label: city.Name,
				Link:  fmt.Sprintf("%s/%s", d.webURL, city.ID),
			})
		}

		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	// build out qualifications
	qualifications := make([]*response.TitleDescriptionItem, 0, 3)
	if doctor.Residency != "" {
		qualifications = append(qualifications, &response.TitleDescriptionItem{
			Title:       "RESIDENCY",
			Description: doctor.Residency,
		})
	}

	if doctor.MedicalSchool != "" {
		qualifications = append(qualifications, &response.TitleDescriptionItem{
			Title:       "MEDICAL SCHOOL",
			Description: doctor.MedicalSchool,
		})
	}

	// build out availability section
	var availability []*response.ImageTextItem
	if doctor.IsSpruceDoctor {
		availability = []*response.ImageTextItem{
			{
				ImageName: "img/new_patients.png",
				Text:      "Accepting new patients with Spruce",
			},
			{
				ImageName: "img/24_hours.png",
				Text:      "Typically responds in 24 hours",
			},
		}
	} else {
		availability = []*response.ImageTextItem{
			{
				ImageName: "img/phone.png",
				Text:      fmt.Sprintf("Contact %s's office for next available appointment", doctorResponse.ShortDisplayName),
			},
		}
	}

	// add phone number if present
	var phone, phoneLink string
	var officeInfo *response.Address
	if doctor.Address != nil {
		phone = doctor.Address.Phone
		phoneLink = fmt.Sprintf("tel:%s", phone)

		if !doctor.IsSpruceDoctor {
			addressLine1 := doctor.Address.AddressLine1
			if doctor.Address.AddressLine2 != "" {
				addressLine1 += " " + doctor.Address.AddressLine2
			}

			addressLine2 := fmt.Sprintf("%s, %s %s", doctor.Address.City, doctor.Address.State, cleanupZipcode(doctor.Address.Zipcode))

			mapsURL, err := d.buildGoogleMapsImageURL(doctor)
			if err != nil {
				return nil, errors.Trace(err)
			}

			parsedPhone, err := common.ParsePhone(doctor.Address.Phone)
			if err != nil {
				return nil, errors.Trace(err)
			}

			// hard code the office hours for now as we don't have them for each doctor
			// and are going with office hours that should generally work
			officeInfo = &response.Address{
				AddressLine1:         addressLine1,
				AddressLine2:         addressLine2,
				Latitude:             doctor.Address.Latitude,
				Longitude:            doctor.Address.Longitude,
				State:                doctor.Address.State,
				City:                 doctor.Address.City,
				Phone:                parsedPhone.String(),
				Zipcode:              doctor.Address.Zipcode,
				GoogleMapsLink:       fmt.Sprintf("https://maps.google.com/?q=%s %s", doctor.Address.AddressLine1, addressLine2),
				GoogleMapsImageURL:   mapsURL,
				CondensedOfficeHours: "Mo,Tu,We,Th,Fr 09:00-17:00",
				OfficeHours: []*response.OfficeHoursItem{
					{
						Day:   "Mon",
						Hours: "9:00 am - 5:00 pm",
					},
					{
						Day:   "Tues",
						Hours: "9:00 am - 5:00 pm",
					},
					{
						Day:   "Wed",
						Hours: "9:00 am - 5:00 pm",
					},
					{
						Day:   "Thurs",
						Hours: "9:00 am - 5:00 pm",
					},
					{
						Day:   "Fri",
						Hours: "9:00 am - 5:00 pm",
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
			}
		}
	}

	// break up the insurance accepted into two containers
	// to evenly divide them
	var insuranceAccepted []*response.Container
	if len(doctor.InsurancesAccepted) > 0 {
		insuranceParts := splitN(doctor.InsurancesAccepted, 2)
		for _, ip := range insuranceParts {
			insuranceAccepted = append(insuranceAccepted, &response.Container{
				Items: ip,
			})
		}
	}

	conditionsTreatedParts := splitN(conditionsTreatedOnline, 4)
	var conditionsTreated []*response.Container
	for _, cp := range conditionsTreatedParts {
		conditionsTreated = append(conditionsTreated, &response.Container{
			Items: cp,
		})
	}

	var officeSectionTitle string
	var title string
	if doctor.IsSpruceDoctor {
		officeSectionTitle = fmt.Sprintf("See %s Online", doctorResponse.ShortDisplayName)
		title = fmt.Sprintf("%s, Dermatologist Treating Patients Online | Spruce Health", doctorResponse.LongDisplayName)
	} else {
		officeSectionTitle = fmt.Sprintf("See %s In Office", doctorResponse.ShortDisplayName)
		title = fmt.Sprintf("%s, Dermatologist in %s, %s | Spruce Health", doctorResponse.LongDisplayName, doctor.Address.City, doctor.Address.State)
	}

	dp := &response.DoctorPage{
		HTMLTitle:                 title,
		LongDisplayName:           doctorResponse.LongDisplayName,
		ProfileImageURL:           doctorResponse.ProfileImageURL,
		Description:               doctorResponse.Description,
		SEODescription:            doctor.SEODescription,
		ProfileURL:                doctorResponse.ProfileURL,
		BannerImageURL:            bannerImageURL,
		StartOnlineVisitURL:       doctorResponse.StartOnlineVisitURL,
		IsSpruceDoctor:            doctorResponse.IsSpruceDoctor,
		Specialties:               doctorResponse.Specialties,
		ReviewsSection:            reviewSection,
		StateCoverageText:         stateCoverageText,
		AcceptedInsurance:         insuranceAccepted,
		ConditionsTreated:         conditionsTreated,
		OfficeSectionTitle:        officeSectionTitle,
		Qualifications:            qualifications,
		AvailabilityItems:         availability,
		PhysicalOfficeInformation: officeInfo,
		PhoneLink:                 phoneLink,
		SpruceDoctors:             spruceDoctors,
		Breadcrumb:                &bc,
		OtherBreadcrumbs:          otherBCs,
	}

	return dp, nil
}

func (d *doctorService) buildReviewsSection(doctor *models.Doctor, doctorResponse *response.Doctor) (*response.ReviewsSection, error) {

	var title string
	var sourceImageName string
	var reviews []*response.Review
	var moreReviewsURL string
	var averageRatingImageURL string

	if doctor.IsSpruceDoctor {
		title = fmt.Sprintf("%d REVIEWS ON SPRUCE", doctor.ReviewCount)
		sourceImageName = "img/source_spruce.svg"
		averageRatingImageURL = response.StaticURL(d.staticResourceURL, response.DetermineImageNameForRating(doctor.AverageRating))

		spruceReviews, err := d.doctorDAL.SpruceReviews(doctor.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}

		reviews = make([]*response.Review, len(spruceReviews))
		for i, r := range spruceReviews {
			reviews[i] = &response.Review{
				Text:            r.Text,
				Source:          "spruce",
				SourceImageName: "img/source_spruce.svg",
				Author:          "Verified Patient",
				RatingImageURL:  response.StaticURL(d.staticResourceURL, response.DetermineImageNameForRating(r.Rating)),
				Date:            r.CreatedDate.Format("02/01/2006"),
				Rating:          r.Rating,
				Citation:        "https://www.sprucehealth.com",
			}
		}

	} else {
		title = fmt.Sprintf("%d REVIEWS ON YELP", doctor.ReviewCount)
		sourceImageName = "img/source_yelp.svg"
		if doctor.ReviewCount > 3 {
			moreReviewsURL = doctor.YelpURL
		}

		b, err := d.yelpClient.Business(strings.TrimSpace(doctor.YelpBusinessID))
		if err != nil {
			golog.Warningf("Unable to get yelp reviews for business %s: %s", doctor.YelpBusinessID, err.Error())
			return nil, nil
		}
		averageRatingImageURL = b.LargeRatingImgURL

		reviews = make([]*response.Review, len(b.Reviews))
		for i, r := range b.Reviews {
			reviews[i] = &response.Review{
				Text:            r.Excerpt,
				Source:          "yelp",
				YelpPageURL:     doctor.YelpURL,
				SourceImageName: "img/source_yelp.svg",
				RatingImageURL:  r.RatingImageLargeURL,
				Author:          r.User.Name,
				Date:            time.Unix(r.TimeCreated, 0).Format("02/01/2006"),
				Rating:          r.Rating,
				Citation:        doctor.YelpURL,
			}
		}
	}

	// build out reviews
	return &response.ReviewsSection{
		MoreReviewsURL:        moreReviewsURL,
		Title:                 title,
		SourceImageName:       sourceImageName,
		AverageRatingImageURL: averageRatingImageURL,
		Reviews:               reviews,
		AverageRating:         doctor.AverageRating,
		ReviewCount:           doctor.ReviewCount,
	}, nil
}

func buildStateCoverageText(doctorResponse *response.Doctor, states []*models.State) string {
	stateFullNameList := make([]string, len(states))
	for i, s := range states {
		stateFullNameList[i] = s.FullName
	}

	var stateText string
	if len(states) > 1 {
		stateText = strings.Join(stateFullNameList[:len(stateFullNameList)-1], ", ")
		stateText += ", and " + stateFullNameList[len(stateFullNameList)-1]
	} else {
		stateText = stateFullNameList[0]
	}

	return fmt.Sprintf("%s treats patient in %s through Spruce", doctorResponse.ShortDisplayName, stateText)

}

func (d *doctorService) buildGoogleMapsImageURL(doctor *models.Doctor) (string, error) {

	return googlemaps.GenerateImageURL(&googlemaps.StaticMapConfig{
		Width:         280,
		Height:        88,
		Scale:         2,
		Key:           d.staticMapsKey,
		URLSigningKey: d.staticMapsURLSigningKey,
		MapType:       googlemaps.MapTypeRoadmap,
		Markers: []googlemaps.MarkerConfig{
			{
				Color: googlemaps.ColorRed,
				Locations: []googlemaps.Coordinates{
					{
						Latitude:  doctor.Address.Latitude,
						Longitude: doctor.Address.Longitude,
					},
				},
			},
		},
	})
}

// splitN splits the provide slice of strings into n
// parts
func splitN(slice []string, n int) [][]string {
	parts := make([][]string, 0, n)
	partSize := len(slice) / n
	for i := 0; i < n; i++ {
		part := make([]string, 0, partSize)
		for j := i * partSize; j < (i*partSize+partSize) && j < len(slice); j++ {
			part = append(part, slice[j])
		}
		parts = append(parts, part)
	}

	return parts
}
