package careprovider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_SelectionHandler struct {
	api.DataAPI
	doctorMap                     map[int64]*common.Doctor
	careTeamsByCase               map[int64]*common.PatientCareTeam
	eligibleDoctorIDs             []int64
	doctorIDsInCareProvidingState []int64
	availableDoctorIDs            []int64
	careProvidingStateError       error
}

func (m *mockDataAPI_SelectionHandler) Doctors(doctorIDs []int64) ([]*common.Doctor, error) {
	doctors := make([]*common.Doctor, len(doctorIDs))
	for i, doctorID := range doctorIDs {
		doctors[i] = m.doctorMap[doctorID]
	}
	return doctors, nil
}
func (m *mockDataAPI_SelectionHandler) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return 1, nil
}
func (m *mockDataAPI_SelectionHandler) GetCareTeamsForPatientByCase(patientID int64) (map[int64]*common.PatientCareTeam, error) {
	return m.careTeamsByCase, nil
}
func (m *mockDataAPI_SelectionHandler) EligibleDoctorIDs(ids []int64, careProvidingStateID int64) ([]int64, error) {
	return m.eligibleDoctorIDs, nil
}
func (m *mockDataAPI_SelectionHandler) DoctorIDsInCareProvidingState(careProvidingStateID int64) ([]int64, error) {
	return m.doctorIDsInCareProvidingState, nil
}
func (m *mockDataAPI_SelectionHandler) GetCareProvidingStateID(stateCode, pathwayTag string) (int64, error) {
	return 1, m.careProvidingStateError
}
func (m *mockDataAPI_SelectionHandler) AvailableDoctorIDs(n int) ([]int64, error) {
	return m.availableDoctorIDs, nil
}

// This test is to ensure that we don't pick the same doctor thumbnail URL
// when attempting to randomly identify doctor URLs to pick
func TestSelection_RandomPhotoSelection(t *testing.T) {
	doctors := generateDoctors(4)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: []int64{doctors[0].DoctorID.Int64(), doctors[1].DoctorID.Int64()},
		availableDoctorIDs:            []int64{doctors[0].DoctorID.Int64(), doctors[1].DoctorID.Int64(), doctors[2].DoctorID.Int64(), doctors[3].DoctorID.Int64()},
		doctorMap:                     doctorMap,
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 3 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 3, len(options))

	// the first item should be first available
	imageURLs := []string{
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[2].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[3].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[0].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[1].DoctorID.Int64())}
	fas := testFirstAvailableOption(options[0], imageURLs, t)

	// ensure that no value is shown twice in the imageURL
	seen := make(map[string]bool)
	for _, imageURL := range fas.ImageURLs {
		if seen[imageURL] {
			t.Fatalf("Seeing the same URL again")
		}
		seen[imageURL] = true
	}

	// the next item should be a care provider selection
	careProviderID, err := strconv.ParseInt(options[1].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[1], m.doctorMap[careProviderID], t)

	// third item should be a care provider selection
	careProviderID, err = strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], m.doctorMap[careProviderID], t)
}

// This test is to ensure that the API successfully returns if the client
// attempts to make a call for a non existent pathway tag or
// unavailable state
func TestSelection_Unauthenticated_NoDoctors(t *testing.T) {
	doctors := generateDoctors(4)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: []int64{doctors[0].DoctorID.Int64(), doctors[1].DoctorID.Int64()},
		availableDoctorIDs:            []int64{doctors[0].DoctorID.Int64(), doctors[1].DoctorID.Int64(), doctors[2].DoctorID.Int64(), doctors[3].DoctorID.Int64()},
		doctorMap:                     doctorMap,
		careProvidingStateError:       api.ErrNotFound("test"),
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 1 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 1, len(options))

	// the first item should be first available
	imageURLs := []string{
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[2].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[3].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[0].DoctorID.Int64()),
		app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctors[1].DoctorID.Int64())}
	testFirstAvailableOption(options[0], imageURLs, t)
}

// Test to ensure that in the unauthenticated state, we return as many doctors as we have available
// if we cannot meet the selectionCount.
func TestSelection_Unauthenticated_NotEnoughDoctors(t *testing.T) {
	doctors := generateDoctors(4)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: []int64{doctors[0].DoctorID.Int64(), doctors[1].DoctorID.Int64()},
		availableDoctorIDs:            []int64{doctors[2].DoctorID.Int64(), doctors[3].DoctorID.Int64()},
		doctorMap:                     doctorMap,
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 3 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 3, len(options))

	// the first item should be first available
	testFirstAvailableOption(options[0], []string{"", ""}, t)

	// the next item should be a care provider selection
	careProviderID, err := strconv.ParseInt(options[1].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[1], m.doctorMap[careProviderID], t)

	// third item should be a care provider selection
	careProviderID, err = strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], m.doctorMap[careProviderID], t)
}

// Test to ensure that in the unauthenticated state, doctor selection works
// when there are several available doctors in a state for a given pathway
// as well as enough doctors available overall to pick images from
// for displaying the first available state
func TestSelection_Unauthenticated_SufficientDoctors(t *testing.T) {
	doctors := generateDoctors(20)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	availableDoctorIDs := make([]int64, 20)
	for i, doctor := range doctors {
		availableDoctorIDs[i] = doctor.DoctorID.Int64()
	}

	doctorIDsInCareProvidingState := make([]int64, 10)
	for i := 0; i < 10; i++ {
		doctorIDsInCareProvidingState[i] = doctors[i].DoctorID.Int64()
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: doctorIDsInCareProvidingState,
		availableDoctorIDs:            availableDoctorIDs,
		doctorMap:                     doctorMap,
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 4 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 4, len(options))

	testFirstAvailableOption(options[0], make([]string, 6), t)
	careProviderID, err := strconv.ParseInt(options[1].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[1], doctorMap[careProviderID], t)
	careProviderID, err = strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], doctorMap[careProviderID], t)
	careProviderID, err = strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], doctorMap[careProviderID], t)
}

// Test to ensure that doctor selection works as expected when a patient
// is authenticated and the patient has a single case with a care team
// where the doctor is not eligible for the pathway. In this situation
// while we do give preference to doctors from previous cases, the doctor
// should not be picked since they are not eligible.
func TestSelection_Authenticated_SingleCase(t *testing.T) {
	doctors := generateDoctors(3)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	availableDoctorIDs := make([]int64, 3)
	for i, doctor := range doctors {
		availableDoctorIDs[i] = doctor.DoctorID.Int64()
	}

	// ensure not to include the first doctor in the doctors available for the care providing state
	doctorIDsInCareProvidingState := make([]int64, 2)
	for i := 0; i < 2; i++ {
		doctorIDsInCareProvidingState[i] = doctors[i+1].DoctorID.Int64()
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: doctorIDsInCareProvidingState,
		availableDoctorIDs:            availableDoctorIDs,
		doctorMap:                     doctorMap,
		careTeamsByCase: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   doctors[0].DoctorID.Int64(),
						ProviderRole: api.DOCTOR_ROLE,
						Status:       api.STATUS_ACTIVE,
					},
				},
			},
		},
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	// authenticated state
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 3 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 3, len(options))

	testFirstAvailableOption(options[0], make([]string, 3), t)

	careProviderID1, err := strconv.ParseInt(options[1].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[1], doctorMap[careProviderID1], t)

	careProviderID2, err := strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], doctorMap[careProviderID2], t)

	// ensure that neither doctors picked were the first doctor
	test.Equals(t, true, careProviderID1 != doctors[0].DoctorID.Int64() && careProviderID2 != doctors[0].DoctorID.Int64())
}

// Test to ensure that in the authenticated state the doctor from a previous case is picked
// and is the second result in the list if the doctor is eligible to see patients for the provided pathway
// in the given state. This is because we want to give patients the ability to pick the doctor again
func TestSelection_Authenticated_SingleCase_DoctorEligible(t *testing.T) {
	doctors := generateDoctors(10)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	availableDoctorIDs := make([]int64, 10)
	for i, doctor := range doctors {
		availableDoctorIDs[i] = doctor.DoctorID.Int64()
	}

	doctorIDsInCareProvidingState := make([]int64, 10)
	for i := 0; i < 10; i++ {
		doctorIDsInCareProvidingState[i] = doctors[i].DoctorID.Int64()
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: doctorIDsInCareProvidingState,
		availableDoctorIDs:            availableDoctorIDs,
		doctorMap:                     doctorMap,
		eligibleDoctorIDs:             []int64{doctors[0].DoctorID.Int64()},
		careTeamsByCase: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   doctors[0].DoctorID.Int64(),
						ProviderRole: api.DOCTOR_ROLE,
						Status:       api.STATUS_ACTIVE,
					},
				},
			},
		},
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	// authenticated state
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 4 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 4, len(options))

	testFirstAvailableOption(options[0], make([]string, 6), t)

	// the first care provider MUST be the doctor from the previous case
	testCareProviderSelection(options[1], doctors[0], t)

	careProviderID, err := strconv.ParseInt(options[2].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[2], doctorMap[careProviderID], t)

	careProviderID, err = strconv.ParseInt(options[3].(map[string]interface{})["care_provider_id"].(string), 10, 64)
	testCareProviderSelection(options[3], doctorMap[careProviderID], t)
}

// Test to ensure that if the patient has multiple cases with all doctors eligible
// for the pathway such that no other doctors need to be randomly selected, then we
// only pick the doctors from the previous cases
func TestSelection_Authenticated_MultipleCases_AllDoctorsEligible(t *testing.T) {
	doctors := generateDoctors(20)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	availableDoctorIDs := make([]int64, 20)
	for i, doctor := range doctors {
		availableDoctorIDs[i] = doctor.DoctorID.Int64()
	}

	doctorIDsInCareProvidingState := make([]int64, 10)
	for i := 0; i < 10; i++ {
		doctorIDsInCareProvidingState[i] = doctors[i].DoctorID.Int64()
	}

	eligibleDoctorIDs := make([]int64, 10)
	careTeamsByCase := make(map[int64]*common.PatientCareTeam)
	for i := 10; i < 20; i++ {
		eligibleDoctorIDs[i-10] = doctors[i].DoctorID.Int64()
		careTeamsByCase[int64(i)] = &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					ProviderID:   doctors[i].DoctorID.Int64(),
					ProviderRole: api.DOCTOR_ROLE,
					Status:       api.STATUS_ACTIVE,
				},
			},
		}
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: doctorIDsInCareProvidingState,
		availableDoctorIDs:            availableDoctorIDs,
		doctorMap:                     doctorMap,
		eligibleDoctorIDs:             eligibleDoctorIDs,
		careTeamsByCase:               careTeamsByCase,
	}

	h := NewSelectionHandler(m, "api.spruce.local", 3)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	// authenticated state
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 4 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 4, len(options))

	testFirstAvailableOption(options[0], make([]string, 6), t)

	// the 3 providers specified must be doctors at location 10, 11, 12
	testCareProviderSelection(options[1], doctors[10], t)
	testCareProviderSelection(options[2], doctors[11], t)
	testCareProviderSelection(options[3], doctors[12], t)
}

// Test to ensure that if the patient has multiple cases with some of the doctors from the cases
// eligible for the current pathway/state combination, then we definitely pick those some
// and then randomly pick the rest from the rest of the available doctors
func TestSelection_Authenticated_MultipleCases_SomeDoctorsEligible(t *testing.T) {
	doctors := generateDoctors(20)
	doctorMap := make(map[int64]*common.Doctor)
	for _, doctor := range doctors {
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	availableDoctorIDs := make([]int64, 20)
	for i, doctor := range doctors {
		availableDoctorIDs[i] = doctor.DoctorID.Int64()
	}

	doctorIDsInCareProvidingState := make([]int64, 10)
	for i := 0; i < 10; i++ {
		doctorIDsInCareProvidingState[i] = doctors[i].DoctorID.Int64()
	}

	// make the first 5 doctors in the list eligible as well as members of the care team for
	// the first 5 cases
	eligibleDoctorIDs := make([]int64, 5)
	careTeamsByCase := make(map[int64]*common.PatientCareTeam)
	for i := 0; i < 5; i++ {
		eligibleDoctorIDs[i] = doctors[i].DoctorID.Int64()
		careTeamsByCase[int64(i)] = &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					ProviderID:   doctors[i].DoctorID.Int64(),
					ProviderRole: api.DOCTOR_ROLE,
					Status:       api.STATUS_ACTIVE,
				},
			},
		}
	}

	// now there are 5 more cases that contain ineligible doctors
	for i := 10; i < 15; i++ {
		careTeamsByCase[int64(i)] = &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					ProviderID:   doctors[i].DoctorID.Int64(),
					ProviderRole: api.DOCTOR_ROLE,
					Status:       api.STATUS_ACTIVE,
				},
			},
		}
	}

	m := &mockDataAPI_SelectionHandler{
		doctorIDsInCareProvidingState: doctorIDsInCareProvidingState, // 0-10 doctors (so that doctors 5-9 are picked beyond the doctors from the cases)
		availableDoctorIDs:            availableDoctorIDs,            // all doctors
		doctorMap:                     doctorMap,                     // all doctors
		eligibleDoctorIDs:             eligibleDoctorIDs,             // first 5 doctors
		careTeamsByCase:               careTeamsByCase,               // first 5 and 3rd 5 doctors
	}

	h := NewSelectionHandler(m, "api.spruce.local", 10)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local?state_code=CA&pathway_id=acne", nil)
	test.OK(t, err)

	// authenticated state
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// unmarshal the response to check the output
	var jsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &jsonMap))

	// there should be 10 items total in the response
	options := jsonMap["options"].([]interface{})
	test.Equals(t, 11, len(options))

	testFirstAvailableOption(options[0], make([]string, 6), t)

	// the first 5 providers specified must be doctors at location 0, 1, 2, 3, 4
	// because those doctors were from the previous case
	testCareProviderSelection(options[1], doctors[0], t)
	testCareProviderSelection(options[2], doctors[1], t)
	testCareProviderSelection(options[3], doctors[2], t)
	testCareProviderSelection(options[4], doctors[3], t)
	testCareProviderSelection(options[5], doctors[4], t)

	// the last 5 providers should be doctors between 5 and 9 (that have doctor ids 6 through 11)
	remainingDoctors := make(map[int64]bool)
	for i := 6; i < 11; i++ {
		careProviderID, err := strconv.ParseInt(options[i].(map[string]interface{})["care_provider_id"].(string), 10, 64)
		test.OK(t, err)
		remainingDoctors[careProviderID] = true
		testCareProviderSelection(options[i], doctorMap[careProviderID], t)
	}

	// ensure that the remaining doctors that were picked were from the list of doctors
	// that were eligible but not in the patient's cases
	for i := 6; i <= 10; i++ {
		test.Equals(t, true, remainingDoctors[int64(i)])
	}

}

func generateDoctors(n int) []*common.Doctor {
	doctors := make([]*common.Doctor, n)
	for i := 0; i < n; i++ {
		doctors[i] = &common.Doctor{
			DoctorID:         encoding.NewObjectID(int64(i + 1)),
			ShortDisplayName: fmt.Sprintf("doctorDisplay%d", i),
			LongTitle:        fmt.Sprintf("doctorTitle%d", i),
		}
	}
	return doctors
}

func testCareProviderSelection(j interface{}, doctor *common.Doctor, t *testing.T) {
	var cps careProviderSelection
	jsonData, err := json.Marshal(j)
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &cps))
	test.OK(t, cps.Validate("care_provider_selection"))
	test.Equals(t, "care_provider_selection:care_provider", cps.Type)
	test.Equals(t, doctor.ShortDisplayName, cps.Title)
	test.Equals(t, doctor.LongTitle, cps.Description)
	test.Equals(t, doctor.DoctorID.Int64(), cps.CareProviderID)
	test.Equals(t, fmt.Sprintf("Choose %s", doctor.ShortDisplayName), cps.ButtonTitle)
	test.Equals(t, app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, doctor.DoctorID.Int64()), cps.ImageURL)
}

func testFirstAvailableOption(j interface{}, imageURLs []string, t *testing.T) firstAvailableSelection {
	var fas firstAvailableSelection
	jsonData, err := json.Marshal(j)
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &fas))
	test.OK(t, fas.Validate("care_provider_selection"))
	test.Equals(t, "care_provider_selection:first_available", fas.Type)
	test.Equals(t, "First Available", fas.Title)
	test.Equals(t, "Choose this option for a response within 24 hours. You'll be treated by the first available doctor on Spruce.", fas.Description)
	test.Equals(t, "Choose First Available", fas.ButtonTitle)
	test.Equals(t, len(imageURLs), len(fas.ImageURLs))
	return fas
}
