package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

func (d *dataService) PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error) {
	var x bool
	err := d.db.QueryRow(`SELECT 1 FROM patient_feedback WHERE patient_id = ? AND feedback_for = ?`, patientID, feedbackFor).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return true, err
}

func (d *dataService) RecordPatientFeedback(patientID common.PatientID, feedbackFor string, rating int, comment *string) error {
	_, err := d.db.Exec(
		`INSERT INTO patient_feedback (patient_id, feedback_for, rating, comment) VALUES (?, ?, ?, ?)`,
		patientID, feedbackFor, rating, comment)
	return err
}

func (d *dataService) PatientFeedback(feedbackFor string) ([]*common.PatientFeedback, error) {
	rows, err := d.db.Query(`SELECT patient_id, rating, comment, created FROM patient_feedback WHERE feedback_for = ?`, feedbackFor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var feedback []*common.PatientFeedback
	for rows.Next() {
		var pf common.PatientFeedback
		var comment sql.NullString
		if err := rows.Scan(&pf.PatientID, &pf.Rating, &comment, &pf.Created); err != nil {
			return nil, err
		}
		pf.Comment = comment.String
		feedback = append(feedback, &pf)
	}
	return feedback, rows.Err()
}
