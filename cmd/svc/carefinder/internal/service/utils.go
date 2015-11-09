package service

import (
	"math/rand"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
)

// cleanupZipcode returns the first 5 digits of the zipcode
func cleanupZipcode(zipcode string) string {
	if len(zipcode) > 5 {
		return zipcode[:5]
	}

	return zipcode
}

func shuffle(ids []string) {
	for i := len(ids) - 1; i > 0; i-- {
		j := rand.Intn(i)
		ids[i], ids[j] = ids[j], ids[i]
	}
}

func isMobile(r *http.Request) bool {
	return strings.Contains(r.UserAgent(), "iPhone") || strings.Contains(r.UserAgent(), "iPod") || strings.Contains(strings.ToLower(r.UserAgent()), "android")
}

type byReviewCount []*models.Doctor

func (c byReviewCount) Len() int      { return len(c) }
func (c byReviewCount) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byReviewCount) Less(i, j int) bool {
	return c[i].ReviewCount < c[j].ReviewCount
}

func spruceDoctorBreadcrumbs(webURL string, doctor *response.Doctor, state *models.State) []*response.BreadcrumbItem {
	return []*response.BreadcrumbItem{
		{
			Label: "Find a Dermatologist",
		},
		{
			Label: state.FullName,
			Link:  response.StatePageURL(state.Key, webURL),
		},
		{
			Label: doctor.LongDisplayName,
			Link:  response.DoctorPageURL(doctor.ID, "", webURL),
		},
	}
}

func localDoctorBreadcrumbs(webURL string, doctor *response.Doctor, city *models.City) []*response.BreadcrumbItem {
	return []*response.BreadcrumbItem{
		{
			Label: "Find a Dermatologist",
		},
		{
			Label: city.State,
			Link:  response.StatePageURL(city.StateKey, webURL),
		},
		{
			Label: city.Name,
			Link:  response.CityPageURL(city, webURL),
		},
		{
			Label: doctor.LongDisplayName,
			Link:  response.DoctorPageURL(doctor.ID, city.ID, webURL),
		},
	}
}
