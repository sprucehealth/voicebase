package careprovider

import (
	"math/rand"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
)

func RandomDoctorURLs(n int, dataAPI api.DataAPI, apiDomain string) ([]string, error) {
	// attempt to get way more available doctors than needed so that we can randomly
	// pick thumbnails
	doctorIDs, err := dataAPI.AvailableDoctorIDs(5 * n)
	if err != nil {
		return nil, err
	}
	numAvailable := len(doctorIDs)

	numToPick := n
	if numAvailable <= n {
		numToPick = numAvailable
	}

	imageURLs := make([]string, 0, numToPick)

	for numToPick > 0 && numAvailable > 0 {

		randIndex := rand.Intn(numAvailable)
		doctorID := doctorIDs[randIndex]
		imageURLs = append(imageURLs, app_url.ThumbnailURL(apiDomain, api.DOCTOR_ROLE, doctorID))

		// swap the last with the index picked so that we don't pick it again
		doctorIDs[randIndex], doctorIDs[numAvailable-1] = doctorIDs[numAvailable-1], doctorIDs[randIndex]
		numToPick--
		numAvailable--
	}

	return imageURLs, nil
}
