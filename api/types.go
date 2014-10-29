package api

import "github.com/sprucehealth/backend/common"

type PatientIntake struct {
	PatientID      int64
	PatientVisitID int64
	LVersionID     int64
	Intake         map[int64][]*common.AnswerIntake
}

func (p *PatientIntake) TableName() string {
	return "info_intake"
}

func (p *PatientIntake) Role() *ColumnValue {
	return &ColumnValue{
		Column: "patient_id",
		Value:  p.PatientID,
	}
}

func (p *PatientIntake) Context() *ColumnValue {
	return &ColumnValue{
		Column: "patient_visit_id",
		Value:  p.PatientVisitID,
	}
}

func (p *PatientIntake) LayoutVersionID() int64 {
	return p.LVersionID
}

func (p *PatientIntake) Answers() map[int64][]*common.AnswerIntake {
	return p.Intake
}

type DiagnosisIntake struct {
	DoctorID       int64
	PatientVisitID int64
	LVersionID     int64
	Intake         map[int64][]*common.AnswerIntake
}

func (d *DiagnosisIntake) TableName() string {
	return "diagnosis_intake"
}

func (d *DiagnosisIntake) Role() *ColumnValue {
	return &ColumnValue{
		Column: "doctor_id",
		Value:  d.DoctorID,
	}
}

func (d *DiagnosisIntake) Context() *ColumnValue {
	return &ColumnValue{
		Column: "patient_visit_id",
		Value:  d.PatientVisitID,
	}
}

func (d *DiagnosisIntake) LayoutVersionID() int64 {
	return d.LVersionID
}

func (d *DiagnosisIntake) Answers() map[int64][]*common.AnswerIntake {
	return d.Intake
}
