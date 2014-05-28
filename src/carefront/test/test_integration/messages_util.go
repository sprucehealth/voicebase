package test_integration

import (
	"bytes"
	"carefront/api"
	"carefront/messages"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func StartConversationFromDoctorToPatient(t *testing.T, dataAPI api.DataAPI, doctorAccountId, patientId, topicId int64) int64 {
	if topicId == 0 {
		id, err := dataAPI.AddConversationTopic("Foo", 100, true)
		if err != nil {
			t.Fatal(err)
		}
		topicId = id
	}

	doctorConvoServer := httptest.NewServer(messages.NewDoctorConversationHandler(dataAPI))
	defer doctorConvoServer.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.NewConversationRequest{
		PatientId: patientId,
		TopicId:   topicId,
		Message:   "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := AuthPost(doctorConvoServer.URL, "application/json", body, doctorAccountId)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
	newConvRes := &messages.NewConversationResponse{}
	if err := json.NewDecoder(res.Body).Decode(newConvRes); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return newConvRes.ConversationId
}

func StartConversationFromPatientToDoctor(t *testing.T, dataAPI api.DataAPI, patientAccountId, topicId int64) int64 {
	if topicId == 0 {
		id, err := dataAPI.AddConversationTopic("Foo", 100, true)
		if err != nil {
			t.Fatal(err)
		}
		topicId = id
	}

	patientConvoHandler := httptest.NewServer(messages.NewPatientConversationHandler(dataAPI))
	defer patientConvoHandler.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.NewConversationRequest{
		TopicId: topicId,
		Message: "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := AuthPost(patientConvoHandler.URL, "application/json", body, patientAccountId)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
	newConvRes := &messages.NewConversationResponse{}
	if err := json.NewDecoder(res.Body).Decode(newConvRes); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return newConvRes.ConversationId
}

func PatientReplyToConversation(t *testing.T, dataAPI api.DataAPI, conversationId, patientAccountId int64) {
	patientMessageServer := httptest.NewServer(messages.NewPatientMessagesHandler(dataAPI))
	defer patientMessageServer.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.ReplyRequest{
		ConversationId: conversationId,
		Message:        "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := AuthPost(patientMessageServer.URL, "application/json", body, patientAccountId)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
}

func DoctorReplyToConversation(t *testing.T, dataAPI api.DataAPI, conversationId, doctorAccountId int64) {
	doctorMessageServer := httptest.NewServer(messages.NewDoctorMessagesHandler(dataAPI))
	defer doctorMessageServer.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.ReplyRequest{
		ConversationId: conversationId,
		Message:        "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := AuthPost(doctorMessageServer.URL, "application/json", body, doctorAccountId)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
}
