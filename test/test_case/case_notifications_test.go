package test_case

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/treatment_plan"
)

func TestCaseNotifications_VisitSubmitted(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// there should exist 1 notification at this point to indicate to the patient that they have
	// submiited their visit
	testNotifyTypes := getNotificationTypes()

	notificationItems, err := testData.DataApi.GetNotificationsForCase(treatmentPlan.PatientCaseId.Int64(), testNotifyTypes)
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
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	visit, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(visit.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})

	testNotifyTypes := getNotificationTypes()

	// there should exist a notification for the patient case
	notificationItems, err := testData.DataApi.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNMessage {
		t.Fatalf("Expected notification to be of type %s instead got %s", patient_case.CNMessage, notificationItems[0].NotificationType)
	}

	// if the patient messages the doctor there should be no impact on the patient case notifications
	test_integration.PostCaseMessage(t, testData, patient.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})
	notificationItems, err = testData.DataApi.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	}

	// if the doctor sends the patient another message there should be 2 remaining case notifications
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "foo",
	})
	notificationItems, err = testData.DataApi.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 2 {
		t.Fatalf("Expected %d notification items instead got %d", 2, len(notificationItems))
	}

	// now lets go ahead and dismiss 1 case notification
	notificationId := notificationItems[0].Id
	DismissCaseNotification(notificationId, patient.AccountId.Int64(), testData, t)

	// there should only remain 1 notification item
	notificationItems, err = testData.DataApi.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].Id == notificationId {
		t.Fatalf("Expected remaining notification item to have different notification id than the item just dismissed")
	}

	// dismiss the last remaining notification item
	DismissCaseNotification(notificationItems[0].Id, patient.AccountId.Int64(), testData, t)
	notificationItems, err = testData.DataApi.GetNotificationsForCase(caseID, testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 0 {
		t.Fatalf("Expected %d notification items instead got %d", 0, len(notificationItems))
	}
}

func TestCaseMessage_TreatmentPlan(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	patient, err := testData.DataApi.GetPatientFromTreatmentPlanId(treatmentPlan.Id.Int64())
	if err != nil {
		t.Fatal(err)
	}

	// there should now exist a notification item for the treatment plan
	testNotifyTypes := getNotificationTypes()

	notificationItems, err := testData.DataApi.GetNotificationsForCase(treatmentPlan.PatientCaseId.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 1 {
		t.Fatalf("Expected %d notification items instead got %d", 1, len(notificationItems))
	} else if notificationItems[0].NotificationType != patient_case.CNTreatmentPlan {
		t.Fatalf("Expected notification to be of type %s instead got %s", patient_case.CNTreatmentPlan, notificationItems[0].NotificationType)
	}

	// now lets go ahead and open the treatment plan for viewing
	tpHandler := treatment_plan.NewTreatmentPlanHandler(testData.DataApi)
	patientServer := httptest.NewServer(tpHandler)
	defer patientServer.Close()

	res, err := testData.AuthGet(patientServer.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), patient.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	// now there should be no treatment plan notificatin left
	notificationItems, err = testData.DataApi.GetNotificationsForCase(treatmentPlan.PatientCaseId.Int64(), testNotifyTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notificationItems) != 0 {
		t.Fatalf("Expected %d notification items instead got %d", 0, len(notificationItems))
	}
}
