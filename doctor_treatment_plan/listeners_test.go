package doctor_treatment_plan

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type mockDataAPI_tpContent struct {
	api.DataAPI
	tpMap         map[int64]*common.TreatmentPlan
	ftp           *common.FavoriteTreatmentPlan
	regimenPlan   *common.RegimenPlan
	treatments    []*common.Treatment
	noteTPMap     map[int64]string
	patient       *common.Patient
	doctor        *common.Doctor
	scheduledMsgs []*common.TreatmentPlanScheduledMessage

	tpIDMarkedDeviated int64
	updates            []*api.TreatmentPlanScheduledMessageUpdate
}

func (m *mockDataAPI_tpContent) GetAbridgedTreatmentPlan(tpID, doctorID int64) (*common.TreatmentPlan, error) {
	return m.tpMap[tpID], nil
}
func (m *mockDataAPI_tpContent) MarkTPDeviatedFromContentSource(tpID int64) error {
	m.tpIDMarkedDeviated = tpID
	return nil
}
func (m *mockDataAPI_tpContent) FavoriteTreatmentPlan(id int64) (*common.FavoriteTreatmentPlan, error) {
	return m.ftp, nil
}
func (m *mockDataAPI_tpContent) GetTreatmentPlan(id, doctorID int64) (*common.TreatmentPlan, error) {
	return m.tpMap[id], nil
}
func (m *mockDataAPI_tpContent) GetTreatmentsBasedOnTreatmentPlanID(tpID int64) ([]*common.Treatment, error) {
	return m.treatments, nil
}
func (m *mockDataAPI_tpContent) GetRegimenPlanForTreatmentPlan(tpID int64) (*common.RegimenPlan, error) {
	return m.regimenPlan, nil
}
func (m *mockDataAPI_tpContent) GetTreatmentPlanNote(tpID int64) (string, error) {
	return m.noteTPMap[tpID], nil
}
func (m *mockDataAPI_tpContent) Patient(id common.PatientID, basicInfoOnly bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_tpContent) Doctor(id int64, basicInfoOnly bool) (*common.Doctor, error) {
	return m.doctor, nil
}
func (m *mockDataAPI_tpContent) ListTreatmentPlanScheduledMessages(tpID int64) ([]*common.TreatmentPlanScheduledMessage, error) {
	return m.scheduledMsgs, nil
}
func (m *mockDataAPI_tpContent) CreateScheduledMessage(*common.ScheduledMessage) (int64, error) {
	return 0, nil
}
func (m *mockDataAPI_tpContent) UpdateTreatmentPlanScheduledMessage(id int64, update *api.TreatmentPlanScheduledMessageUpdate) error {
	m.updates = append(m.updates, update)
	return nil
}

func TestScheduledMessage_TokenReplacement(t *testing.T) {
	m := &mockDataAPI_tpContent{
		patient: &common.Patient{
			FirstName: "Joe",
			LastName:  "Schmoe",
		},
		doctor: &common.Doctor{
			LongDisplayName:  "Dr. Big Bang",
			ShortDisplayName: "Dr. Bang",
		},
		scheduledMsgs: []*common.TreatmentPlanScheduledMessage{
			{
				Message: "#1 Hello {PTFIRSTNAME}, Sincerely {DRFULLNAME}",
			},
			{
				Message: "#2 Hello {PTFIRSTNAME}, Sincerely {DRFULLNAME}",
			},
		},
	}
	dispatch.Testing = true
	dispatcher := dispatch.New()
	InitListeners(m, dispatcher)
	if err := dispatcher.Publish(&TreatmentPlanSubmittedEvent{
		VisitID:       10,
		TreatmentPlan: &common.TreatmentPlan{},
	}); err != nil {
		t.Fatal(err)
	}

	expectedString1 := "#1 Hello Joe, Sincerely Dr. Big Bang"
	expectedString2 := "#2 Hello Joe, Sincerely Dr. Big Bang"

	// there should be two updates with no tokens in them
	if len(m.updates) != 2 {
		t.Fatalf("Expected 2 updates but got %d", len(m.updates))
	} else if *m.updates[0].Message != expectedString1 {
		t.Fatalf("Expected %s but got %s", expectedString1, *m.updates[0].Message)
	} else if *m.updates[1].Message != expectedString2 {
		t.Fatalf("Expected %s but got %s", expectedString2, *m.updates[1].Message)
	}
}

func TestTPDeviation_RegimenChanged_RevisingPrevTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeTreatmentPlan,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
			2: &common.TreatmentPlan{
				RegimenPlan: &common.RegimenPlan{},
			},
		},
		regimenPlan: &common.RegimenPlan{},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, RegimenSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.regimenPlan = &common.RegimenPlan{
		Sections: []*common.RegimenSection{
			&common.RegimenSection{
				Steps: []*common.DoctorInstructionItem{
					{},
				},
			},
		},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, RegimenSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}
}

func TestTPDeviation_RegimenChanged_FTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeFTP,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
		},
		ftp: &common.FavoriteTreatmentPlan{
			RegimenPlan: &common.RegimenPlan{},
		},
		regimenPlan: &common.RegimenPlan{},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, RegimenSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.regimenPlan = &common.RegimenPlan{
		Sections: []*common.RegimenSection{
			&common.RegimenSection{
				Steps: []*common.DoctorInstructionItem{
					{},
				},
			},
		},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, RegimenSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}

}

func TestTPDeviation_TreatmentsChanged_RevisingPrevTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeTreatmentPlan,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
			2: &common.TreatmentPlan{
				TreatmentList: &common.TreatmentList{},
			},
		},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, TreatmentsSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.treatments = []*common.Treatment{
		{},
		{},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, TreatmentsSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}
}

func TestTPDeviation_TreatmentsChanged_FTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeFTP,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
		},
		ftp: &common.FavoriteTreatmentPlan{
			TreatmentList: &common.TreatmentList{},
		},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, TreatmentsSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.treatments = []*common.Treatment{
		{},
		{},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, TreatmentsSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}
}

func TestTPDeviation_NoteChanged_RevisingPrevTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeTreatmentPlan,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
			2: &common.TreatmentPlan{
				TreatmentList: &common.TreatmentList{},
			},
		},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, NoteSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.noteTPMap = map[int64]string{
		1: "changed",
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, NoteSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}
}

func TestTPDeviation_NoteChanged_FTP(t *testing.T) {
	m := &mockDataAPI_tpContent{

		tpMap: map[int64]*common.TreatmentPlan{
			1: &common.TreatmentPlan{
				ID: encoding.DeprecatedNewObjectID(1),
				ContentSource: &common.TreatmentPlanContentSource{
					Type: common.TPContentSourceTypeFTP,
					ID:   encoding.DeprecatedNewObjectID(2),
				},
			},
		},
		ftp: &common.FavoriteTreatmentPlan{},
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, NoteSection); err != nil {
		t.Fatal(err)
	}

	// ensure that tp was not marked to be deviated
	if m.tpIDMarkedDeviated > 0 {
		t.Fatalf("tp %d was marked as being deviated when it shouldn't have been", m.tpIDMarkedDeviated)
	}

	// now lets update the regimenPlan of the tp being revised to contain sections such that it should deviate
	m.noteTPMap = map[int64]string{
		1: "changed",
	}

	if err := markTPDeviatedIfContentChanged(1, 2, m, NoteSection); err != nil {
		t.Fatal(err)
	}

	// tp should now have deviated
	if m.tpIDMarkedDeviated == 0 {
		t.Fatalf("tp %d did not deviate from source when it should've", m.tpIDMarkedDeviated)
	}
}
