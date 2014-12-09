package test_case

import (
	"testing"

	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseNotifications_IncompleteVisit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)

	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	// there should exist 1 notification to indicate an incomplete visit
	testNotifyTypes := getNotificationTypes()

	notificationItems, err := testData.DataAPI.GetNotificationsForCase(patientCase.ID.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNIncompleteVisit {
		t.Fatalf("Expected %s but got %s", patient_case.CNIncompleteVisit, notificationItems[0].NotificationType)
	}
}

func TestCaseNotifications_VisitSubmitted(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// there should exist 1 notification at this point to indicate to the patient that they have
	// submitted their visit
	testNotifyTypes := getNotificationTypes()

	notificationItems, err := testData.DataAPI.GetNotificationsForCase(treatmentPlan.PatientCaseID.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNVisitSubmitted {
		t.Fatalf("Expected %s but got %s", patient_case.CNVisitSubmitted, notificationItems[0].NotificationType)
	}
}

// This test is to ensure that the right interactions take place
// pertaining to case messages and their corresponding notifications
func TestCaseNotifications_Message(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	visit, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	caseID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, doctorID)
	patientCli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	messageId1, err := doctorCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)

	testNotifyTypes := getNotificationTypes()

	// there should exist a notification for the patient case
	notificationItems, err := testData.DataAPI.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNMessage {
		t.Fatalf("Expected notification to be of type %s instead got %s", patient_case.CNMessage, notificationItems[0].NotificationType)
	}

	// if the patient messages the doctor there should be no impact on the patient case notifications
	_, err = patientCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)

	notificationItems, err = testData.DataAPI.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	}

	// if the doctor sends the patient another message there should be 2 remaining case notifications
	messageId2, err := doctorCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)

	notificationItems, err = testData.DataAPI.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 2 {
		t.Fatalf("Expected %d notification items instead got %d", 2, len(notificationItems))
	}

	notificationId := notificationItems[1].ID

	// now lets go ahead and have the patient read the message
	test_integration.GenerateAppEvent(app_event.ViewedAction, "case_message", messageId2, patient.AccountID.Int64(), testData, t)

	// there should only remain 1 notification item
	notificationItems, err = testData.DataAPI.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].ID == notificationId {
		t.Fatalf("Expected remaining notification item to have different notification id than the item just dismissed")
	}

	// read the remaining message
	test_integration.GenerateAppEvent(app_event.ViewedAction, "case_message", messageId1, patient.AccountID.Int64(), testData, t)
	notificationItems, err = testData.DataAPI.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 0 {
		t.Fatalf("Expected %d notification items instead got %d", 0, len(notificationItems))
	}
}

func TestCaseNotifications_MessageFromMA(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	maCli := test_integration.DoctorClient(testData, t, ma.DoctorID.Int64())

	// have the MA message the patient
	_, err = maCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)

	testNotifyTypes := getNotificationTypes()

	// there should exist a notification for the patient case
	notificationItems, err := testData.DataAPI.GetNotificationsForCase(tp.PatientCaseID.Int64(), testNotifyTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(notificationItems))
	test.Equals(t, patient_case.CNMessage, notificationItems[0].NotificationType)
}

func TestCaseNotifications_TreatmentPlan(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	patient, err := testData.DataAPI.GetPatientFromTreatmentPlanID(treatmentPlan.ID.Int64())
	test.OK(t, err)

	// there should now exist a notification item for the treatment plan
	testNotifyTypes := getNotificationTypes()

	notificationItems, err := testData.DataAPI.GetNotificationsForCase(treatmentPlan.PatientCaseID.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNTreatmentPlan {
		t.Fatalf("Expected notification to be of type %s instead got %s", patient_case.CNTreatmentPlan, notificationItems[0].NotificationType)
	}

	// now lets go ahead and mark that the patient read the treatment plan
	test_integration.GenerateAppEvent(app_event.ViewedAction, "treatment_plan", treatmentPlan.ID.Int64(), patient.AccountID.Int64(), testData, t)

	// now there should be no treatment plan notificatin left
	notificationItems, err = testData.DataAPI.GetNotificationsForCase(treatmentPlan.PatientCaseID.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 0 {
		t.Fatalf("Expected %d notification items instead got %d", 0, len(notificationItems))
	}
}
