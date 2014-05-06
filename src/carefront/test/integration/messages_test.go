package integration

import (
	"bytes"
	"carefront/api"
	"carefront/common"
	"carefront/messages"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestPersonCreation(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// Make sure a person row is inserted when creating a patient

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientId := pr.Patient.PatientId.Int64()
	if pid, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, patientId); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.PATIENT_ROLE, patientId, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}

	// Make sure a person row is inserted when creating a doctor

	dr, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	doctorId := dr.DoctorId
	if pid, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.DOCTOR_ROLE, doctorId, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}
}

func TestConversationTopics(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	title := "Help"
	ordinal := 123
	id, err := testData.DataApi.AddConversationTopic(title, ordinal, true)
	if err != nil {
		t.Fatal(err)
	}

	topics, err := testData.DataApi.GetConversationTopics()
	if err != nil {
		t.Fatal(err)
	} else if len(topics) < 1 {
		t.Fatalf("Expected at least 1 topic. Got %d", len(topics))
	} else {
		var topic *common.ConversationTopic
		for _, t := range topics {
			if t.Id == id {
				topic = t
				break
			}
		}
		if topic == nil {
			t.Fatal("Created topic not found")
		} else if topic.Title != title {
			t.Fatalf("Expected title '%s'. Got '%s'", title, topic.Title)
		} else if !topic.Active {
			t.Fatal("Expected topic to be active")
		}
	}
}

func TestCreateConversation(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientId, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, pr.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	dr, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	doctorId, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, dr.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}

	photoID := uploadPhoto(t, testData, pr.Patient.AccountId.Int64())
	attachments := []*common.ConversationAttachment{
		&common.ConversationAttachment{
			ItemType: common.AttachmentTypePhoto,
			ItemId:   photoID,
		},
	}

	cid, err := testData.DataApi.CreateConversation(patientId, doctorId, topicId, "Helllloooooo", attachments)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := testData.DataApi.GetConversation(cid); err != nil {
		t.Fatal(err)
	} else if c.Id != cid {
		t.Fatalf("GetConversation did not set Id. Expected %d, got %d.", cid, c.Id)
	} else if c.Title != "Foo" {
		t.Fatalf("GetConversation did not set Title")
	} else if len(c.Participants) != 2 {
		t.Fatalf("Expected 2 participants. Got %d", len(c.Participants))
	} else if c.MessageCount != 1 {
		t.Fatalf("Expected MessageCount of 1. Got %d", c.MessageCount)
	} else if len(c.Messages) != 1 {
		t.Fatalf("Expected 1 message. Got %d", len(c.Messages))
	} else if c.OwnerId != doctorId {
		t.Fatal("Expected doctor to be owner")
	} else if !c.Unread {
		t.Fatal("Conversation should be unread")
	} else {
		for _, p := range c.Participants {
			switch p.RoleType {
			case api.PATIENT_ROLE:
				if p.Id != patientId {
					t.Fatalf("Expected participant patient id %d. Got %d", patientId, p.Id)
				}
			case api.DOCTOR_ROLE:
				if p.Id != doctorId {
					t.Fatalf("Expected participant doctor id %d. Got %d", doctorId, p.Id)
				}
			default:
				t.Fatalf("Unexpected participant role %s", p.RoleType)
			}
		}
		m := c.Messages[0]
		from := c.Participants[m.FromId]
		if from.RoleType != api.PATIENT_ROLE || from.RoleId != pr.Patient.PatientId.Int64() {
			t.Fatalf("Expected message sender to be PATIENT/%d. Got %s/%d", pr.Patient.PatientId.Int64(), from.RoleType, from.RoleId)
		}
		owner := c.Participants[c.OwnerId]
		if owner.RoleType != api.DOCTOR_ROLE || owner.RoleId != dr.DoctorId {
			t.Fatalf("Expected conversation owner to be DOCTOR/%d. Got %s/%d", dr.DoctorId, owner.RoleType, owner.RoleId)
		}
		if len(m.Attachments) != 1 {
			t.Fatalf("Expected 1 attachment. Got %d", len(m.Attachments))
		}
		a := m.Attachments[0]
		if a.ItemType != common.AttachmentTypePhoto || a.ItemId != photoID {
			t.Fatalf("Wrong attachment type or ID")
		}
		photo, err := testData.DataApi.GetPhoto(photoID)
		if err != nil {
			t.Fatal(err)
		}
		if photo.ClaimerType != common.ClaimerTypeConversationMessage {
			t.Fatalf("Expected claimer type of '%s'. Got '%s'", common.ClaimerTypeConversationMessage, photo.ClaimerType)
		}
		if photo.ClaimerId != m.Id {
			t.Fatalf("Expected claimer id to be %d. Got %d", m.Id, photo.ClaimerId)
		}
	}
}

func TestReplyToConversation(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientId, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, pr.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	dr, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	doctorId, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, dr.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}

	cid, err := testData.DataApi.CreateConversation(patientId, doctorId, topicId, "Helllloooooo", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testData.DataApi.ReplyToConversation(cid, doctorId, "Yep yep", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := testData.DataApi.GetConversation(cid); err != nil {
		t.Fatal(err)
	} else if len(c.Participants) != 2 {
		t.Fatalf("Expected 2 participants. Got %d", len(c.Participants))
	} else if c.MessageCount != 2 {
		t.Fatalf("Expected MessageCount of 2. Got %d", c.MessageCount)
	} else if len(c.Messages) != 2 {
		t.Fatalf("Expected 2 message. Got %d", len(c.Messages))
	} else if c.OwnerId != patientId {
		t.Fatal("Expected patient to be owner")
	} else if !c.Unread {
		t.Fatal("Conversation should be unread")
	} else {
		for _, p := range c.Participants {
			switch p.RoleType {
			case api.PATIENT_ROLE:
				if p.Id != patientId {
					t.Fatalf("Expected participant patient id %d. Got %d", patientId, p.Id)
				}
			case api.DOCTOR_ROLE:
				if p.Id != doctorId {
					t.Fatalf("Expected participant doctor id %d. Got %d", doctorId, p.Id)
				}
			default:
				t.Fatalf("Unexpected participant role %s", p.RoleType)
			}
		}
		m := c.Messages[1]
		from := c.Participants[m.FromId]
		if from.RoleType != api.DOCTOR_ROLE || from.RoleId != dr.DoctorId {
			t.Fatalf("Expected message sender to be DOCTOR/%d. Got %s/%d", dr.DoctorId, from.RoleType, from.RoleId)
		}
		owner := c.Participants[c.OwnerId]
		if owner.RoleType != api.PATIENT_ROLE || owner.RoleId != pr.Patient.PatientId.Int64() {
			t.Fatalf("Expected conversation owner to be PATIENT/%d. Got %s/%d", pr.Patient.PatientId.Int64(), owner.RoleType, owner.RoleId)
		}
	}
}

func TestGetConversationsWithParticipants(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientId, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, pr.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	dr, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	doctorId, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, dr.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := testData.DataApi.CreateConversation(patientId, doctorId, topicId, "Helllloooooo", nil)
	if err != nil {
		t.Fatal(err)
	}

	con, par, err := testData.DataApi.GetConversationsWithParticipants([]int64{patientId, doctorId})
	if err != nil {
		t.Fatal(err)
	} else if len(con) != 1 {
		t.Fatalf("Expected 1 conversation. Got %d", len(con))
	} else if len(par) != 2 {
		t.Fatalf("Expected 2 participants. Got %d", len(par))
	} else if con[0].Id != cid {
		t.Fatalf("Expected conversation %d. Got %d", cid, con[0].Id)
	}
}

func TestConversationReadFlag(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientId, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, pr.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	drRes, _, _ := signupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	doctorId, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, drRes.DoctorId)
	if err != nil {
		t.Fatal(err)
	}
	dr, err := testData.DataApi.GetDoctorFromId(drRes.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := testData.DataApi.CreateConversation(patientId, doctorId, topicId, "Helllloooooo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := testData.DataApi.GetConversation(cid); err != nil {
		t.Fatal(err)
	} else if !c.Unread {
		t.Fatalf("Expected conversation to be unread")
	}

	h := messages.NewDoctorReadHandler(testData.DataApi)
	ts := httptest.NewServer(h)
	defer ts.Close()

	res, err := AuthPost(ts.URL, "application/x-www-form-urlencoded", strings.NewReader("conversation_id="+strconv.FormatInt(cid, 10)), dr.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}

	if c, err := testData.DataApi.GetConversation(cid); err != nil {
		t.Fatal(err)
	} else if c.Unread {
		t.Fatalf("Expected conversation to be read")
	}

	_, err = testData.DataApi.ReplyToConversation(cid, doctorId, "Yep yep", nil)
	if err != nil {
		t.Fatal(err)
	}

	if c, err := testData.DataApi.GetConversation(cid); err != nil {
		t.Fatal(err)
	} else if !c.Unread {
		t.Fatalf("Expected conversation to be unread")
	}
}

func TestConversationHandlers(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	pr := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	topicId, err := testData.DataApi.AddConversationTopic("Foo", 100, true)
	if err != nil {
		t.Fatal(err)
	}

	patientConvoServer := httptest.NewServer(messages.NewPatientConversationHandler(testData.DataApi))
	defer patientConvoServer.Close()
	doctorConvoServer := httptest.NewServer(messages.NewDoctorConversationHandler(testData.DataApi))
	defer doctorConvoServer.Close()
	doctorMessageServer := httptest.NewServer(messages.NewDoctorMessagesHandler(testData.DataApi))
	defer doctorMessageServer.Close()

	// New conversation

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.NewConversationRequest{
		TopicId: topicId,
		Message: "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := AuthPost(patientConvoServer.URL, "application/json", body, pr.Patient.AccountId.Int64())
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

	// Make sure conversation was created

	var doctorPersonId int64
	if c, err := testData.DataApi.GetConversation(newConvRes.ConversationId); err != nil {
		t.Fatal(err)
	} else {
		doctorPersonId = c.OwnerId
		for _, p := range c.Participants {
			t.Logf("Participant: %+v", p)
		}
	}
	var dr *common.Doctor
	if p, err := testData.DataApi.GetPeople([]int64{doctorPersonId}); err != nil {
		t.Fatal(err)
	} else {
		dr = p[doctorPersonId].Doctor
	}

	// List conversations

	res, err = AuthGet(fmt.Sprintf("%s?patient_id=%d", doctorConvoServer.URL, pr.Patient.PatientId.Int64()), dr.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
	convList := &messages.ConversationListResponse{}
	if err := json.NewDecoder(res.Body).Decode(convList); err != nil {
		t.Fatal(err)
	}
	if len(convList.Conversations) != 1 {
		t.Fatalf("Expected 1 conversation. Got %d", len(convList.Conversations))
	} else if len(convList.Participants) != 2 {
		t.Fatalf("Expected 2 participants. Got %d", len(convList.Participants))
	}

	// Reply

	body = &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&messages.ReplyRequest{
		ConversationId: newConvRes.ConversationId,
		Message:        "Foo",
	}); err != nil {
		t.Fatal(err)
	}
	res, err = AuthPost(doctorMessageServer.URL, "application/json", body, dr.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}

}
