package query

import (
	"fmt"
	"strings"
	"time"
)

type TimestampQuery struct {
	Begin *time.Time
	End   *time.Time
}

func (eq *TimestampQuery) TimestampConditionString(n int) []string {
	q := make([]string, 0, 2)
	if eq.Begin != nil {
		q = append(q, fmt.Sprintf(`timestamp >= $%d`, n))
		n++
	}
	if eq.End != nil {
		q = append(q, fmt.Sprintf(`timestamp <= $%d`, n))
		n++
	}
	return q
}

func (eq *TimestampQuery) TimestampConditionValues() []interface{} {
	v := make([]interface{}, 0, 2)
	if eq.Begin != nil {
		v = append(v, eq.Begin)
	}
	if eq.End != nil {
		v = append(v, eq.End)
	}
	return v
}

type ServerEventQuery struct {
	TimestampQuery
	Event           *string
	SessionID       *string
	AccountID       *int64
	PatientID       *int64
	DoctorID        *int64
	VisitID         *int64
	CaseID          *int64
	TreatmentPlanID *int64
	Role            *string
}

func (seq *ServerEventQuery) SQL() (string, []interface{}) {
	q := `SELECT name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json FROM server_event`
	conditionFields := make([]string, 0, 11)
	conditionValues := make([]interface{}, 0, 11)
	n := 1
	if seq.Event != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`name=$%d`, n))
		conditionValues = append(conditionValues, seq.Event)
		n++
	}
	if seq.SessionID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`session_id=$%d`, n))
		conditionValues = append(conditionValues, seq.SessionID)
		n++
	}
	if seq.AccountID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`account_id=$%d`, n))
		conditionValues = append(conditionValues, seq.AccountID)
		n++
	}
	if seq.PatientID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`patient_id=$%d`, n))
		conditionValues = append(conditionValues, seq.PatientID)
		n++
	}
	if seq.DoctorID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`doctor_id=$%d`, n))
		conditionValues = append(conditionValues, seq.DoctorID)
		n++
	}
	if seq.VisitID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`visit_id=$%d`, n))
		conditionValues = append(conditionValues, seq.VisitID)
		n++
	}
	if seq.CaseID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`case_id=$%d`, n))
		conditionValues = append(conditionValues, seq.CaseID)
		n++
	}
	if seq.TreatmentPlanID != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`treatment_plan_id=$%d`, n))
		conditionValues = append(conditionValues, seq.TreatmentPlanID)
		n++
	}
	if seq.Role != nil {
		conditionFields = append(conditionFields, fmt.Sprintf(`role=$%d`, n))
		conditionValues = append(conditionValues, seq.Role)
		n++
	}
	conditionFields = append(conditionFields, seq.TimestampConditionString(n)...)
	conditionValues = append(conditionValues, seq.TimestampConditionValues()...)
	if len(conditionFields) > 0 {
		q += fmt.Sprintf(` WHERE %s`, strings.Join(conditionFields, ` AND `))
	}
	return q, conditionValues
}

func ServerEventsByVisitID(visitID int64) *ServerEventQuery {
	return &ServerEventQuery{
		TimestampQuery: TimestampQuery{},
		VisitID:        &visitID,
	}
}
