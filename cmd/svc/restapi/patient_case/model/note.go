package model

import "time"

// PatientCaseNote represents the data associated with a patient_case_note record
type PatientCaseNote struct {
	ID             int64
	CaseID         int64
	AuthorDoctorID int64
	Created        time.Time
	Modified       time.Time
	NoteText       string
}

// PatientCaseNoteUpdate represents the mutable data in a patient_case_note record
type PatientCaseNoteUpdate struct {
	ID       int64
	NoteText string
}
