package doctor_queue

import (
	"bytes"
	"carefront/api"
	"carefront/messages"
	"carefront/test/integration"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestConversationItemsInDoctorQueue(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer integration.TearDownIntegrationTest(t, testData)

	pr := integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}

	patientConvoServer := httptest.NewServer(messages.NewPatientConversationHandler(testData.DataApi))
	defer patientConvoServer.Close()
	doctorMessageServer := httptest.NewServer(messages.NewDoctorMessagesHandler(testData.DataApi))
	defer doctorMessageServer.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.NewConversationRequest{
		TopicId: topicId,
		Message: "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := integration.AuthPost(patientConvoServer.URL, "application/json", body, pr.Patient.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	newConvRes := &messages.NewConversationResponse{}
	if err := json.NewDecoder(res.Body).Decode(newConvRes); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	doctorId := integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that an item is inserted into the doctor queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in the pending items but got %d instead", len(pendingItems))
	} else if pendingItems[0].EventType != api.EVENT_TYPE_CONVERSATION {
		t.Fatalf("Expected item type to be %s instead it was %s", api.EVENT_TYPE_CONVERSATION, pendingItems[0].EventType)
	}

	// Reply
	body = &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.ReplyRequest{
		ConversationId: newConvRes.ConversationId,
		Message:        "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err = integration.AuthPost(doctorMessageServer.URL, "application/json", body, doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the item is marked as completed for the doctor
	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor queue: %s", err)
	} else if len(pendingItems) != 0 {
		t.Fatalf("Expected no item in the pending items but got %d instead", len(pendingItems))
	}

	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctorId)
	if err != nil {
		t.Fatalf("Unable to get completed items in the doctor queue: %s", err)
	} else if len(completedItems) != 1 {
		t.Fatalf("Expected 1 item in the completed tab instead got %d", len(completedItems))
	} else if completedItems[0].EventType != api.EVENT_TYPE_CONVERSATION {
		t.Fatalf("Expected item of type %s instead got %s", api.EVENT_TYPE_CONVERSATION, completedItems[0].EventType)
	}
}
