package query

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/test"
)

func TestQueryServerEventByVisitID(t *testing.T) {
	var id int64
	id = 1
	q := ServerEventsByVisitID(&id)
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE visit_id=$1`, s)
	test.Equals(t, 1, len(v))
	i, ok := v[0].(*int64)
	test.Assert(t, ok, "Failed conversion to intPtr")
	test.Equals(t, int64(1), *i)
}

func TestQueryServerEventComplexBetween(t *testing.T) {
	var id int64
	id = 1
	ts := time.Now()
	event := `name`
	session_id := `session_id`
	account_id := id
	patient_id := id
	doctor_id := id
	visit_id := id
	case_id := id
	treatment_plan_id := id
	role := `role`
	q := &ServerEventQuery{
		TimestampQuery:  TimestampQuery{Begin: &ts, End: &ts},
		Event:           &event,
		SessionID:       &session_id,
		AccountID:       &account_id,
		PatientID:       &patient_id,
		DoctorID:        &doctor_id,
		VisitID:         &visit_id,
		CaseID:          &case_id,
		TreatmentPlanID: &treatment_plan_id,
		Role:            &role,
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE name=$1 AND session_id=$2 AND account_id=$3 AND patient_id=$4 AND doctor_id=$5 AND visit_id=$6 AND case_id=$7 AND treatment_plan_id=$8 AND role=$9 AND timestamp >= $10 AND timestamp <= $11`, s)
	test.Equals(t, 11, len(v))
	i, ok := v[0].(*string)
	test.Assert(t, ok, "Failed conversion to *string")
	test.Equals(t, event, *i)
}

func TestQueryServerEventComplexAfter(t *testing.T) {
	var id int64
	id = 1
	ts := time.Now()
	event := `name`
	session_id := `session_id`
	account_id := id
	patient_id := id
	doctor_id := id
	q := &ServerEventQuery{
		TimestampQuery: TimestampQuery{Begin: &ts},
		Event:          &event,
		SessionID:      &session_id,
		AccountID:      &account_id,
		PatientID:      &patient_id,
		DoctorID:       &doctor_id,
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE name=$1 AND session_id=$2 AND account_id=$3 AND patient_id=$4 AND doctor_id=$5 AND timestamp >= $6`, s)
	test.Equals(t, 6, len(v))
	i, ok := v[5].(*time.Time)
	test.Assert(t, ok, "Failed conversion to *time.Time")
	test.Equals(t, ts, *i)
}

func TestQueryServerEventComplexBefore(t *testing.T) {
	var id int64
	id = 1
	ts := time.Now()
	visit_id := id
	case_id := id
	treatment_plan_id := id
	role := `role`
	q := &ServerEventQuery{
		TimestampQuery:  TimestampQuery{End: &ts},
		VisitID:         &visit_id,
		CaseID:          &case_id,
		TreatmentPlanID: &treatment_plan_id,
		Role:            &role,
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE visit_id=$1 AND case_id=$2 AND treatment_plan_id=$3 AND role=$4 AND timestamp <= $5`, s)
	test.Equals(t, 5, len(v))
	i, ok := v[4].(*time.Time)
	test.Assert(t, ok, "Failed conversion to *time.Time")
	test.Equals(t, ts, *i)
}
