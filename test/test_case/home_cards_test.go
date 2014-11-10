package test_case

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestHomeCards_UnAuthenticated(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	items := getHomeCardsForPatient(0, testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected %d items but got %d", 2, len(items))
	}
	ensureStartVisitCard(items[0], t)
	ensureSectionWithNSubViews(4, items[1], t)

	// now lets try with a signed up patient account;
	// should be the same state as above
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	items = getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected %d items but got %d", 2, len(items))
	}

	ensureStartVisitCard(items[0], t)
	ensureSectionWithNSubViews(4, items[1], t)
}

func TestHomeCards_UnavailableState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationAPI.CityStateToReturn = &address.CityState{
		City:              "New York City",
		State:             "New York",
		StateAbbreviation: "NY",
	}

	items := getHomeCardsForPatient(0, testData, t)
	if len(items) != 1 {
		t.Fatalf("Expected %d items but got %d", 2, len(items))
	}
	ensureSectionWithNSubViews(4, items[0], t)
}

func TestHomeCards_IncompleteVisit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)

	items := getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(4, items[2], t)

	// create another patient and ensure that this patient also has the continue card visit
	pr2 := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	test_integration.CreatePatientVisitForPatient(pr2.Patient.PatientId.Int64(), testData, t)
	items = getHomeCardsForPatient(pr2.Patient.AccountId.Int64(), testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(4, items[2], t)

	// now ensure that the first patient's home state is still maintained as expected

	items = getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)

	if len(items) != 3 {
		t.Fatalf("Expected 3 items but got %d instead", len(items))
	}

	ensureContinueVisitCard(items[0], t)
	ensureSectionWithNSubViews(1, items[1], t)
	ensureSectionWithNSubViews(4, items[2], t)
}

func TestHomeCards_VisitSubmitted(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)

	items := getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)

	pr2 := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv2 := test_integration.CreatePatientVisitForPatient(pr2.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr2.Patient.PatientId.Int64(), pv2.PatientVisitId, testData, t)

	// ensure the state of the second patient
	items = getHomeCardsForPatient(pr2.Patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)

	// ensure that the home cards state of the first patient is still intact
	items = getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)
}

func TestHomeCards_NoUpdatesState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	// now get the patient to view the treatment plan
	test_integration.GenerateAppEvent(app_event.ViewedAction, "treatment_plan", tp.Id.Int64(), patient.AccountId.Int64(), testData, t)

	// in this state there should be no updates, which means that there should be the buttons notification view
	items := getHomeCardsForPatient(patient.AccountId.Int64(), testData, t)
	test.Equals(t, 2, len(items))

	cView := items[0].(map[string]interface{})
	test.Equals(t, "patient_home:case_view", cView["type"])

	nView := cView["notification_view"].(map[string]interface{})
	test.Equals(t, "patient_home_case_notification:buttons", nView["type"])
	test.Equals(t, 3, len(nView["buttons"].([]interface{})))
}

func TestHomeCards_UnsuitableState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	answerIntakeBody := test_integration.PrepareAnswersForDiagnosingAsUnsuitableForSpruce(testData, t, pv.PatientVisitId)
	test_integration.SubmitPatientVisitDiagnosisWithIntake(pv.PatientVisitId, doctor.AccountId.Int64(), answerIntakeBody, testData, t)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	// now lets get the doctor to send a message to the patient
	messageID := test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo",
	})

	// lets get the patient to view it
	test_integration.GenerateAppEvent(app_event.ViewedAction, "case_message", messageID, patient.AccountId.Int64(), testData, t)

	// in this state there should be no updates, which means that there should be the buttons notification view
	items := getHomeCardsForPatient(patient.AccountId.Int64(), testData, t)
	test.Equals(t, 2, len(items))

	cView := items[0].(map[string]interface{})
	test.Equals(t, "patient_home:case_view", cView["type"])

	nView := cView["notification_view"].(map[string]interface{})
	test.Equals(t, "patient_home_case_notification:buttons", nView["type"])
	test.Equals(t, 2, len(nView["buttons"].([]interface{})))
}

func TestHomeCards_MessageFromDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)

	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)
	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, caseID)
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})

	items := getHomeCardsForPatient(pr.Patient.AccountId.Int64(), testData, t)
	if len(items) != 1 {
		t.Fatalf("Expected 2 items but got %d instead", len(items))
	}
	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
}

func TestHomeCards_MessageFromMA(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	// have the MA message the patient
	test_integration.PostCaseMessage(t, testData, ma.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo",
	})

	items := getHomeCardsForPatient(patient.AccountId.Int64(), testData, t)
	test.Equals(t, 1, len(items))

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
}

func TestHomeCards_TreatmentPlanFromDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)

	pv, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)

	items := getHomeCardsForPatient(patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 1 item but got %d", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], false, t)
	ensureSectionWithNSubViews(1, items[1], t)
}

func TestHomeCards_MultipleNotifications(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)

	pv, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)

	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})

	items := getHomeCardsForPatient(patient.AccountId.Int64(), testData, t)
	if len(items) != 2 {
		t.Fatalf("Expected 2 item but got %d", len(items))
	}

	ensureCaseCardWithEmbeddedNotification(items[0], true, t)
}

func getHomeCardsForPatient(accountID int64, testData *test_integration.TestData, t *testing.T) []interface{} {
	responseData := make(map[string]interface{})

	res, err := testData.AuthGet(testData.APIServer.URL+router.PatientHomeURLPath+"?zip_code=94115", accountID)
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
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
