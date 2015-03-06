package main

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
)

type visitDoctorPair struct {
	visitID  int64
	doctorID int64
}

type diagnosisIntakeItem struct {
	questionTag    string
	answerSelected *string
	text           string
	created        time.Time
}

type diagnosisDetailsIntake struct {
	visitDiagnosisItemID int64
	visitID              int64
	doctorID             int64
	layoutVersionID      int64
	answeredDate         time.Time
	questionID           int64
	potentialAnswerID    *int64
}

type diagnosisDetailsIFace interface {
	ActiveDiagnosisDetailsIntake(codeID string, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error)
}

type dataManagerIFace interface {
	distinctVisitDoctorPairsSince(time time.Time) ([]*visitDoctorPair, error)
	diagnosisSetExistsForPair(p *visitDoctorPair) (bool, error)
	diagnosisItems(p *visitDoctorPair) ([]*diagnosisIntakeItem, error)

	beginTransaction() (*sql.Tx, error)
	commitTransaction(tx *sql.Tx) error
	rollbackTransaction(tx *sql.Tx) error
	createVisitDiagnosisSet(tx *sql.Tx, set *common.VisitDiagnosisSet) error
	createDiagnosisDetailsIntake(tx *sql.Tx, intake *diagnosisDetailsIntake) error
}

type dataManager struct {
	db *sql.DB
}

func (d *dataManager) distinctVisitDoctorPairsSince(time time.Time) ([]*visitDoctorPair, error) {
	rows, err := d.db.Query(`
		SELECT DISTINCT patient_visit_id, doctor_id 
		FROM diagnosis_intake
		WHERE answered_date >= ?`, time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visitDoctorPairs []*visitDoctorPair
	for rows.Next() {
		var visitID, doctorID int64
		if err := rows.Scan(&visitID, &doctorID); err != nil {
			return nil, err
		}

		visitDoctorPairs = append(visitDoctorPairs, &visitDoctorPair{
			visitID:  visitID,
			doctorID: doctorID,
		})
	}

	return visitDoctorPairs, rows.Err()
}

func (d *dataManager) diagnosisSetExistsForPair(p *visitDoctorPair) (bool, error) {
	// ensure that this visitDoctorPair doesn't already have a diagnosis set created for it
	var id int64
	if err := d.db.QueryRow(`
			SELECT id FROM visit_diagnosis_set
			WHERE patient_visit_id = ?
			AND doctor_id = ?
			AND active = 1`, p.visitID, p.doctorID).
		Scan(&id); err != sql.ErrNoRows && err != nil {
		return false, err
	}

	return id > 0, nil
}

func (d *dataManager) diagnosisItems(p *visitDoctorPair) ([]*diagnosisIntakeItem, error) {
	rows, err := d.db.Query(`
		SELECT question.question_tag, potential_answer.answer_text, diagnosis_intake.answer_text, diagnosis_intake.answered_date 
		FROM diagnosis_intake 
		LEFT OUTER JOIN question ON question.id = question_id 
		LEFT OUTER JOIN potential_answer ON potential_answer.id = potential_answer_id
		WHERE patient_visit_id = ? AND doctor_id = ?`, p.visitID, p.doctorID)
	if err != nil {
		return nil, err
	}

	var intakeItems []*diagnosisIntakeItem
	for rows.Next() {
		var item diagnosisIntakeItem
		if err := rows.Scan(
			&item.questionTag,
			&item.answerSelected,
			&item.text,
			&item.created); err != nil {
			return nil, err
		}

		intakeItems = append(intakeItems, &item)
	}

	return intakeItems, rows.Err()
}

func (d *dataManager) beginTransaction() (*sql.Tx, error) {
	return d.db.Begin()
}

func (d *dataManager) commitTransaction(tx *sql.Tx) error {
	return tx.Commit()
}

func (d *dataManager) rollbackTransaction(tx *sql.Tx) error {
	return tx.Rollback()
}

func (d *dataManager) createVisitDiagnosisSet(tx *sql.Tx, set *common.VisitDiagnosisSet) error {

	// inactivate any previous diagnosis sets pertaining to this visit
	_, err := tx.Exec(`
		UPDATE visit_diagnosis_set
		SET active = 0
		WHERE patient_visit_id = ?
		AND doctor_id = ?
		AND active = 1
		`, set.VisitID, set.DoctorID)
	if err != nil {
		return err
	}

	// create the new set
	res, err := tx.Exec(`
		INSERT INTO visit_diagnosis_set (patient_visit_id, doctor_id, notes, active, unsuitable, unsuitable_reason, created) 
		VALUES (?,?,?,?,?,?,?)`, set.VisitID, set.DoctorID, set.Notes, true, set.Unsuitable, set.UnsuitableReason, set.Created)
	if err != nil {
		return err
	}

	set.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}

	if len(set.Items) > 0 {
		// insert the item 1 at a time versus a batch insert because
		// we need the IDs of the items being inserted
		insertItemStmt, err := tx.Prepare(`
			INSERT INTO visit_diagnosis_item
			(visit_diagnosis_set_id, diagnosis_code_id, layout_version_id) 
			VALUES (?,?,?)`)
		if err != nil {
			return err
		}
		defer insertItemStmt.Close()

		for _, item := range set.Items {
			res, err := insertItemStmt.Exec(set.ID, item.CodeID, item.LayoutVersionID)
			if err != nil {
				return err
			}

			item.ID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *dataManager) createDiagnosisDetailsIntake(tx *sql.Tx, intake *diagnosisDetailsIntake) error {
	_, err := tx.Exec(
		`INSERT INTO diagnosis_details_intake 
			(doctor_id, visit_diagnosis_item_id, layout_version_id, answered_date, question_id, potential_answer_id)
		 VALUES (?,?,?,?,?,?)`,
		intake.doctorID,
		intake.visitDiagnosisItemID,
		intake.layoutVersionID,
		intake.answeredDate,
		intake.questionID,
		intake.potentialAnswerID)
	return err
}
