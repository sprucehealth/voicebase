package integration

import (
	"carefront/common"
	"carefront/homelog"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"testing"
)

type titleSubtitleItem struct {
	SomeId int64
}

func (*titleSubtitleItem) TypeName() string {
	return "title_subtitle"
}

type visitReviewedNotification struct {
	SomeId int64
}

func (*visitReviewedNotification) TypeName() string {
	return "visit_reviewed"
}

var notificationTypes = map[string]reflect.Type{
	(&titleSubtitleItem{}).TypeName():         reflect.TypeOf(titleSubtitleItem{}),
	(&visitReviewedNotification{}).TypeName(): reflect.TypeOf(visitReviewedNotification{}),
}

func TestPatientNotificationsAPI(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &titleSubtitleItem{
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

	notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification. Got %d", len(notes))
	} else if notes[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", notes[0].Data.TypeName())
	} else if notes[0].Data.(*titleSubtitleItem).SomeId != 1234 {
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
	if notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes); err != nil {
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
	if notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notification. Got %d", len(notes))
	}
}

func TestHealthLogAPI(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &titleSubtitleItem{
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

	if items, _, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*titleSubtitleItem).SomeId != 1234 {
		t.Fatal("Test item data mismatch")
	}

	// Inserting an item with a duplicate UID should update the item
	data.SomeId = 9999
	item.Data = data
	_, err = testData.DataApi.InsertOrUpdatePatientHealthLogItem(patientId, item)
	if err != nil {
		t.Fatalf("Failed to update log item: %s", err.Error())
	}
	if items, _, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*titleSubtitleItem).SomeId != 9999 {
		t.Fatal("Test item data mismatch")
	}
}

func TestHealthLog(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	visit := createPatientVisitForPatient(patientId, testData, t)
	submitPatientVisitForPatient(patientId, visit.PatientVisitId, testData, t)

	if items, _, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	}

	ts := httptest.NewServer(homelog.NewListHandler(testData.DataApi))
	defer ts.Close()

	resp, err := authGet(ts.URL, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get home")
	}
	defer resp.Body.Close()

	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf("Failed to get body: %+v", err)
	} else {
		CheckSuccessfulStatusCode(resp, "Unable to get home: "+string(body), t)
	}
}

func TestVisitCreatedNotification(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Error getting doctor from id: %s", err.Error())
	}

	visit := createPatientVisitForPatient(patientId, testData, t)
	submitPatientVisitForPatient(patientId, visit.PatientVisitId, testData, t)
	startReviewingPatientVisit(visit.PatientVisitId, doctor, testData, t)
	submitPatientVisitBackToPatient(visit.PatientVisitId, doctor, testData, t)

	// make a call to get patient notifications
	listNotificationsHandler := homelog.NewListHandler(testData.DataApi)
	ts := httptest.NewServer(listNotificationsHandler)
	defer ts.Close()

	notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification for patient instead got %d", len(notes))
	} else if notes[0].Data.TypeName() != "visit_reviewed" {
		t.Fatalf("Expected notification of type %s instead got %s", "visit_reviewed", notes[0].Data.TypeName())
	}

}
