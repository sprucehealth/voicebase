package schedmsg

import (
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
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
	PersonID      int64
	CareTeams     map[int64]*common.PatientCareTeam
	People        map[int64]*common.Person
	Doc           *common.Doctor
	CaseMessageID int64
	Error         error
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
	return d.PersonID, d.Error
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

func (d mockedDataAPI_WorkerTest) CreateCaseMessage(msg *common.CaseMessage) (int64, error) {
	return d.CaseMessageID, d.Error
}

func TestCaseNotReassignedOnTPScheduledMessageNoCC(t *testing.T) {
	data := &mockedDataAPI_WorkerTest{&api.DataService{}, nil, nil, nil, 0, nil, nil, nil, 0, nil}
	publisher := &TestPublisher{}
	worker := NewWorker(data, nil, publisher, metrics.NewRegistry(), 1)
	data.TP = &common.TreatmentPlan{Status: api.StatusActive, DoctorID: encoding.NewObjectID(1), PatientCaseID: encoding.NewObjectID(1), PatientID: 1}
	data.TPSM = &common.TreatmentPlanScheduledMessage{}
	data.PCase = &common.PatientCase{ID: encoding.NewObjectID(1)}
	data.CareTeams = map[int64]*common.PatientCareTeam{1: {Assignments: make([]*common.CareProviderAssignment, 0)}}
	data.PersonID = 1
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
	data := &mockedDataAPI_WorkerTest{&api.DataService{}, nil, nil, nil, 0, nil, nil, nil, 0, nil}
	publisher := &TestPublisher{}
	worker := NewWorker(data, nil, publisher, metrics.NewRegistry(), 1)
	data.TP = &common.TreatmentPlan{Status: api.StatusActive, DoctorID: encoding.NewObjectID(1), PatientCaseID: encoding.NewObjectID(1), PatientID: 1}
	data.TPSM = &common.TreatmentPlanScheduledMessage{}
	data.PCase = &common.PatientCase{ID: encoding.NewObjectID(1)}
	data.CareTeams = map[int64]*common.PatientCareTeam{1: {Assignments: []*common.CareProviderAssignment{&common.CareProviderAssignment{ProviderRole: api.RoleMA, ProviderID: 1}}}}
	data.PersonID = 1
	data.People = map[int64]*common.Person{1: &common.Person{Doctor: &common.Doctor{}}}
	data.CaseMessageID = 1
	data.Doc = &common.Doctor{DoctorID: encoding.NewObjectID(99)}
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
			test.Equals(t, int64(99), re.MA.DoctorID.Int64())
		}
	}
	test.Assert(t, foundPost, "Expected a single PostEvent")
	test.Assert(t, foundReassign, "Expected a single CaseAssignEvent")
}
