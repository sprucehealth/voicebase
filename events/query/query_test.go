package query

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
)

func TestQueryServerEventByVisitID(t *testing.T) {
	q := ServerEventsByVisitID(1)
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE visit_id=$1`, s)
	test.Equals(t, 1, len(v))
	i, ok := v[0].(*int64)
	test.Assert(t, ok, "Failed conversion to intPtr")
	test.Equals(t, int64(1), *i)
}

func TestQueryServerEventComplexBetween(t *testing.T) {
	ts := time.Now()
	event := "event"
	q := &ServerEventQuery{
		TimestampQuery:  TimestampQuery{Begin: &ts, End: &ts},
		Event:           &event,
		SessionID:       ptr.String("session_id"),
		AccountID:       ptr.Int64(1),
		PatientID:       ptr.Int64(1),
		DoctorID:        ptr.Int64(1),
		VisitID:         ptr.Int64(1),
		CaseID:          ptr.Int64(1),
		TreatmentPlanID: ptr.Int64(1),
		Role:            ptr.String("role"),
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE name=$1 AND session_id=$2 AND account_id=$3 AND patient_id=$4 AND doctor_id=$5 AND visit_id=$6 AND case_id=$7 AND treatment_plan_id=$8 AND role=$9 AND timestamp >= $10 AND timestamp <= $11`, s)
	test.Equals(t, 11, len(v))
	i, ok := v[0].(*string)
	test.Assert(t, ok, "Failed conversion to *string")
	test.Equals(t, event, *i)
}

func TestQueryServerEventComplexAfter(t *testing.T) {
	ts := time.Now()
	event := `name`
	q := &ServerEventQuery{
		TimestampQuery: TimestampQuery{Begin: &ts},
		Event:          &event,
		SessionID:      ptr.String("session_id"),
		AccountID:      ptr.Int64(1),
		PatientID:      ptr.Int64(1),
		DoctorID:       ptr.Int64(1),
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE name=$1 AND session_id=$2 AND account_id=$3 AND patient_id=$4 AND doctor_id=$5 AND timestamp >= $6`, s)
	test.Equals(t, 6, len(v))
	i, ok := v[5].(*time.Time)
	test.Assert(t, ok, "Failed conversion to *time.Time")
	test.Equals(t, ts, *i)
}

func TestQueryServerEventComplexBefore(t *testing.T) {
	ts := time.Now()
	q := &ServerEventQuery{
		TimestampQuery:  TimestampQuery{End: &ts},
		VisitID:         ptr.Int64(1),
		CaseID:          ptr.Int64(1),
		TreatmentPlanID: ptr.Int64(1),
		Role:            ptr.String("role"),
	}
	s, v := q.SQL()
	test.Equals(t, `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event WHERE visit_id=$1 AND case_id=$2 AND treatment_plan_id=$3 AND role=$4 AND timestamp <= $5`, s)
	test.Equals(t, 5, len(v))
	i, ok := v[4].(*time.Time)
	test.Assert(t, ok, "Failed conversion to *time.Time")
	test.Equals(t, ts, *i)
}
