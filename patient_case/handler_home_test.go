package patient_case

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockHomeHandlerDataAPI struct {
	api.DataAPI
	stateName         string
	patientCases      []*common.PatientCase
	patientVisits     []*common.PatientVisit
	treatmentPlans    []*common.TreatmentPlan
	pathwayMap        map[string]*common.Pathway
	isElligible       bool
	caseNotifications map[int64][]*common.CaseNotification
	careTeamsByCase   map[int64]*common.PatientCareTeam
	formEntryExists   bool
	patientZipcode    string
}

// overriding all the data access methods that are relevant to the home API

func (m *mockHomeHandlerDataAPI) State(stateCode string) (string, string, error) {
	return m.stateName, m.stateName, nil
}
func (m *mockHomeHandlerDataAPI) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return 1, nil
}
func (m *mockHomeHandlerDataAPI) GetCasesForPatient(patientID int64, states []string) ([]*common.PatientCase, error) {
	return m.patientCases, nil
}
func (m *mockHomeHandlerDataAPI) PathwayForTag(tag string, opts api.PathwayOption) (*common.Pathway, error) {
	return m.pathwayMap[tag], nil
}
func (m *mockHomeHandlerDataAPI) SpruceAvailableInState(stateAbbreviation string) (bool, error) {
	return m.isElligible, nil
}
func (m *mockHomeHandlerDataAPI) NotificationsForCases(patientID int64, types map[string]reflect.Type) (map[int64][]*common.CaseNotification, error) {
	return m.caseNotifications, nil
}
func (m *mockHomeHandlerDataAPI) GetCareTeamsForPatientByCase(patientID int64) (map[int64]*common.PatientCareTeam, error) {
	return m.careTeamsByCase, nil
}
func (m *mockHomeHandlerDataAPI) DoesActiveTreatmentPlanForCaseExist(caseID int64) (bool, error) {
	return true, nil
}
func (m *mockHomeHandlerDataAPI) GetVisitsForCase(caseID int64, statuses []string) ([]*common.PatientVisit, error) {
	return m.patientVisits, nil
}
func (m *mockHomeHandlerDataAPI) GetTreatmentPlansForCase(caseID int64) ([]*common.TreatmentPlan, error) {
	return m.treatmentPlans, nil
}
func (m *mockHomeHandlerDataAPI) FormEntryExists(tableName, uniqueKey string) (bool, error) {
	return m.formEntryExists, nil
}
func (m *mockHomeHandlerDataAPI) PatientLocation(patientID int64) (zipcode string, state string, err error) {
	return m.patientZipcode, "", nil
}

type mockHandlerHomeAddressValidationAPI struct {
	lookupFunc func(string) (*address.CityState, error)
}

func (m *mockHandlerHomeAddressValidationAPI) ZipcodeLookup(zipcode string) (*address.CityState, error) {
	if m.lookupFunc != nil {
		return m.lookupFunc(zipcode)
	}
	return nil, nil
}

// Test the state of the home cards for an unauthenticated user
// in whose state Spruce is available.
// Expected home cards:
// 1. Start Visit card
// 2. Learn about spruce section
func TestHome_UnAuthenticated_Eligible(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	dataAPI.isElligible = true

	// lookup unauthenticated by zipcode
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	testUnauthenticatedExperience(t, w)

	// lookup unauthenticated by state
	r, err = http.NewRequest("GET", "/?state_code=CA", nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	testUnauthenticatedExperience(t, w)
}

// Test the state of the home cards for an unauthenticated user
// in whose state Spruce is not available and the user has
// not yet signed up to be notified when Spruce will be available
// Expected home cards:
// 1. Sign up to be notified card
// 2. Learn about spruce section
func TestHome_UnAuthenticated_Ineligible(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	// simulate the scneario where the user is not eligible and has not
	// yet signed up to be notified when spruce is available in their state
	dataAPI.isElligible = false
	dataAPI.formEntryExists = false

	// lookup unauthenticated by zipcode
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	setRequestHeaders(r)

	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))
	test.Equals(t, false, jsonMap["show_action_button"].(bool))

	var notifyMeCard phNotifyMeView
	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &notifyMeCard))
	testNotifyMeCard(t, &notifyMeCard, "California")
	testLearnAboutSpruceSection(t, items[1].(map[string]interface{}))
}

// Test the state of the home cards for an unauthenticated user
// in whose state Spruce is not available and the user has signed up
// to be notified when Spruce will be available in their state.
// Expected home cards:
// 1. Notification Confirmation
// 2. Learn about spruce section
func TestHome_UnAuthenticated_Ineligible_NotifyConfirmation(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	// simulate the scneario where the user is not eligible and has signed up
	// to be notified when spruce becomes available in their state
	dataAPI.isElligible = false
	dataAPI.formEntryExists = true

	// lookup unauthenticated by zipcode
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	var card phHeroIconView
	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &card))
	testNotifyMeConfirmationCard(t, &card, "California")
	testLearnAboutSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards for a user that has an incomplete visit but did not pick a doctor.
// Expected home cards:
// 1. Continue your visit card
// 2. Contact us card
// 3. Learn about spruce section
func TestHome_Authenticated_IncompleteCase_NoDoctor(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	patientVisitID := int64(10)

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusOpen,
		},
	}

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: "Care Coordinator",
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNIncompleteVisit,
				UID:              CNIncompleteVisit,
				Data: &incompleteVisitNotification{
					PatientVisitID: patientVisitID,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	// there should be 3 items in the home feed (continue visit, contact us, learn more about spruce)
	items := jsonMap["items"].([]interface{})
	test.Equals(t, 3, len(items))

	// first card should be a continue visit card
	var card phContinueVisit
	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &card))
	testContinueVisitCard(t, &card, caseName, patientVisitID, "", "")

	// second card should be a contact us section
	testContactUsSection(t, items[1].(map[string]interface{}))

	// third card should be learn more about spruce section
	testLearnAboutSpruceSection(t, items[2].(map[string]interface{}))
}

// Test home cards when user has an incomplete visit with a doctor picked.
// Expected home cards:
// 1. Continue your visit card (with doctor assigned)
// 2. Contact us card
// 3. Learn about spruce section
func TestHome_Authenticated_IncompleteCase_DoctorAssigned(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	patientVisitID := int64(10)

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusOpen,
		},
	}

	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 1)
	doctorShortDisplayName := "Dr. X"
	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					ProviderID:       1,
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorShortDisplayName,
				},
				{
					ProviderID:       2,
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: "Care Coordinator",
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNIncompleteVisit,
				UID:              CNIncompleteVisit,
				Data: &incompleteVisitNotification{
					PatientVisitID: patientVisitID,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	// there should be 3 items in the home feed (continue visit, contact us, learn more about spruce)
	items := jsonMap["items"].([]interface{})
	test.Equals(t, 3, len(items))

	// first card should be a continue visit card
	var card phContinueVisit
	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &card))
	testContinueVisitCard(t, &card, caseName, patientVisitID, doctorProfileURL, doctorShortDisplayName)

	// second card should be a contact us section
	testContactUsSection(t, items[1].(map[string]interface{}))

	// third card should be learn more about spruce section
	testLearnAboutSpruceSection(t, items[2].(map[string]interface{}))
}

// Test home cards when user has a pre-submission-triaged visit with a doctor picked.
// Expected home cards:
// 1. triaged visit card
// 2. Learn about spruce section
func TestHome_Authenticated_CaseTriaged(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()
	dataAPI.patientZipcode = "94115"
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	now := time.Now()
	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusPreSubmissionTriage,
			ClosedDate:     &now,
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	// there should be 2 items in the home feed (visit triaged, learn more about spruce)
	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// first card should be a section explaining a triaged visit

	section := items[0].(map[string]interface{})
	test.Equals(t, "Your rash visit has ended and you should seek medical care today.", section["title"])
	jsonData, err := json.Marshal(section["views"])
	test.OK(t, err)
	var psts []phSmallIconText
	test.OK(t, json.Unmarshal(jsonData, &psts))
	test.Equals(t, 1, len(psts))
	test.Equals(t, "How to find a local care provider", psts[0].Title)
	test.Equals(t, "https://www.google.com/?gws_rd=ssl#q=urgent+care+in+94115", psts[0].ActionURL)

	// second card should be learn more about spruce section
	testLearnAboutSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when user has a pre-submission-triaged visit that has expired and should not longer be shown to patient.
// Expected home cards:
// 1. Learn about spruce section
func TestHome_Authenticated_CaseTriaged_Expired(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()
	dataAPI.patientZipcode = "94115"
	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	twoDaysAgo := time.Now().Add(-2 * 24 * time.Hour)
	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusPreSubmissionTriage,
			ClosedDate:     &twoDaysAgo,
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	// there should be 2 items in the home feed (visit triaged, learn more about spruce)
	items := jsonMap["items"].([]interface{})
	test.Equals(t, 1, len(items))
	testLearnAboutSpruceSection(t, items[0].(map[string]interface{}))
}

// Test home cards when user has a completed visit but no doctor picked
// Expected home cards:
// 1. Completed visit card
// 2. Referral card
func TestHome_Authenticated_CompletedVisit_NoDoctor(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	patientVisitID := int64(10)

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
		},
	}

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: "Care Coordinator",
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNVisitSubmitted,
				UID:              CNVisitSubmitted,
				Data: &visitSubmittedNotification{
					CaseID:  1,
					VisitID: patientVisitID,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], "Pending Review")

	// test the visit submitted view within the case card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, "We'll notify you when your doctor has reviewed your visit.", standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, app_url.IconVisitSubmitted.String(), standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when user has a completed visit and a doctor picked
// Expected home cards:
// 1. Completed visit card (with doctor's picture)
// 2. Referral Card
func TestHome_Authenticated_CompletedVisit_DoctorAssigned(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"
	patientVisitID := int64(10)
	doctorShortDisplayName := "Dr. X"
	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 1)

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					ProviderID:       1,
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorShortDisplayName,
				},
				{
					ProviderID:       2,
					Status:           api.STATUS_ACTIVE,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: "Care Coordinator",
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNVisitSubmitted,
				UID:              CNVisitSubmitted,
				Data: &visitSubmittedNotification{
					CaseID:  1,
					VisitID: patientVisitID,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorShortDisplayName))

	// test the visit submitted view within the case card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("We'll notify you when %s has reviewed your visit.", doctorShortDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, doctorProfileURL, standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when a user has a message from their care coordinator but no doctor picked.
// Expected home cards:
// 1. Case Card with message notification (picture of CC)
// 2. Referral Card
func TestHome_Authenticated_Messages_NoDoctor(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
		},
	}

	maProfileURL := app_url.ThumbnailURL("api.spruce.local", api.MA_ROLE, 1)
	maDisplayName := "Care Coordinator"
	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNMessage,
				UID:              CNMessage,
				Data: &messageNotification{
					MessageID: 1,
					DoctorID:  1,
					CaseID:    1,
					Role:      api.MA_ROLE,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], "Pending Review")

	// test the message card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("You have a new message from %s.", maDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, maProfileURL, standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when the patient has multiple messages from their care coordinator
// but no doctor picked
// Expected home cards:
// 1. Case card with CC's profile image and indication of multiple notifications
// 2. Referral Card
func TestHome_Authenticated_MultipleMessages_NoDoctor(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
		},
	}

	maProfileURL := app_url.ThumbnailURL("api.spruce.local", api.MA_ROLE, 1)
	maDisplayName := "Care Coordinator"
	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNMessage,
				UID:              CNMessage,
				Data: &messageNotification{
					MessageID: 1,
					DoctorID:  1,
					CaseID:    1,
					Role:      api.MA_ROLE,
				},
			},
			{
				ID:               2,
				PatientCaseID:    1,
				NotificationType: CNMessage,
				UID:              CNMessage,
				Data: &messageNotification{
					MessageID: 2,
					DoctorID:  1,
					CaseID:    1,
					Role:      api.MA_ROLE,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], "Pending Review")

	// test the message card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, "You have two new updates.", standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, maProfileURL, standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards ehen the user has a message from their care coordinator
// and a doctor picked.
// Expected home cards:
// 1. Case Card with cc's image and message notification
// 2. Referral Card
func TestHome_Authenticated_Message_DoctorAssigned(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	maProfileURL := app_url.ThumbnailURL("api.spruce.local", api.MA_ROLE, 1)
	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNMessage,
				UID:              CNMessage,
				Data: &messageNotification{
					MessageID: 1,
					DoctorID:  1,
					CaseID:    1,
					Role:      api.MA_ROLE,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	// test the message card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("You have a new message from %s.", maDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, maProfileURL, standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when a user has a message from their care coordinator
// and their visit has been treated by their doctor.
// Expected home cards:
// 1. Completed visit
// 2. Meet your care team card
// 3. Referral card
func TestHome_Authenticated_Message_VisitTreated(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	dataAPI.patientVisits = []*common.PatientVisit{
		{
			PatientVisitID: encoding.NewObjectID(1),
			PatientCaseID:  encoding.NewObjectID(1),
			Status:         common.PVStatusTreated,
		},
	}

	maProfileURL := app_url.ThumbnailURL("api.spruce.local", api.MA_ROLE, 1)
	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNMessage,
				UID:              CNMessage,
				Data: &messageNotification{
					MessageID: 1,
					DoctorID:  1,
					CaseID:    1,
					Role:      api.MA_ROLE,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be three items (the case card, meet the care team and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 3, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	// test the message card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("You have a new message from %s.", maDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, maProfileURL, standardView.IconURL)

	testMeetCareTeamSection(t, caseName, items[1].(map[string]interface{}))
	testShareSpruceSection(t, items[2].(map[string]interface{}))
}

// Test home cards when the patient has their visit treated by a doctor
// but have not viewed their treatment plan yet
// Expected home cards:
// 1. Case card with TP notification
// 2. Meet your care team section
// 3. Referral Card
func TestHome_Authenticated_VisitTreated_TPNotViewed(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	dataAPI.patientVisits = []*common.PatientVisit{
		{
			PatientVisitID: encoding.NewObjectID(1),
			PatientCaseID:  encoding.NewObjectID(1),
			Status:         common.PVStatusTreated,
		},
	}

	dataAPI.treatmentPlans = []*common.TreatmentPlan{
		{
			PatientViewed: false,
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNTreatmentPlan,
				UID:              CNTreatmentPlan,
				Data: &treatmentPlanNotification{
					MessageID:       1,
					DoctorID:        1,
					TreatmentPlanID: 2,
					CaseID:          1,
				},
			},
		},
	}

	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2)
	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 3, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	// test the treatment plan card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("%s reviewed your visit and created a treatment plan.", doctorDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, doctorProfileURL, standardView.IconURL)

	testMeetCareTeamSection(t, caseName, items[1].(map[string]interface{}))
	testShareSpruceSection(t, items[2].(map[string]interface{}))
}

// Test home cards when the user has no updates for their case.
// Expected home cards:
// 1. Case card with no updates and picture of doctor
// 2. Referral card
func TestHome_Authenticated_NoUpdates(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2)
	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, "", standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, doctorProfileURL, standardView.IconURL)
	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when when the user has viewed their treatment plan
// Expected home cards:
// 1. Case card with no updates
// 2. Referral Card
func TestHome_Authenticated_VisitTreated_TPViewed(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	dataAPI.patientVisits = []*common.PatientVisit{
		{
			PatientVisitID: encoding.NewObjectID(1),
			PatientCaseID:  encoding.NewObjectID(1),
			Status:         common.PVStatusTreated,
		},
	}

	dataAPI.treatmentPlans = []*common.TreatmentPlan{
		{
			PatientViewed: true,
		},
	}

	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2)
	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	// test the treatment plan card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, "", standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, doctorProfileURL, standardView.IconURL)

	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when the user has multiple treatment plans one of which has
// not been viewed yet
// Expected home cards:
// 1. Case Card with TP notification
// 2. Referral Card
func TestHome_Authenticated_MultipleTPs(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName := "Rash"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	dataAPI.patientVisits = []*common.PatientVisit{
		{
			PatientVisitID: encoding.NewObjectID(1),
			PatientCaseID:  encoding.NewObjectID(1),
			Status:         common.PVStatusTreated,
		},
	}

	dataAPI.treatmentPlans = []*common.TreatmentPlan{
		{
			PatientViewed: true,
		},
		{
			PatientViewed: false,
		},
	}

	maDisplayName := "Care Coordinator"
	doctorDisplayName := "Dr. X"
	doctorProfileURL := app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2)

	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName,
					LongDisplayName:  doctorDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNTreatmentPlan,
				UID:              CNTreatmentPlan,
				Data: &treatmentPlanNotification{
					MessageID:       1,
					DoctorID:        1,
					TreatmentPlanID: 2,
					CaseID:          1,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	// there should be two items (the case card and the referral card)
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))

	// test the case card
	caseCard := items[0].(map[string]interface{})
	testCaseCard(t, caseCard, dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName))

	// test the treatment plan card
	jsonData, err := json.Marshal(caseCard["notification_view"])
	test.OK(t, err)
	var standardView phCaseNotificationStandardView
	test.OK(t, json.Unmarshal(jsonData, &standardView))
	test.Equals(t, "patient_home_case_notification:standard", standardView.Type)
	test.OK(t, standardView.Validate())
	test.Equals(t, fmt.Sprintf("%s reviewed your visit and created a treatment plan.", doctorDisplayName), standardView.Title)
	test.Equals(t, app_url.ViewCaseAction(1).String(), standardView.ActionURL.String())
	test.Equals(t, doctorProfileURL, standardView.IconURL)

	testShareSpruceSection(t, items[1].(map[string]interface{}))
}

// Test home cards when there are multiple incomplete visits
// Expected home cards:
// 1. Incomplete card
// 2. Incomplete visit card
// 3. Learn about spruce section
func TestHome_MultipleCases_Incomplete(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName1 := "Rash"
	caseName2 := "Bed Bugs"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName1,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusOpen,
		},
		{
			ID:             encoding.NewObjectID(2),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName2,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusOpen,
		},
	}

	maDisplayName := "Care Coordinator"
	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
			},
		},
		2: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNIncompleteVisit,
				UID:              CNIncompleteVisit,
				Data: &incompleteVisitNotification{
					PatientVisitID: 1,
				},
			},
		},
		2: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNIncompleteVisit,
				UID:              CNIncompleteVisit,
				Data: &incompleteVisitNotification{
					PatientVisitID: 2,
				},
			},
		},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 4, len(items))

	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	var card phContinueVisit
	test.OK(t, json.Unmarshal(jsonData, &card))
	testContinueVisitCard(t, &card, caseName1, 1, "", "")

	jsonData, err = json.Marshal(items[1])
	test.OK(t, err)
	card = phContinueVisit{}
	test.OK(t, json.Unmarshal(jsonData, &card))
	testContinueVisitCard(t, &card, caseName2, 2, "", "")

	testContactUsSection(t, items[2].(map[string]interface{}))
	testLearnAboutSpruceSection(t, items[3].(map[string]interface{}))
}

// Test home cards when there are multiple incomplete visits
// Expected home cards:
// 1. Case Card with pending TP
// 2. Case card with pending TP
// 3. Refer Spruce card
func TestHome_MultipleCases_TPPending(t *testing.T) {
	dataAPI, addressAPI := setupMockAccessors()

	h := NewHomeHandler(dataAPI, "api.spruce.local", addressAPI)
	r, err := http.NewRequest("GET", "/?zip_code=94115", nil)
	test.OK(t, err)
	setRequestHeaders(r)

	// authenticated
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 1
	ctxt.Role = api.PATIENT_ROLE

	caseName1 := "Rash"
	caseName2 := "Bed Bugs"

	dataAPI.patientCases = []*common.PatientCase{
		{
			ID:             encoding.NewObjectID(1),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName1,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
		{
			ID:             encoding.NewObjectID(2),
			PatientID:      encoding.NewObjectID(2),
			PathwayTag:     "rash",
			Name:           caseName2,
			MedicineBranch: "Dermatology",
			Status:         common.PCStatusActive,
			Claimed:        true,
		},
	}

	maDisplayName := "Care Coordinator"
	doctorDisplayName1 := "Doctor 1"
	doctorDisplayName2 := "Doctor 2"
	dataAPI.careTeamsByCase = map[int64]*common.PatientCareTeam{
		1: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       2,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName1,
					LongDisplayName:  doctorDisplayName1,
				},
			},
		},
		2: &common.PatientCareTeam{
			Assignments: []*common.CareProviderAssignment{
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       1,
					ProviderRole:     api.MA_ROLE,
					ShortDisplayName: maDisplayName,
					LongDisplayName:  maDisplayName,
				},
				{
					Status:           api.STATUS_ACTIVE,
					ProviderID:       3,
					ProviderRole:     api.DOCTOR_ROLE,
					ShortDisplayName: doctorDisplayName2,
					LongDisplayName:  doctorDisplayName2,
				},
			},
		},
	}

	dataAPI.caseNotifications = map[int64][]*common.CaseNotification{
		1: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    1,
				NotificationType: CNTreatmentPlan,
				UID:              CNTreatmentPlan,
				Data: &treatmentPlanNotification{
					MessageID:       1,
					DoctorID:        1,
					TreatmentPlanID: 2,
					CaseID:          1,
				},
			},
		},
		2: []*common.CaseNotification{
			{
				ID:               1,
				PatientCaseID:    2,
				NotificationType: CNTreatmentPlan,
				UID:              CNTreatmentPlan,
				Data: &treatmentPlanNotification{
					MessageID:       2,
					DoctorID:        3,
					TreatmentPlanID: 3,
					CaseID:          2,
				},
			},
		},
	}

	dataAPI.treatmentPlans = []*common.TreatmentPlan{
		{},
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	items := jsonMap["items"].([]interface{})
	test.Equals(t, 3, len(items))

	testCaseCard(t, items[0].(map[string]interface{}), dataAPI.patientCases[0], fmt.Sprintf("With %s", doctorDisplayName1))
	testCaseCard(t, items[1].(map[string]interface{}), dataAPI.patientCases[1], fmt.Sprintf("With %s", doctorDisplayName2))
	testShareSpruceSection(t, items[2].(map[string]interface{}))
}

func setRequestHeaders(r *http.Request) {
	r.Header.Add("S-Version", "Patient;test;1.1.0")
	r.Header.Add("S-OS", "iOS;7.1")
	r.Header.Add("S-Device", "iPhone6,1")
	r.Header.Add("S-Device-ID", "31540817651")
}

func setupMockAccessors() (*mockHomeHandlerDataAPI, *mockHandlerHomeAddressValidationAPI) {
	dataAPI := &mockHomeHandlerDataAPI{
		stateName:       "California",
		isElligible:     false,
		formEntryExists: true,
		pathwayMap: map[string]*common.Pathway{
			api.AcnePathwayTag: &common.Pathway{
				Tag:    api.AcnePathwayTag,
				Name:   "Acne",
				ID:     1,
				Status: common.PathwayActive,
			},
		},
	}

	addressAPI := &mockHandlerHomeAddressValidationAPI{
		lookupFunc: func(zipcode string) (*address.CityState, error) {
			return &address.CityState{
				City:              "San Francisco",
				State:             "California",
				StateAbbreviation: "CA",
			}, nil
		},
	}

	return dataAPI, addressAPI
}

func testUnauthenticatedExperience(t *testing.T, w *httptest.ResponseRecorder) {
	var jsonMap map[string]interface{}
	test.OK(t, json.NewDecoder(w.Body).Decode(&jsonMap))

	// test the expected number of cards
	items := jsonMap["items"].([]interface{})
	test.Equals(t, 2, len(items))
	test.Equals(t, true, jsonMap["show_action_button"].(bool))

	// test the start card
	var startCard phStartVisit
	jsonData, err := json.Marshal(items[0])
	test.OK(t, err)
	test.OK(t, json.Unmarshal(jsonData, &startCard))
	testStartVisitCard(t, &startCard)
	testLearnAboutSpruceSection(t, items[1].(map[string]interface{}))
}

func testCaseCard(t *testing.T, caseCard map[string]interface{}, patientCase *common.PatientCase, subtitle string) {
	test.Equals(t, fmt.Sprintf("%s Case", patientCase.Name), caseCard["title"].(string))
	test.Equals(t, subtitle, caseCard["subtitle"].(string))
	test.Equals(t, app_url.ViewCaseAction(patientCase.ID.Int64()).String(), caseCard["action_url"].(string))
	test.Equals(t, "patient_home:case_view", caseCard["type"].(string))
}

func testNotifyMeConfirmationCard(t *testing.T, card *phHeroIconView, state string) {
	test.OK(t, card.Validate())
	test.Equals(t, "patient_home:hero_icon_view", card.Type)
	test.Equals(t, fmt.Sprintf("We'll notify you when Spruce is available in %s.", state), card.Title)
	test.Equals(t, app_url.IconTickLarge.String(), card.IconURL.String())
}

func testNotifyMeCard(t *testing.T, notifyMeCard *phNotifyMeView, state string) {
	test.OK(t, notifyMeCard.Validate())
	test.Equals(t, "patient_home:notify_me", notifyMeCard.Type)
	test.Equals(t, fmt.Sprintf("Sign up to be notified when Spruce is available in %s.", state), notifyMeCard.Title)
	test.Equals(t, "Your Email Address", notifyMeCard.Placeholder)
	test.Equals(t, app_url.NotifyMeAction().String(), notifyMeCard.ActionURL.String())
	test.Equals(t, "Sign Up", notifyMeCard.ButtonTitle)
}

func testContactUsSection(t *testing.T, sectionViewMap map[string]interface{}) {
	test.Equals(t, "Have a question or need help?", sectionViewMap["title"].(string))

	jsonData, err := json.Marshal(sectionViewMap["views"])
	test.OK(t, err)

	var cards []*phSmallIconText
	test.OK(t, json.Unmarshal(jsonData, &cards))
	test.Equals(t, 1, len(cards))
	test.Equals(t, "Contact Spruce", cards[0].Title)
	test.Equals(t, app_url.IconSupport.String(), cards[0].IconURL.String())
	test.Equals(t, app_url.ViewSupportAction().String(), cards[0].ActionURL)
	test.Equals(t, true, cards[0].RoundedIcon)
}

func testShareSpruceSection(t *testing.T, sectionViewMap map[string]interface{}) {
	test.Equals(t, "Refer a friend to Spruce", sectionViewMap["title"])

	jsonData, err := json.Marshal(sectionViewMap["views"])
	test.OK(t, err)
	var cards []phSmallIconText
	test.OK(t, json.Unmarshal(jsonData, &cards))
	test.Equals(t, 1, len(cards))
	test.OK(t, cards[0].Validate())
	test.Equals(t, app_url.ViewReferFriendAction().String(), cards[0].ActionURL)
	// NOTE: Intentionally not checking the the referral text as that is dynamic and can change over time
}

func testStartVisitCard(t *testing.T, startCard *phStartVisit) {
	test.OK(t, startCard.Validate())
	test.Equals(t, "patient_home:start_visit", startCard.Type)
	test.Equals(t, 4, len(startCard.ImageURLs))
	test.Equals(t, "Start Your First Visit", startCard.Title)
	test.Equals(t, "Receive an effective, personalized treatment plan from a dermatologist in less than 24 hours.", startCard.Description)
	test.Equals(t, "Get Started", startCard.ButtonTitle)
	test.Equals(t, app_url.StartVisitAction().String(), startCard.ActionURL.String())
}

func testContinueVisitCard(t *testing.T, card *phContinueVisit, caseName string, patientVisitID int64, doctorThumbnailURL, doctorShortDisplayName string) {
	test.OK(t, card.Validate())
	test.Equals(t, "patient_home:continue_visit", card.Type)
	test.Equals(t, fmt.Sprintf("Continue Your %s Visit", caseName), card.Title)
	test.Equals(t, "You're Almost There!", card.Subtitle)

	if doctorShortDisplayName != "" {
		test.Equals(t, fmt.Sprintf("Complete your visit and get a personalized treatment plan from %s.", doctorShortDisplayName), card.Description)
	} else {
		test.Equals(t, "Complete your visit and get a personalized treatment plan from your doctor.", card.Description)
	}

	test.Equals(t, "Continue Visit", card.ButtonTitle)
	if doctorThumbnailURL == "" {
		test.Equals(t, app_url.IconCaseLarge.String(), card.IconURL)
	} else {
		test.Equals(t, doctorThumbnailURL, card.IconURL)
	}
	test.Equals(t, app_url.ContinueVisitAction(patientVisitID).String(), card.ActionURL.String())
}

func testMeetCareTeamSection(t *testing.T, caseName string, sectionViewMap map[string]interface{}) {
	test.Equals(t, fmt.Sprintf("Meet your %s care team", caseName), sectionViewMap["title"])

	// parse out the careProviderView
	sectionViewMap = intifyEpochFloatsInInterfaceMap(sectionViewMap)
	jsonData, err := json.Marshal(sectionViewMap["views"])
	test.OK(t, err)

	var cards []*phCareProviderView
	test.OK(t, json.Unmarshal(jsonData, &cards))
	test.Equals(t, 2, len(cards))
	test.Equals(t, true, cards[0].CareProvider.LongDisplayName != "")
	test.Equals(t, true, cards[1].CareProvider.LongDisplayName != "")
}

// HACK
func intifyEpochFloatsInInterfaceMap(r map[string]interface{}) map[string]interface{} {
	for k, v := range r {
		switch v.(type) {
		case float64:
			if strings.Contains(k, "epoch") {
				r[k] = int64(v.(float64))
			}
		case map[string]interface{}:
			r[k] = intifyEpochFloatsInInterfaceMap(v.(map[string]interface{}))
		case []interface{}:
			r[k] = intifyEpochFloatsInInterfaceSlice(v.([]interface{}))
		}
	}
	return r
}

// Hack
func intifyEpochFloatsInInterfaceSlice(s []interface{}) []interface{} {
	for i, v := range s {
		switch v.(type) {
		case map[string]interface{}:
			s[i] = intifyEpochFloatsInInterfaceMap(v.(map[string]interface{}))
		case []interface{}:
			s[i] = intifyEpochFloatsInInterfaceSlice(v.([]interface{}))
		}
	}
	return s
}

func testLearnAboutSpruceSection(t *testing.T, sectionViewMap map[string]interface{}) {
	// test the learn about spruce card
	test.Equals(t, "Learn more about Spruce", sectionViewMap["title"].(string))
	test.Equals(t, "patient_home:section", sectionViewMap["type"])
	jsonData, err := json.Marshal(sectionViewMap["views"])
	test.OK(t, err)
	var sectionItems []*phSmallIconText
	test.OK(t, json.Unmarshal(jsonData, &sectionItems))
	test.Equals(t, 4, len(sectionItems))
	test.Equals(t, "Meet the doctors", sectionItems[0].Title)
	test.Equals(t, app_url.IconSpruceDoctors.String(), sectionItems[0].IconURL.String())
	test.Equals(t, "patient_home:small_icon_text", sectionItems[0].Type)
	test.Equals(t, "What a Spruce visit includes", sectionItems[1].Title)
	test.Equals(t, app_url.IconCaseLarge.String(), sectionItems[1].IconURL.String())
	test.Equals(t, "patient_home:small_icon_text", sectionItems[1].Type)
	test.Equals(t, "See a sample treatment plan", sectionItems[2].Title)
	test.Equals(t, app_url.IconTreatmentPlanLarge.String(), sectionItems[2].IconURL.String())
	test.Equals(t, "patient_home:small_icon_text", sectionItems[2].Type)
	test.Equals(t, "Frequently Asked Questions", sectionItems[3].Title)
	test.Equals(t, app_url.IconFAQ.String(), sectionItems[3].IconURL.String())
	test.Equals(t, "patient_home:small_icon_text", sectionItems[3].Type)
}
