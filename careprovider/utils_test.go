package careprovider

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
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

	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local", nil)
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))

	// ensure that no image is repeated
	seen := make(map[string]bool)
	for _, imageURL := range imageURLs {
		test.Equals(t, false, seen[imageURL])
		seen[imageURL] = true
	}

}

// This test is to ensure that if an unpreferred list is presented to the method to get
// random doctor urls, then that list is obeyed if there are enough available doctor ids.
func TestRandomDoctorURLs_UnpreferredList_AllIncludedInAvailableList(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	unpreferredList := []int64{6, 7, 8, 9}
	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local", unpreferredList)
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))

	// ensure that no image is repeated
	seen := make(map[string]bool)
	for _, imageURL := range imageURLs {
		test.Equals(t, false, seen[imageURL])
		seen[imageURL] = true
	}

	checkNoIDsFromUnPreferredList(t, unpreferredList, imageURLs)
}

// This test is to ensure that we ignore any ids in the unpreferred list that
// are not in the available list
func TestRandomDoctorURLs_UnpreferredList_SomeNotIncludedInAvailableList(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	// now lets expand the unpreferredList to includeIDs not present in the availableList
	unpreferredList := []int64{6, 7, 8, 9, 10, 11, 12}
	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local", unpreferredList)
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))
	checkNoIDsFromUnPreferredList(t, unpreferredList, imageURLs)
}

// This test is to ensure that we dip into the unpreferred list if there are not enough
// ids present in the available list. Oh and also making sure that the ones picked from the unpreferred list
// are at the very end of the list.
func TestRandomDoctorURLs_UnpreferredList_ForcedToUseFromUnPreferredList(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	// now lets expand the unpreferredList to includeIDs not present in the availableList
	unpreferredList := []int64{1, 6, 7, 8, 9}
	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local", unpreferredList)
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))

	unpreferredImageURLs := make(map[string]bool, len(unpreferredList))
	for _, doctorID := range unpreferredList {
		unpreferredImageURLs[app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctorID)] = true
	}

	// at this point only one of the imageURLs from the unpreferredList should be included. And that too, it should be at the end.
	for i := 0; i < 4; i++ {
		if unpreferredImageURLs[imageURLs[i]] {
			t.Fatalf("Expected %s to not be present in the list until there were no more ids to pick from and so we dipped into the unpreferred pool", imageURLs[i])
		}
	}

	if !unpreferredImageURLs[imageURLs[4]] {
		t.Fatalf("Expected %s to have been picked from the unpreferred list but it wasn't", imageURLs[4])
	}
}

// This test is to ensure that we dip into the unpreferred list to only pick from it
// if we have no other choice.
func TestRandomDoctorURLs_UnpreferredList_OnlyPickFromUnpreferredList(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	// now lets expand the unpreferredList to includeIDs not present in the availableList
	unpreferredList := m.doctorIDs
	imageURLs, err := RandomDoctorURLs(5, m, "api.spruce.local", unpreferredList)
	test.OK(t, err)
	test.Equals(t, 5, len(imageURLs))

	unpreferredImageURLs := make(map[string]bool, len(unpreferredList))
	for _, doctorID := range unpreferredList {
		unpreferredImageURLs[app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctorID)] = true
	}

	// all images picked should be present in the unpreferred list
	for _, imageURL := range imageURLs {
		if !unpreferredImageURLs[imageURL] {
			t.Fatalf("Expected %s to be present in the imageURLs generated from ids in the unpreferred list but it wasn't %s", imageURL)
		}
	}
}

// This test is to ensure that we ignore the items in the unpreferred list that are not in the available list
// and then dip into the unpreferred list should the need arise due to insufficient ids in the available list.
// Oh and also making sure that the ones picked from the unpreferred list are at the very end.
func TestRandomDoctorURLs_UnpreferredList_ForcedToUseFromUnPreferredList_FilterOutNotIncludedInAvailableList(t *testing.T) {
	m := &mockDataAPI_RandomDoctorURLs{
		doctorIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	unpreferredList := []int64{1, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	imageURLs, err := RandomDoctorURLs(20, m, "api.spruce.local", unpreferredList)
	test.OK(t, err)
	test.Equals(t, 9, len(imageURLs))

	unpreferredImageURLs := make(map[string]bool, len(unpreferredList))
	for _, doctorID := range unpreferredList {
		unpreferredImageURLs[app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctorID)] = true
	}

	imageURLsForAvailableDoctors := make(map[string]bool)
	for _, doctorID := range m.doctorIDs {
		imageURLsForAvailableDoctors[app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctorID)] = true
	}

	// first 4 should be from available list (but not in unpreferred list)
	for i := 0; i < 4; i++ {
		if unpreferredImageURLs[imageURLs[i]] {
			t.Fatalf("Expected %s to not be present in the list until there were no more ids to pick from and so we dipped into the unpreferred pool", imageURLs[i])
		} else if !imageURLsForAvailableDoctors[imageURLs[i]] {
			t.Fatalf("Expected %s to be present in the imageURLs from the available list but it wasn't", imageURLs[i])
		}
	}

	// last 5 should be from unpreferred list but also included in the available list
	for i := 4; i < 9; i++ {
		if !unpreferredImageURLs[imageURLs[i]] {
			t.Fatalf("Expected %s to have been picked from the unpreferred list but it wasn't", imageURLs[i])
		} else if !imageURLsForAvailableDoctors[imageURLs[i]] {
			t.Fatalf("Expected %s to be present in the imageURLs from the available list but it wasn't", imageURLs[i])
		}
	}
}

func checkNoIDsFromUnPreferredList(t *testing.T, unpreferredList []int64, imageURLs []string) {
	// create a set of all the imageURLs for doctorIDs in the unpreferredList
	unpreferredImageURLs := make(map[string]bool, len(unpreferredList))
	for _, doctorID := range unpreferredList {
		unpreferredImageURLs[app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctorID)] = true
	}

	// ensure that no doctorID from the unpreferredList was picked
	for _, imageURL := range imageURLs {
		if unpreferredImageURLs[imageURL] {
			t.Fatalf("Expected %s to not in the list of imageURls", imageURL)
		}
	}
}
