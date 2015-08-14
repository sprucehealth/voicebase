package schedmsg

import (
	"testing"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
)

type TestPublisher struct {
	Events map[interface{}]int
}

func (p *TestPublisher) Publish(el interface{}) error {
	if p.Events == nil {
		p.Events = make(map[interface{}]int)
	}
	p.Events[el]++
	return nil
}

func (p *TestPublisher) PublishAsync(el interface{}) {
	p.Publish(el)
}

type mockedDataAPI_WorkerTest struct {
	api.DataAPI
	PCase         *common.PatientCase
	TPSM          *common.TreatmentPlanScheduledMessage
	TP            *common.TreatmentPlan
	PersonID      map[int64]int64
	CareTeams     map[int64]*common.PatientCareTeam
	People        map[int64]*common.Person
	Doc           *common.Doctor
	CaseMessageID int64
	Error         error

	msgCreated *common.CaseMessage
}

func (d mockedDataAPI_WorkerTest) GetAbridgedTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error) {
	return d.TP, d.Error
}

func (d mockedDataAPI_WorkerTest) TreatmentPlanScheduledMessage(id int64) (*common.TreatmentPlanScheduledMessage, error) {
	return d.TPSM, d.Error
}

func (d mockedDataAPI_WorkerTest) GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error) {
	return d.PCase, d.Error
}

func (d mockedDataAPI_WorkerTest) GetPersonIDByRole(roleType string, roleID int64) (int64, error) {
	return d.PersonID[roleID], d.Error
}

func (d mockedDataAPI_WorkerTest) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return d.CareTeams, d.Error
}

func (d mockedDataAPI_WorkerTest) Doctor(id int64, basicInfoOnly bool) (doctor *common.Doctor, err error) {
	return d.Doc, d.Error
}

func (d mockedDataAPI_WorkerTest) GetPeople(ids []int64) (map[int64]*common.Person, error) {
	return d.People, d.Error
}

func (d *mockedDataAPI_WorkerTest) CreateCaseMessage(msg *common.CaseMessage) (int64, error) {
	d.msgCreated = msg
	return d.CaseMessageID, d.Error
}

// TestScheduledMessage_FromActiveDoctor ensures that a treatment plan
// scheduled message goes from the active doctor on the case versus the doctor on the TP.
func TestScheduledMessage_FromActiveDoctor(t *testing.T) {
	activeDoctorOnCareTeamID := int64(510)
	doctorOnTPID := int64(500)

	data := &mockedDataAPI_WorkerTest{
		PCase: &common.PatientCase{
			ID: encoding.DeprecatedNewObjectID(1),
		},
		TPSM: &common.TreatmentPlanScheduledMessage{},
		TP: &common.TreatmentPlan{
			Status:        api.StatusActive,
			DoctorID:      encoding.DeprecatedNewObjectID(doctorOnTPID),
			PatientCaseID: encoding.DeprecatedNewObjectID(1),
			PatientID:     common.NewPatientID(1),
		},
		PersonID: map[int64]int64{
			activeDoctorOnCareTeamID: activeDoctorOnCareTeamID,
			doctorOnTPID:             doctorOnTPID,
		},
		CareTeams: map[int64]*common.PatientCareTeam{
			1: {
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderRole: api.RoleDoctor,
						ProviderID:   activeDoctorOnCareTeamID,
					},
					{
						ProviderRole: api.RoleCC,
						ProviderID:   521,
					},
				},
			},
		},
		People: map[int64]*common.Person{
			510: &common.Person{
				Doctor: &common.Doctor{},
			},
		},
		Doc:           &common.Doctor{},
		CaseMessageID: 0,
		Error:         nil,
	}
	publisher := &TestPublisher{}
	worker := NewWorker(data, nil, publisher, metrics.NewRegistry(), 1)
	worker.processMessage(&common.ScheduledMessage{
		Message: &TreatmentPlanMessage{},
	})
	test.Equals(t, 2, len(publisher.Events))
	// ensure that the case message has the personID that maps to the active doctor on the care team
	// and not the personID of the doctor on the TP
	test.Equals(t, activeDoctorOnCareTeamID, data.msgCreated.PersonID)
}

func TestCaseNotReassignedOnTPScheduledMessageNoCC(t *testing.T) {
	data := &mockedDataAPI_WorkerTest{}
	publisher := &TestPublisher{}
	worker := NewWorker(data, nil, publisher, metrics.NewRegistry(), 1)
	data.TP = &common.TreatmentPlan{Status: api.StatusActive, DoctorID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientID: common.NewPatientID(1)}
	data.TPSM = &common.TreatmentPlanScheduledMessage{}
	data.PCase = &common.PatientCase{ID: encoding.DeprecatedNewObjectID(1)}
	data.CareTeams = map[int64]*common.PatientCareTeam{1: {Assignments: make([]*common.CareProviderAssignment, 0)}}
	data.PersonID = map[int64]int64{
		1: 1,
	}
	data.People = map[int64]*common.Person{1: &common.Person{Doctor: &common.Doctor{}}}
	data.CaseMessageID = 1
	msg := &common.ScheduledMessage{
		Message: &TreatmentPlanMessage{},
	}
	worker.processMessage(msg)
	test.Equals(t, 1, len(publisher.Events))
	for k, v := range publisher.Events {
		_, ok := k.(*messages.PostEvent)
		test.Assert(t, ok, "Expected only event present to be of type *messages.PostEvent")
		test.Equals(t, 1, v)
	}
}

func TestCaseReassignedOnTPScheduledMessageCC(t *testing.T) {
	data := &mockedDataAPI_WorkerTest{}
	publisher := &TestPublisher{}
	worker := NewWorker(data, nil, publisher, metrics.NewRegistry(), 1)
	data.TP = &common.TreatmentPlan{Status: api.StatusActive, DoctorID: encoding.DeprecatedNewObjectID(1), PatientCaseID: encoding.DeprecatedNewObjectID(1), PatientID: common.NewPatientID(1)}
	data.TPSM = &common.TreatmentPlanScheduledMessage{}
	data.PCase = &common.PatientCase{ID: encoding.DeprecatedNewObjectID(1)}
	data.CareTeams = map[int64]*common.PatientCareTeam{1: {Assignments: []*common.CareProviderAssignment{&common.CareProviderAssignment{ProviderRole: api.RoleCC, ProviderID: 1}, &common.CareProviderAssignment{ProviderRole: api.RoleDoctor, ProviderID: 1}}}}
	data.PersonID = map[int64]int64{
		1: 1,
	}
	data.People = map[int64]*common.Person{1: &common.Person{Doctor: &common.Doctor{}}}
	data.CaseMessageID = 1
	data.Doc = &common.Doctor{ID: encoding.DeprecatedNewObjectID(99)}
	msg := &common.ScheduledMessage{
		Message: &TreatmentPlanMessage{},
	}
	worker.processMessage(msg)
	test.Equals(t, 2, len(publisher.Events))
	var foundPost, foundReassign bool
	for k, v := range publisher.Events {
		_, ok := k.(*messages.PostEvent)
		if ok {
			foundPost = true
			test.Equals(t, 1, v)
		}
		re, ok := k.(*messages.CaseAssignEvent)
		if ok {
			foundReassign = true
			test.Equals(t, 1, v)
			test.Equals(t, int64(99), re.MA.ID.Int64())
		}
	}
	test.Assert(t, foundPost, "Expected a single PostEvent")
	test.Assert(t, foundReassign, "Expected a single CaseAssignEvent")
}
