package api

import (
	"database/sql"
)

func (d *DataService) PatientFeedbackRecorded(patientID int64, feedbackFor string) (bool, error) {
	var x bool
	err := d.db.QueryRow(`SELECT 1 FROM patient_feedback WHERE patient_id = ? AND feedback_for = ?`, patientID, feedbackFor).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return true, err
}

func (d *DataService) RecordPatientFeedback(patientID int64, feedbackFor string, rating int, comment *string) error {
	_, err := d.db.Exec(
		`INSERT INTO patient_feedback (patient_id, feedback_for, rating, comment) VALUES (?, ?, ?, ?)`,
		patientID, feedbackFor, rating, comment)
	return err
}
