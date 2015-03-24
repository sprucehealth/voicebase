package careprovider

import (
	"math/rand"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
)

func RandomDoctorURLs(n int, dataAPI api.DataAPI, apiDomain string, unpreferredDoctorIDs []int64) ([]string, error) {
	// attempt to get way more available doctors than needed so that we can randomly
	// pick thumbnails
	doctorIDs, err := dataAPI.AvailableDoctorIDs(5 * n)
	if err != nil {
		return nil, err
	}
	numAvailable := len(doctorIDs)

	availableDoctorIDMap := make(map[int64]bool, len(doctorIDs))
	for _, doctorID := range doctorIDs {
		availableDoctorIDMap[doctorID] = true
	}

	numToPick := n
	if numAvailable <= n {
		numToPick = numAvailable
	}

	// create a set of the unpreferred list
	unpreferredDoctorIDsMap := make(map[int64]bool)
	filteredUnPreferredList := make([]int64, 0, len(unpreferredDoctorIDs))
	for _, unpreferredID := range unpreferredDoctorIDs {
		// don't include any that are not in the available doctors map
		if !availableDoctorIDMap[unpreferredID] {
			continue
		}

		unpreferredDoctorIDsMap[unpreferredID] = true
		filteredUnPreferredList = append(filteredUnPreferredList, unpreferredID)
	}

	imageURLs := make([]string, 0, numToPick)
	for numToPick > 0 && numAvailable > 0 {

		randIndex := rand.Intn(numAvailable)
		doctorID := doctorIDs[randIndex]

		// swap the last with the index picked so that we don't pick it again
		doctorIDs[randIndex], doctorIDs[numAvailable-1] = doctorIDs[numAvailable-1], doctorIDs[randIndex]
		numAvailable--

		// avoid picking any ids in the unpreferred list
		if unpreferredDoctorIDsMap[doctorID] {
			continue
		}

		imageURLs = append(imageURLs, app_url.ThumbnailURL(apiDomain, api.DOCTOR_ROLE, doctorID))
		numToPick--
	}

	// if at this point we have not been able to pick as many images as we'd like, go ahead
	// and dig into the pool of unpreferred doctor ids
	for i := 0; i < len(filteredUnPreferredList) && numToPick > 0; i++ {
		imageURLs = append(imageURLs, app_url.ThumbnailURL(apiDomain, api.DOCTOR_ROLE, filteredUnPreferredList[i]))
		numToPick--
	}

	return imageURLs, nil
}
