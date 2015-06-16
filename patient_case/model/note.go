package model

import "time"

type PatientCaseNote struct {
	ID             int64
	CaseID         int64
	AuthorDoctorID int64
	Created        time.Time
	Modified       time.Time
	NoteText       string
}

type PatientCaseNoteUpdate struct {
	ID       int64
	NoteText string
}
