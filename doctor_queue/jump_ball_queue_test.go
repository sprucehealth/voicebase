package doctor_queue

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_visit"
)

type mockDataAPIJBCQ struct {
	api.DataAPI
	pc              *common.PatientCase
	tempClaimedItem *api.DoctorQueueItem
	patient         *common.Patient

	extendClaim     bool
	permanentClaim  bool
	updatesExecuted []*api.DoctorQueueUpdate
}

func (m *mockDataAPIJBCQ) GetPatientCaseFromPatientVisitID(id int64) (*common.PatientCase, error) {
	return m.pc, nil
}
func (m *mockDataAPIJBCQ) ExtendClaimForDoctor(doctorID, patientID, caseID int64, duration time.Duration) error {
	m.extendClaim = true
	return nil
}
func (m *mockDataAPIJBCQ) GetTempClaimedCaseInQueue(caseID int64) (*api.DoctorQueueItem, error) {
	return m.tempClaimedItem, nil
}
func (m *mockDataAPIJBCQ) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPIJBCQ) UpdateDoctorQueue(updates []*api.DoctorQueueUpdate) error {
	m.updatesExecuted = append(m.updatesExecuted, updates...)
	return nil
}
func (m *mockDataAPIJBCQ) TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorID int64, pc *common.PatientCase) error {
	m.permanentClaim = true
	return nil
}

// TestClaimExtension_DiagnosisModified ensures that an existing claim is extended
// upon diagnosis modification
func TestClaimExtension_DiagnosisModified(t *testing.T) {
	m := &mockDataAPIJBCQ{
		pc: &common.PatientCase{},
	}

	dispatcher := dispatch.New()
	initJumpBallCaseQueueListeners(m, nil, dispatcher, metrics.NewRegistry(), 0)

	dispatcher.Publish(&patient_visit.DiagnosisModifiedEvent{})

	if !m.extendClaim {
		t.Fatalf("Expected claim to be extended on diagnosis modification but it wasnt")
	}
}

func TestPermanentClaim_TPStarted(t *testing.T) {
	testPermanentClaimOnEvent(&doctor_treatment_plan.NewTreatmentPlanStartedEvent{
		Case: &common.PatientCase{},
	}, t)
}

func TestPermanentClaim_PatientMessaged(t *testing.T) {
	testPermanentClaimOnEvent(&messages.PostEvent{
		Person: &common.Person{
			RoleType: api.RoleDoctor,
		},
		Case: &common.PatientCase{},
	}, t)
}

func TestPermanentClaim_CaseAssignment(t *testing.T) {
	testPermanentClaimOnEvent(&messages.CaseAssignEvent{
		Person: &common.Person{
			RoleType: api.RoleDoctor,
		},
		Case: &common.PatientCase{},
	}, t)
}

func testPermanentClaimOnEvent(ev interface{}, t *testing.T) {
	m := &mockDataAPIJBCQ{
		pc:              &common.PatientCase{},
		patient:         &common.Patient{},
		tempClaimedItem: &api.DoctorQueueItem{},
	}

	dispatcher := dispatch.New()
	initJumpBallCaseQueueListeners(m, nil, dispatcher, metrics.NewRegistry(), 0)

	dispatcher.Publish(ev)

	if !m.permanentClaim {
		t.Fatal("Expected permanent claim upon TP start but got none")
	} else if len(m.updatesExecuted) != 1 {
		t.Fatalf("Expected 1 updates to be executed on the doctor queue but got %d", len(m.updatesExecuted))
	} else if m.updatesExecuted[0].Action != api.DQActionInsert {
		t.Fatalf("Expected action %#v but got %#v", api.DQActionInsert, m.updatesExecuted[0].Action)
	} else if m.updatesExecuted[0].QueueItem.EventType != api.DQEventTypePatientVisit {
		t.Fatalf("Expected event type %s but got %s", m.updatesExecuted[0].QueueItem.EventType, api.DQEventTypePatientVisit)
	}
}
