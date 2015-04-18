package doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/notify"
)

type mockDataAPI_listener struct {
	api.DataAPI
	patient     *common.Patient
	doctor      *common.Doctor
	assignments []*common.CareProviderAssignment

	updatesRequested []*api.DoctorQueueUpdate
}

func (m *mockDataAPI_listener) UpdateDoctorQueue(updates []*api.DoctorQueueUpdate) error {
	m.updatesRequested = append(m.updatesRequested, updates...)
	return nil
}
func (m *mockDataAPI_listener) UpdatePatientCaseFeedItem(item *common.PatientCaseFeedItem) error {
	return nil
}
func (m *mockDataAPI_listener) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_listener) GetActiveMembersOfCareTeamForCase(caseID int64, basicInfoOnly bool) ([]*common.CareProviderAssignment, error) {
	return m.assignments, nil
}
func (m *mockDataAPI_listener) Doctor(id int64, basicInfoOnly bool) (*common.Doctor, error) {
	return m.doctor, nil
}

type mockAuthAPI_listener struct {
	api.AuthAPI
	phoneNumbers []*common.PhoneNumber
}

func (m *mockAuthAPI_listener) GetPhoneNumbersForAccount(accountID int64) ([]*common.PhoneNumber, error) {
	return m.phoneNumbers, nil
}

type nullSMSAPI struct{}

func (nullSMSAPI) Send(fromNumber, toNumber, text string) error {
	return nil
}

// TestCaseAssignment_CCToDoctor tests to ensure that when a CC assigns a case to a doctor
// the right updates to the doctor queue are made. They are:
// 1. Delete any pending case assignment for the doctor/CC
// 2. Insert an item into the history for the sender of the case assignment
// 3. Insert an item into the inbox of the recipient of the case assignment.
func TestCaseAssignment_CCToDoctor(t *testing.T) {
	testCaseAssignment(t, api.RoleMA)
}

func TestCaseAssignment_DoctorToCC(t *testing.T) {
	testCaseAssignment(t, api.RoleDoctor)
}

func testCaseAssignment(t *testing.T, role string) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{},
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())

	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	ma := &common.Doctor{
		DoctorID:         encoding.NewObjectID(4),
		ShortDisplayName: "Care Coordinator",
	}

	doctor := &common.Doctor{
		DoctorID:         encoding.NewObjectID(2),
		ShortDisplayName: "Doctor",
	}

	var providerID int64
	switch role {
	case api.RoleMA:
		providerID = doctor.DoctorID.Int64()
	case api.RoleDoctor:
		providerID = ma.DoctorID.Int64()
	}

	dispatcher.Publish(&messages.CaseAssignEvent{
		Message: &common.CaseMessage{
			CaseID: 10,
		},
		Person: &common.Person{
			RoleType: role,
			RoleID:   1,
		},
		Case: &common.PatientCase{
			PatientID: encoding.NewObjectID(5),
			ID:        encoding.NewObjectID(10),
			Claimed:   true,
		},
		MA:     ma,
		Doctor: doctor,
	})

	// at this point there should be 3 items in the doctor queue
	if len(m.updatesRequested) != 3 {
		t.Fatalf("Expected 3 items for update but got %d", len(m.updatesRequested))
	}

	itemToDelete := m.updatesRequested[0]
	if itemToDelete.Action != api.DQActionDelete {
		t.Fatalf("Expected %s but got %s", itemToDelete.Action, api.DQActionDelete)
	} else if itemToDelete.QueueItem.EventType != api.DQEventTypeCaseAssignment {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, itemToDelete.QueueItem.EventType)
	} else if itemToDelete.QueueItem.DoctorID != 1 {
		t.Fatalf("Expected DoctorID 1 but got %d", itemToDelete.QueueItem.DoctorID)
	} else if itemToDelete.QueueItem.Status != api.DQItemStatusPending {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusPending, itemToDelete.QueueItem.Status)
	}

	historyItem := m.updatesRequested[1]
	if historyItem.Action != api.DQActionInsert {
		t.Fatalf("Expected %s but got %s", itemToDelete.Action, api.DQActionInsert)
	} else if historyItem.QueueItem.EventType != api.DQEventTypeCaseAssignment {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, historyItem.QueueItem.EventType)
	} else if historyItem.QueueItem.DoctorID != 1 {
		t.Fatalf("Expected DoctorID 1 but got %d", historyItem.QueueItem.DoctorID)
	} else if historyItem.QueueItem.Status != api.DQItemStatusReplied {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusReplied, historyItem.QueueItem.Status)
	}

	inboxItem := m.updatesRequested[2]
	if inboxItem.Action != api.DQActionInsert {
		t.Fatalf("Expected %s but got %s", inboxItem.Action, api.DQActionInsert)
	} else if inboxItem.QueueItem.EventType != api.DQEventTypeCaseAssignment {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, inboxItem.QueueItem.EventType)
	} else if inboxItem.QueueItem.DoctorID != providerID {
		t.Fatalf("Expected DoctorID 2  but got %d", inboxItem.QueueItem.DoctorID)
	} else if inboxItem.QueueItem.Status != api.DQItemStatusPending {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusPending, inboxItem.QueueItem.Status)
	}
}

// This test is to ensure that multiple case assignments from CC to doctor
// only results in a single item in the doctor's inbox (i.e, we dedupe on them)
func TestCaseAssignment_Multiple(t *testing.T) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{},
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())
	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	// assign the case 2 times from the cc to the doctor
	for i := 0; i < 2; i++ {
		dispatcher.Publish(&messages.CaseAssignEvent{
			Message: &common.CaseMessage{
				CaseID: 10,
			},
			Person: &common.Person{
				RoleType: api.RoleMA,
				RoleID:   1,
			},
			Case: &common.PatientCase{
				PatientID: encoding.NewObjectID(5),
				ID:        encoding.NewObjectID(10),
				Claimed:   true,
			},
			MA: &common.Doctor{
				DoctorID:         encoding.NewObjectID(4),
				ShortDisplayName: "Care Coordinator",
			},
			Doctor: &common.Doctor{
				DoctorID:         encoding.NewObjectID(2),
				ShortDisplayName: "Doctor",
			},
		})
	}

	// assigning the case 2 times from the CC -> doctor should result in
	// 2 deletes, 2 inserts into the history of the CC, and 2 dedupes
	// for inserts into the doctor's inbox.
	if len(m.updatesRequested) != 6 {
		t.Fatalf("Expected 3 update requests but got %d", len(m.updatesRequested))
	}

	for i := 0; i < 6; i++ {
		switch i {
		case 2, 5:
			if !m.updatesRequested[i].Dedupe {
				t.Fatalf("Expected insert at %d to dedupe but it didn't", i)
			}
		default:
			if m.updatesRequested[i].Dedupe {
				t.Fatalf("Expected update request at %d to NOT dedupe but it did", i)
			}
		}
	}
}

// TestCaseAssignment_Doctor_PersistsInInbox ensures that a case assignment from an
// MA remains in the doctor's inbox even when the doctor views the message thread.
func TestCaseAssignment_Doctor_PersistsInInbox(t *testing.T) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{},
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())

	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	dispatcher.Publish(&messages.CaseAssignEvent{
		Message: &common.CaseMessage{
			CaseID: 10,
		},
		Person: &common.Person{
			RoleType: api.RoleMA,
			RoleID:   1,
		},
		Case: &common.PatientCase{
			PatientID: encoding.NewObjectID(5),
			ID:        encoding.NewObjectID(10),
			Claimed:   true,
		},
		MA: &common.Doctor{
			DoctorID:         encoding.NewObjectID(4),
			ShortDisplayName: "Care Coordinator",
		},
		Doctor: &common.Doctor{
			DoctorID:         encoding.NewObjectID(2),
			ShortDisplayName: "Doctor",
		},
	})

	// at this point there should be 3 items in the doctor queue
	if len(m.updatesRequested) != 3 {
		t.Fatalf("Expected 3 items for update but got %d", len(m.updatesRequested))
	}

	dispatcher.Publish(&app_event.AppEvent{
		Action:     app_event.ViewedAction,
		Resource:   "all_case_messages",
		ResourceID: 10,
		Role:       api.RoleDoctor,
		AccountID:  12,
	})

	// at this point we should still only have 3 items in the doctor queue updates
	if len(m.updatesRequested) != 3 {
		t.Fatalf("Expected 3 items for update but got %d", len(m.updatesRequested))
	}
}

// TestCaseAssignment_Doctor_DismissByMA ensures that a case assignment to a doctor
// can be dismissed by the MA simply by viewing the message thread
func TestCaseAssignment_Doctor_DismissByMA(t *testing.T) {

}

// TestMessage_PatientToCareTeam_NoDoctor ensures that a patient message
// reaches the MA's inbox as expected
func TestMessage_PatientToCareTeam_NoDoctor(t *testing.T) {
	testMessage_PatientToCareTeam(t, []*common.CareProviderAssignment{
		{
			Status:       api.StatusActive,
			ProviderRole: api.RoleMA,
			ProviderID:   10,
		},
	})
}

// TestMessage_PatientToCareTeam_DoctorAssigned ensures that a patient messsage
// reached the MA's inbox when a doctor is assigned to the case.
func TestMessage_PatientToCareTeam_DoctorAssigned(t *testing.T) {
	testMessage_PatientToCareTeam(t, []*common.CareProviderAssignment{
		{
			Status:       api.StatusActive,
			ProviderRole: api.RoleMA,
			ProviderID:   10,
		},
		{
			Status:       api.StatusActive,
			ProviderRole: api.RoleDoctor,
			ProviderID:   11,
		},
	})
}

func testMessage_PatientToCareTeam(t *testing.T, assignments []*common.CareProviderAssignment) {
	m := &mockDataAPI_listener{
		patient:     &common.Patient{},
		doctor:      &common.Doctor{},
		assignments: assignments,
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())

	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	dispatcher.Publish(&messages.PostEvent{
		Message: &common.CaseMessage{
			CaseID: 10,
		},
		Person: &common.Person{
			RoleType: api.RolePatient,
			RoleID:   1,
		},
		Case: &common.PatientCase{
			PatientID: encoding.NewObjectID(5),
			ID:        encoding.NewObjectID(10),
			Claimed:   true,
		},
	})

	// there should be a single insert into the CC's inbox
	if len(m.updatesRequested) != 1 {
		t.Fatalf("Expected 1 items for update but got %d", len(m.updatesRequested))
	} else if m.updatesRequested[0].Action != api.DQActionInsert {
		t.Fatalf("Expected %s but got %s", api.DQActionInsert, m.updatesRequested[0].Action)
	} else if m.updatesRequested[0].QueueItem.EventType != api.DQEventTypeCaseMessage {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, m.updatesRequested[0].QueueItem.EventType)
	} else if m.updatesRequested[0].QueueItem.Status != api.DQItemStatusPending {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusPending, m.updatesRequested[0].QueueItem.Status)
	}
}

func TestMessage_DoctorToPatient(t *testing.T) {
	testMessage_ProviderToPatient(t, api.RoleDoctor)
}

func TestMessage_MAToPatient(t *testing.T) {
	testMessage_ProviderToPatient(t, api.RoleMA)
}

func TestMessage_DoctorToPatient_Multiple(t *testing.T) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{},
		doctor:  &common.Doctor{},
		assignments: []*common.CareProviderAssignment{
			{
				Status:       api.StatusActive,
				ProviderRole: api.RoleDoctor,
				ProviderID:   10,
			},
			{
				Status:       api.StatusActive,
				ProviderRole: api.RoleDoctor,
				ProviderID:   11,
			},
		},
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())

	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	for i := 0; i < 2; i++ {
		dispatcher.Publish(&messages.PostEvent{
			Message: &common.CaseMessage{
				CaseID: 10,
			},
			Person: &common.Person{
				RoleType: api.RoleDoctor,
				RoleID:   11,
			},
			Case: &common.PatientCase{
				PatientID: encoding.NewObjectID(5),
				ID:        encoding.NewObjectID(10),
				Claimed:   true,
			},
		})
	}

	if len(m.updatesRequested) != 4 {
		t.Fatalf("Expected 4 update requests to doctor queue but only got %d", len(m.updatesRequested))
	}
}

func testMessage_ProviderToPatient(t *testing.T, role string) {
	m := &mockDataAPI_listener{
		patient: &common.Patient{},
		doctor:  &common.Doctor{},
		assignments: []*common.CareProviderAssignment{
			{
				Status:       api.StatusActive,
				ProviderRole: role,
				ProviderID:   10,
			},
			{
				Status:       api.StatusActive,
				ProviderRole: api.RoleDoctor,
				ProviderID:   11,
			},
		},
	}

	a := &mockAuthAPI_listener{
		phoneNumbers: []*common.PhoneNumber{
			{
				Phone: "734846552",
			},
		},
	}

	notifyManager := notify.NewManager(m, a, &sns.MockSNS{}, &nullSMSAPI{}, nil, "", nil, metrics.NewRegistry())

	dispatcher := dispatch.New()
	InitListeners(m, nil, dispatcher, notifyManager, metrics.NewRegistry(), 0, "")

	dispatcher.Publish(&messages.PostEvent{
		Message: &common.CaseMessage{
			CaseID: 10,
		},
		Person: &common.Person{
			RoleType: api.RoleDoctor,
			RoleID:   11,
		},
		Case: &common.PatientCase{
			PatientID: encoding.NewObjectID(5),
			ID:        encoding.NewObjectID(10),
			Claimed:   true,
		},
	})

	// there should be a delete and insert requests
	itemToDelete := m.updatesRequested[0]
	if itemToDelete.Action != api.DQActionDelete {
		t.Fatalf("Expected %s but got %s", itemToDelete.Action, api.DQActionDelete)
	} else if itemToDelete.QueueItem.EventType != api.DQEventTypeCaseAssignment {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, itemToDelete.QueueItem.EventType)
	} else if itemToDelete.QueueItem.DoctorID != 11 {
		t.Fatalf("Expected DoctorID 1 but got %d", itemToDelete.QueueItem.DoctorID)
	} else if itemToDelete.QueueItem.Status != api.DQItemStatusPending {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusPending, itemToDelete.QueueItem.Status)
	}

	historyItem := m.updatesRequested[1]
	if historyItem.Action != api.DQActionInsert {
		t.Fatalf("Expected %s but got %s", itemToDelete.Action, api.DQActionInsert)
	} else if historyItem.QueueItem.EventType != api.DQEventTypeCaseMessage {
		t.Fatalf("Expected %s but got %s", api.DQEventTypeCaseAssignment, historyItem.QueueItem.EventType)
	} else if historyItem.QueueItem.DoctorID != 11 {
		t.Fatalf("Expected DoctorID 1 but got %d", historyItem.QueueItem.DoctorID)
	} else if historyItem.QueueItem.Status != api.DQItemStatusReplied {
		t.Fatalf("Expected %s but got %s", api.DQItemStatusReplied, historyItem.QueueItem.Status)
	}
}
