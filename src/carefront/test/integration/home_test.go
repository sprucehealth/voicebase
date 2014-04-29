package integration

import (
	"carefront/common"
	"reflect"
	"testing"
)

type testNotification struct {
	SomeId int64
}

func (*testNotification) TypeName() string {
	return "test"
}

var notificationTypes = map[string]reflect.Type{
	(&testNotification{}).TypeName(): reflect.TypeOf(testNotification{}),
}

func TestPatientNotifications(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &testNotification{
		SomeId: 1234,
	}
	note := &common.Notification{
		UID:             "note1",
		Expires:         nil,
		Dismissible:     true,
		DismissOnAction: true,
		Priority:        10,
		Data:            data,
	}
	id, err := testData.DataApi.InsertPatientNotification(patientId, note)
	if err != nil {
		t.Fatalf("Failed to insert notification: %s", err.Error())
	}

	notes, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification. Got %d", len(notes))
	} else if notes[0].Data.TypeName() != "test" {
		t.Fatalf("Expected data type of 'test'. Got '%s'", notes[0].Data.TypeName())
	} else if notes[0].Data.(*testNotification).SomeId != 1234 {
		t.Fatal("Test notification data mismatch")
	}

	// Inserting a notification with a duplicate UID for a patient should fail
	_, err = testData.DataApi.InsertPatientNotification(patientId, note)
	if err == nil {
		t.Fatal("Duplicate UID for patient should fail")
	}

	// Test delete
	if err := testData.DataApi.DeletePatientNotifications([]int64{id}); err != nil {
		t.Fatalf("Failed to delete notification: %s", err.Error())
	}
	if notes, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notification. Got %d", len(notes))
	}

	// Test delete by UID
	if _, err := testData.DataApi.InsertPatientNotification(patientId, note); err != nil {
		t.Fatalf("Failed to insert notification: %s", err.Error())
	}
	if err := testData.DataApi.DeletePatientNotificationByUID(patientId, note.UID); err != nil {
		t.Fatalf("Failed to delete notification: %s", err.Error())
	}
	if notes, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notification. Got %d", len(notes))
	}
}

func TestHealthLog(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &testNotification{
		SomeId: 1234,
	}
	item := &common.HealthLogItem{
		UID:  "item1",
		Data: data,
	}
	_, err := testData.DataApi.InsertOrUpdatePatientHealthLogItem(patientId, item)
	if err != nil {
		t.Fatalf("Failed to insert item: %s", err.Error())
	}

	if items, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "test" {
		t.Fatalf("Expected data type of 'test'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*testNotification).SomeId != 1234 {
		t.Fatal("Test item data mismatch")
	}

	// Inserting an item with a duplicate UID should update the item
	data.SomeId = 9999
	item.Data = data
	_, err = testData.DataApi.InsertOrUpdatePatientHealthLogItem(patientId, item)
	if err != nil {
		t.Fatalf("Failed to update log item: %s", err.Error())
	}
	if items, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "test" {
		t.Fatalf("Expected data type of 'test'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*testNotification).SomeId != 9999 {
		t.Fatal("Test item data mismatch")
	}
}
