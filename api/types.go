package api

import "github.com/sprucehealth/backend/common"

type PatientIntake struct {
	PatientID      int64
	PatientVisitID int64
	LVersionID     int64
	SID            string
	SCounter       uint
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

func (p *PatientIntake) SessionID() string {
	return p.SID
}

func (p *PatientIntake) SessionCounter() uint {
	return p.SCounter
}

type DiagnosisIntake struct {
	DoctorID       int64
	PatientVisitID int64
	LVersionID     int64
	SID            string
	SCounter       uint
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

func (d *DiagnosisIntake) SessionID() string {
	return d.SID
}

func (d *DiagnosisIntake) SessionCounter() uint {
	return d.SCounter
}

type DiagnosisDetailsIntake struct {
	DoctorID             int64
	VisitDiagnosisItemID int64
	LVersionID           int64
	SID                  string
	SCounter             uint
	Intake               map[int64][]*common.AnswerIntake
}

func (d *DiagnosisDetailsIntake) TableName() string {
	return "diagnosis_details_intake"
}

func (d *DiagnosisDetailsIntake) Role() *ColumnValue {
	return &ColumnValue{
		Column: "doctor_id",
		Value:  d.DoctorID,
	}
}

func (d *DiagnosisDetailsIntake) Context() *ColumnValue {
	return &ColumnValue{
		Column: "visit_diagnosis_item_id",
		Value:  d.VisitDiagnosisItemID,
	}
}

func (d *DiagnosisDetailsIntake) LayoutVersionID() int64 {
	return d.LVersionID
}

func (d *DiagnosisDetailsIntake) Answers() map[int64][]*common.AnswerIntake {
	return d.Intake
}

func (d *DiagnosisDetailsIntake) SessionID() string {
	return d.SID
}

func (d *DiagnosisDetailsIntake) SessionCounter() uint {
	return d.SCounter
}
