package test_homelog

import (
	"carefront/common"
	"carefront/homelog"
	"carefront/test/test_integration"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type titleSubtitleItem struct {
	Title    string
	Subtitle string
	IconURL  string
	TapURL   string
}

type incompleteVisitNotification struct {
	VisitId int64
}

type visitReviewedNotification struct {
	DoctorId int64
	VisitId  int64
}

type newConversationNotification struct {
	DoctorId       int64
	ConversationId int64
}

type conversationReplyNotification struct {
	DoctorId       int64
	ConversationId int64
}

func (*titleSubtitleItem) TypeName() string {
	return "title_subtitle"
}

func (*incompleteVisitNotification) TypeName() string {
	return "incomplete_visit"
}

func (*visitReviewedNotification) TypeName() string {
	return "visit_reviewed"
}

func (*newConversationNotification) TypeName() string {
	return "new_conversation"
}

func (*conversationReplyNotification) TypeName() string {
	return "conversation_reply"
}

var notificationTypes = map[string]reflect.Type{
	(&titleSubtitleItem{}).TypeName():             reflect.TypeOf(titleSubtitleItem{}),
	(&incompleteVisitNotification{}).TypeName():   reflect.TypeOf(incompleteVisitNotification{}),
	(&visitReviewedNotification{}).TypeName():     reflect.TypeOf(visitReviewedNotification{}),
	(&newConversationNotification{}).TypeName():   reflect.TypeOf(newConversationNotification{}),
	(&conversationReplyNotification{}).TypeName(): reflect.TypeOf(conversationReplyNotification{}),
}

func TestPatientNotificationsAPI(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &titleSubtitleItem{
		Title: "1234",
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
	} else if notes[0].Data.(*titleSubtitleItem).Title != "1234" {
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
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	data := &titleSubtitleItem{
		Title: "4321",
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
	} else if items[0].Data.(*titleSubtitleItem).Title != "4321" {
		t.Fatal("Test item data mismatch")
	}

	// Inserting an item with a duplicate UID should update the item
	data.Title = "9999"
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
	} else if items[0].Data.(*titleSubtitleItem).Title != "9999" {
		t.Fatal("Test item data mismatch")
	}
}

func TestHealthLog(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	visit := test_integration.CreatePatientVisitForPatient(patientId, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientId, visit.PatientVisitId, testData, t)

	if items, _, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes); err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	}

	ts := httptest.NewServer(homelog.NewListHandler(testData.DataApi))
	defer ts.Close()

	resp, err := test_integration.AuthGet(ts.URL, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get home")
	}
	defer resp.Body.Close()

	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf("Failed to get body: %+v", err)
	} else {
		test_integration.CheckSuccessfulStatusCode(resp, "Unable to get home: "+string(body), t)
	}
}

func TestVisitCreatedNotification(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Error getting doctor from id: %s", err.Error())
	}

	visit, _ := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	patientId := patient.PatientId.Int64()
	test_integration.SubmitPatientVisitBackToPatient(visit.PatientVisitId, doctor, testData, t)

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

func TestConversationLogItem(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Error getting doctor from id: %s", err.Error())
	}

	convId := test_integration.StartConversationFromDoctorToPatient(t, testData.DataApi, doctor.AccountId.Int64(), patientId, 0)

	items, _, err := testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*titleSubtitleItem).Subtitle != "1 message" {
		t.Fatalf("Test item subtitle mismatch: %s", items[0].Data.(*titleSubtitleItem).Subtitle)
	}
	firstItem := items[0]

	// Make sure time ticks so that comparing the timestamps is stable
	time.Sleep(time.Second)

	// Make sure a reply updates the log item
	test_integration.PatientReplyToConversation(t, testData.DataApi, convId, patient.AccountId.Int64())

	items, _, err = testData.DataApi.GetHealthLogForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatal(err)
	} else if len(items) != 1 {
		t.Fatalf("Expected 1 item. Got %d", len(items))
	} else if items[0].Data.TypeName() != "title_subtitle" {
		t.Fatalf("Expected data type of 'title_subtitle'. Got '%s'", items[0].Data.TypeName())
	} else if items[0].Data.(*titleSubtitleItem).Subtitle != "2 messages" {
		t.Fatalf("Test item subtitle mismatch: %s", items[0].Data.(*titleSubtitleItem).Subtitle)
	} else if items[0].Timestamp.Sub(firstItem.Timestamp) == 0 {
		t.Fatalf("Timestamp not updated")
	}
}

func TestConversationNotifications(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Error getting doctor from id: %s", err.Error())
	}

	// New conversation from doctor to patient MUST create a notification

	convId := test_integration.StartConversationFromDoctorToPatient(t, testData.DataApi, doctor.AccountId.Int64(), patientId, 0)

	notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification for patient instead got %d", len(notes))
	} else if notes[0].Data.TypeName() != "new_conversation" {
		t.Fatalf("Expected notification of type %s instead got %s", "new_conversation", notes[0].Data.TypeName())
	}

	// Reply from patient to doctor MUST clear the original notification

	test_integration.PatientReplyToConversation(t, testData.DataApi, convId, patient.AccountId.Int64())

	notes, _, err = testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notifications for patient instead got %d", len(notes))
	}

	// Reply from doctor to patient MUST create a notification

	test_integration.DoctorReplyToConversation(t, testData.DataApi, convId, doctor.AccountId.Int64())

	notes, _, err = testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification for patient instead got %d", len(notes))
	} else if notes[0].Data.TypeName() != "conversation_reply" {
		t.Fatalf("Expected notification of type %s instead got %s", "conversation_reply", notes[0].Data.TypeName())
	}
	if err := testData.DataApi.DeletePatientNotifications([]int64{notes[0].Id}); err != nil {
		t.Fatalf("Failed to delete notification: %s", err.Error())
	}

	// New conversation from patient to doctor MUST NOT create a notification

	test_integration.StartConversationFromPatientToDoctor(t, testData.DataApi, patient.AccountId.Int64(), 0)

	notes, _, err = testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 0 {
		t.Fatalf("Expected 0 notifications for patient instead got %d", len(notes))
	}
}
