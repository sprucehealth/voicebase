package test_case

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestHomeCards_UnAuthenticated(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	items := getHomeCardsForPatient("", testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected %d items but got %d", 2, len(items))
	}
	ensureStartVisitCard(items[0], t)
	ensureSectionWithNSubViews(3, items[1], t)

	// now lets try with a signed up patient account;
	// should be the same state as above
	pr := test_integration.SignupRandomTestPatient(t, testData)

	items = getHomeCardsForPatient(pr.Token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected %d items but got %d", 2, len(items))
	}

	ensureStartVisitCard(items[0], t)
	ensureSectionWithNSubViews(3, items[1], t)
}

func TestHomeCards_UnavailableState(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "New York City",
			State:             "New York",
			StateAbbreviation: "NY",
		},
	}

	homeHandler := patient_case.NewHomeHandler(testData.DataApi, testData.AuthApi, stubAddressValidationService)
	patientServer := httptest.NewServer(homeHandler)
	defer patientServer.Close()

	responseData := make(map[string]interface{})

	getRequest, err := http.NewRequest("GET", patientServer.URL+"?zip_code=94105", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(getRequest)
	defer res.Body.Close()

	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatal(err)
	}

	items := responseData["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("Expected %d items but got %d", 1, len(items))
	}
	ensureSectionWithNSubViews(3, items[0], t)
}

func TestHomeCards_IncompleteVisit(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	pr := test_integration.SignupRandomTestPatient(t, testData)
	test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)

	items := getHomeCardsForPatient(pr.Token, testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(3, items[2], t)

	// create another patient and ensure that this patient also has the continue card visit
	pr2 := test_integration.SignupRandomTestPatient(t, testData)
	test_integration.CreatePatientVisitForPatient(pr2.Patient.PatientId.Int64(), testData, t)
	items = getHomeCardsForPatient(pr2.Token, testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(3, items[2], t)

	// now ensure that the first patient's home state is still maintained as expected

	items = getHomeCardsForPatient(pr.Token, testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(3, items[2], t)
}

func TestHomeCards_VisitSubmitted(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	pr := test_integration.SignupRandomTestPatient(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)

	items := getHomeCardsForPatient(pr.Token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)

	pr2 := test_integration.SignupRandomTestPatient(t, testData)
	pv2 := test_integration.CreatePatientVisitForPatient(pr2.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr2.Patient.PatientId.Int64(), pv2.PatientVisitId, testData, t)

	// ensure the state of the second patient
	items = getHomeCardsForPatient(pr2.Token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)

	// ensure that the home cards state of the first patient is still intact
	items = getHomeCardsForPatient(pr.Token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)

}

func TestHomeCards_MessageFromDoctor(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	pr := test_integration.SignupRandomTestPatient(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)
	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, caseID)
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})

	items := getHomeCardsForPatient(pr.Token, testData, t)
	if len(items) != 1 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}
	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
}

func TestHomeCards_TreatmentPlanFromDoctor(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	pv, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	_, token, err := testData.AuthApi.LogIn(patient.Email, "12345")
	if err != nil {
		t.Fatal(err)
	}

	items := getHomeCardsForPatient(token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 1 item but got %d", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)
}

func TestHomeCards_MultipleNotifications(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	pv, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})

	_, token, err := testData.AuthApi.LogIn(patient.Email, "12345")
	if err != nil {
		t.Fatal(err)
	}

	items := getHomeCardsForPatient(token, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 item but got %d", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], true, t)
}

func getHomeCardsForPatient(token string, testData *test_integration.TestData, t *testing.T) []interface{} {

	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	homeHandler := patient_case.NewHomeHandler(testData.DataApi, testData.AuthApi, stubAddressValidationService)
	patientServer := httptest.NewServer(homeHandler)
	defer patientServer.Close()

	responseData := make(map[string]interface{})

	getRequest, err := http.NewRequest("GET", patientServer.URL+"?zip_code=94115", nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		getRequest.Header.Set("Authorization", "token "+token)
	}

	res, err := http.DefaultClient.Do(getRequest)
	defer res.Body.Close()

	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatal(err)
	}

	return responseData["items"].([]interface{})
}

func ensureStartVisitCard(clientView interface{}, t *testing.T) {
	cView := clientView.(map[string]interface{})
	if cView["type"] != "patient_home:start_visit" {
		t.Fatalf("Expected type of card to be start_visit but it was %s", cView["type"])
	}
}

func ensureContinueVisitCard(clientView interface{}, t *testing.T) {
	cView := clientView.(map[string]interface{})
	if cView["type"] != "patient_home:continue_visit" {
		t.Fatalf("Expected type of card to be start_visit but it was %s", cView["type"])
	}
}

func ensureCaseCardWithEmbeddedNotification(clientView interface{}, multipleNotification bool, t *testing.T) {
	cView := clientView.(map[string]interface{})
	if cView["type"] != "patient_home:case_view" {
		t.Fatalf("Expected type of card to be start_visit but it was %s", cView["type"])
	}

	nView := cView["notification_view"].(map[string]interface{})

	viewType := "patient_home_case_notification:standard"
	if multipleNotification {
		viewType = "patient_home_case_notification:multiple"
	}

	if nView["type"] != viewType {
		t.Fatalf("Expected type of card to be %s:standard but was %s", viewType, nView["type"])
	}
}

func ensureVisitCaseCardOnly(clientView interface{}, t *testing.T) {
	cView := clientView.(map[string]interface{})
	if cView["type"] != "patient_home:case_view" {
		t.Fatalf("Expected type of card to be start_visit but it was %s", cView["type"])
	}

	if cView["notification_view"] != nil {
		t.Fatal("Expected no notification to be embedded in the case card")
	}
}

func ensureSectionWithNSubViews(numSubViews int, clientView interface{}, t *testing.T) {
	cView := clientView.(map[string]interface{})
	if cView["type"] != "patient_home:section" {
		t.Fatalf("Expected section but got %s", cView["type"])
	}

	subViews := cView["views"].([]interface{})
	if len(subViews) != numSubViews {
		t.Fatalf("Expected %d items in the learn about spruce section but got %d", numSubViews, len(subViews))
	}
}
