package careprovider

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_RandomDoctorURLs struct {
	api.DataAPI
	doctorIDs []int64
}

func (m *mockDataAPI_RandomDoctorURLs) AvailableDoctorIDs(n int) ([]int64, error) {
	return m.doctorIDs, nil
}

func TestRandomDoctorURLs(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local")
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))

	// ensure that no image is repeated
	seen := make(map[string]bool)
	for _, imageURL := range imageURLs {
		test.Equals(t, false, seen[imageURL])
		seen[imageURL] = true
	}

}
