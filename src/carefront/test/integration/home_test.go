package integration

import (
	"carefront/common"
	"testing"
)

func TestHomeAPI(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := "test"
	note := &common.HomeNotification{
		PatientId:       patientId,
		UID:             "note1",
		Expires:         nil,
		Dismissible:     true,
		DismissOnAction: true,
		Priority:        10,
		Type:            "foo",
		Data:            data,
	}
	id, err := testData.DataApi.InsertHomeNotification(note)
	if err != nil {
		t.Fatalf("Failed to insert notification: %s", err.Error())
	}

	notes, err := testData.DataApi.GetHomeNotificationsForPatient(patientId)
	if err != nil {
		t.Fatal(err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification. Got %d", len(notes))
	}

	// Inserting a notification with a duplicate UID for a patient should fail
	_, err = testData.DataApi.InsertHomeNotification(note)
	if err == nil {
		t.Fatal("Duplicate UID for patient should fail")
	}

	// Test delete
	if err := testData.DataApi.DeleteHomeNotification(id); err != nil {
		t.Fatalf("Failed to delete notification: %s", err.Error())
	}
	if notes, err := testData.DataApi.GetHomeNotificationsForPatient(patientId); err != nil {
		t.Fatal(err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notification. Got %d", len(notes))
	}

	// Test delete by UID
	if _, err := testData.DataApi.InsertHomeNotification(note); err != nil {
		t.Fatalf("Failed to insert notification: %s", err.Error())
	}
	if err := testData.DataApi.DeleteHomeNotificationByUID(patientId, note.UID); err != nil {
		t.Fatalf("Failed to delete notification: %s", err.Error())
	}
	if notes, err := testData.DataApi.GetHomeNotificationsForPatient(patientId); err != nil {
		t.Fatal(err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notification. Got %d", len(notes))
	}
}
