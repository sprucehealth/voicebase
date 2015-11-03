package response

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

func URLForImageID(imageID, contentURL string) (string, error) {
	u, err := url.Parse(imageID)
	if err != nil {
		return "", errors.Trace(err)
	}
	p := strings.SplitN(u.Path, "/", 3)
	return fmt.Sprintf("%s/%s", contentURL, p[2]), nil
}

func CityPageURL(city *models.City, webURL string) string {
	return fmt.Sprintf("%s/%s/%s", webURL, city.StateKey, city.ID)
}

func StatePageURL(stateKey string, webURL string) string {
	return fmt.Sprintf("%s/%s", webURL, strings.ToLower(stateKey))
}

func DoctorPageURL(doctorID, cityID, webURL string) string {
	if cityID == "" {
		return fmt.Sprintf("%s/%s", webURL, doctorID)
	}

	return fmt.Sprintf("%s/%s?city_id=%s", webURL, doctorID, cityID)
}

func StaticURL(staticURL, imageName string) string {
	return fmt.Sprintf("%s/%s", staticURL, imageName)
}

func roundToClosestHalve(val float64) float64 {
	var rounded float64
	_, frac := math.Modf(val)

	switch {
	case frac <= 0.25:
		rounded = math.Floor(val)
	case frac > 0.25 && frac <= 0.5:
		rounded = math.Floor(val) + 0.5
	case frac > 0.5 && frac <= 0.75:
		rounded = math.Floor(val) + 0.5
	case frac > 0.75:
		rounded = math.Ceil(val)
	}

	return rounded
}

func DetermineImageNameForRating(rating float64) string {
	return determineImageNameForRating(roundToClosestHalve(rating))
}

func determineImageNameForRating(rating float64) string {
	var starRatingImg string
	switch rating {
	case 3.0:
		starRatingImg = "img/stars_three.svg"
	case 3.5:
		starRatingImg = "img/stars_threefive.svg"
	case 4.0:
		starRatingImg = "img/stars_four.svg"
	case 4.5:
		starRatingImg = "img/stars_fourfive.svg"
	case 5.0:
		starRatingImg = "img/stars_five.svg"
	}
	return starRatingImg
}
