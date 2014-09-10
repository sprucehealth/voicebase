package api

import (
	"database/sql"

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

func (d *DataService) ClaimTrainingSet(doctorID, healthConditionID int64) error {
	// ensure that there is a training set available
	var trainingSetID int64
	if err := d.db.QueryRow(`SELECT id FROM training_case_set where status = ?`, common.TCSStatusPending).Scan(&trainingSetID); err == sql.ErrNoRows {
		return NoRowsError
	} else if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// lets go ahead and permanently assign the doctor to the training patients
	_, err = tx.Exec(`INSERT INTO patient_case_care_provider_assignment 
		(patient_case_id, role_type_id, provider_id, status) 
		SELECT patient_visit.patient_case_id,?,?,? 
		FROM training_case
		INNER JOIN  patient_visit ON training_case.patient_visit_id = patient_visit.id
		WHERE training_case_set_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, STATUS_ACTIVE, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`INSERT INTO patient_care_provider_assignment 
		(patient_id, role_type_id, provider_id, status, health_condition_id) 
		SELECT patient_visit.patient_id, ?,?,?,?
		FROM training_case
		INNER JOIN patient_visit ON training_case.patient_visit_id = patient_visit.id
		WHERE training_case_set_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, STATUS_ACTIVE, healthConditionID, trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// lets go ahead and claim the training set into the doctor's queue
	_, err = tx.Exec(`
		INSERT INTO doctor_queue (doctor_id, status, event_type, item_id)
		SELECT ?, ?, ?, training_case.patient_visit_id 
		FROM training_case where training_case_set_id = ?`,
		doctorID,
		DQItemStatusPending,
		DQEventTypePatientVisit,
		trainingSetID)
	if err != nil {
		tx.Rollback()
		return err
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
			WHERE training_case.training_case_set_id = ?)`, common.PCStatusClaimed, trainingSetID)
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
