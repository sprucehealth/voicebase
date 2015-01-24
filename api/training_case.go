package api

import (
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

func (d *DataService) CreateTrainingCaseSet(status string) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO training_case_set (status)
		VALUES (?)`, status)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (d *DataService) ClaimTrainingSet(doctorID int64, pathwayTag string) error {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// ensure that there is a training set available
	var trainingSetID int64
	if err := tx.QueryRow(
		`SELECT id FROM training_case_set WHERE status = ?`, common.TCSStatusPending,
	).Scan(&trainingSetID); err == sql.ErrNoRows {
		tx.Rollback()
		return ErrNotFound("training_case_set")
	} else if err != nil {
		tx.Rollback()
		return err
	}

	// lets go ahead and permanently assign the doctor to the training patients
	_, err = tx.Exec(`
		INSERT INTO patient_case_care_provider_assignment
			(patient_case_id, role_type_id, provider_id, status)
		SELECT patient_visit.patient_case_id,?,?,?
		FROM training_case
		INNER JOIN  patient_visit ON training_case.patient_visit_id = patient_visit.id
		WHERE training_case_set_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, STATUS_ACTIVE, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO patient_care_provider_assignment
			(patient_id, role_type_id, provider_id, status, clinical_pathway_id)
		SELECT patient_visit.patient_id, ?,?,?,?
		FROM training_case
		INNER JOIN patient_visit ON training_case.patient_visit_id = patient_visit.id
		WHERE training_case_set_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, STATUS_ACTIVE, pathwayID, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// collect a list of visitIDs belonging to the set
	rows, err := tx.Query(`
		SELECT patient_visit_id, patient_id, patient_case_id
		FROM training_case
		INNER JOIN patient_visit on patient_visit.id = patient_visit_id
		WHERE training_case_set_id = ?`, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	var visitIDs []int64
	var patientIDs []int64
	var caseIDs []int64
	for rows.Next() {
		var visitID, patientID, caseID int64
		if err := rows.Scan(&visitID, &patientID, &caseID); err != nil {
			tx.Rollback()
			return err
		}
		visitIDs = append(visitIDs, visitID)
		patientIDs = append(patientIDs, patientID)
		caseIDs = append(caseIDs, caseID)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}

	// add each visit into the doctor's queue
	for i, visitID := range visitIDs {
		patient, err := d.Patient(patientIDs[i], true)
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := insertItemIntoDoctorQueue(tx, &DoctorQueueItem{
			EventType:        DQEventTypePatientVisit,
			PatientID:        patient.PatientID.Int64(),
			Status:           DQItemStatusPending,
			DoctorID:         doctorID,
			ItemID:           visitID,
			Description:      fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName),
			ShortDescription: "New visit",
			ActionURL:        app_url.ViewPatientVisitInfoAction(patient.PatientID.Int64(), visitID, caseIDs[i]),
		}); err != nil {
			tx.Rollback()
			return err
		}

	}

	// consider all vists as part of this training case as now routed
	_, err = tx.Exec(`
		UPDATE patient_visit SET status = ?
		WHERE id IN (SELECT patient_visit_id from training_case WHERE training_case_set_id = ?) `, common.PVStatusRouted, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// consider all cases as claimed
	_, err = tx.Exec(`
		UPDATE patient_case set status = ?
		WHERE id in (
			SELECT patient_visit.patient_case_id
			FROM training_case
			INNER JOIN patient_visit ON patient_visit.id = training_case.patient_visit_id
			WHERE training_case.training_case_set_id = ?)`, common.PCStatusClaimed.String(), trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// now delete the training set that was claimed
	_, err = tx.Exec(`DELETE FROM training_case_set WHERE id = ?`, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) QueueTrainingCase(tCase *common.TrainingCase) error {
	_, err := d.db.Exec(`
		INSERT INTO training_case
		(training_case_set_id, patient_visit_id, template_name) 
		VALUES (?,?,?) `, tCase.TrainingCaseSetID, tCase.PatientVisitID, tCase.TemplateName)
	return err
}

func (d *DataService) UpdateTrainingCaseSetStatus(id int64, status string) error {
	_, err := d.db.Exec(`UPDATE training_case_set SET status = ? WHERE id = ?`, status, id)
	return err
}

func (d *DataService) TrainingCaseSetCount(status string) (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT count(*) from training_case_set where status = ?`, status).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
