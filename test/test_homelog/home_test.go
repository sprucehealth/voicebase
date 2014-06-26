package test_homelog

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/homelog"
	"github.com/sprucehealth/backend/test/test_integration"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"testing"
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

type treatmentPlanCreatedNotification struct {
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

func (*treatmentPlanCreatedNotification) TypeName() string {
	return "treatment_plan_created"
}

func (*newConversationNotification) TypeName() string {
	return "new_conversation"
}

func (*conversationReplyNotification) TypeName() string {
	return "conversation_reply"
}

var notificationTypes = map[string]reflect.Type{
	(&titleSubtitleItem{}).TypeName():                reflect.TypeOf(titleSubtitleItem{}),
	(&incompleteVisitNotification{}).TypeName():      reflect.TypeOf(incompleteVisitNotification{}),
	(&treatmentPlanCreatedNotification{}).TypeName(): reflect.TypeOf(treatmentPlanCreatedNotification{}),
	(&newConversationNotification{}).TypeName():      reflect.TypeOf(newConversationNotification{}),
	(&conversationReplyNotification{}).TypeName():    reflect.TypeOf(conversationReplyNotification{}),
}

func TestPatientNotificationsAPI(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData)
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

	pr := test_integration.SignupRandomTestPatient(t, testData)
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

	pr := test_integration.SignupRandomTestPatient(t, testData)
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

	resp, err := testData.AuthGet(ts.URL, patient.AccountId.Int64())
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

func TestTreatmentPlanCreatedNotification(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Error getting doctor from id: %s", err.Error())
	}

	visit, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	patientId := patient.PatientId.Int64()
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// make a call to get patient notifications
	listNotificationsHandler := homelog.NewListHandler(testData.DataApi)
	ts := httptest.NewServer(listNotificationsHandler)
	defer ts.Close()

	notes, _, err := testData.DataApi.GetNotificationsForPatient(patientId, notificationTypes)
	if err != nil {
		t.Fatalf("Unable to get notifications for patient %s", err)
	} else if len(notes) != 1 {
		t.Fatalf("Expected 1 notification for patient instead got %d", len(notes))
	} else if notes[0].Data.TypeName() != "treatment_plan_created" {
		t.Fatalf("Expected notification of type %s instead got %s", "visit_reviewed", notes[0].Data.TypeName())
	}
}
