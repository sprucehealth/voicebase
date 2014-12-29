package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	pharmacyService "github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/sku"
)

func (d *DataService) GetPatientIDFromPatientVisitID(patientVisitID int64) (int64, error) {
	var patientID int64
	err := d.db.QueryRow("select patient_id from patient_visit where id = ?", patientVisitID).Scan(&patientID)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientID, err
}

// Adding this only to link the patient and the doctor app so as to show the doctor
// the patient visit review of the latest submitted patient visit
func (d *DataService) GetLatestSubmittedPatientVisit() (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status, sku_id, followup 
		FROM patient_visit 
		WHERE status IN ('SUBMITTED', 'REVIEWING') 
		ORDER BY submitted_date DESC 
		LIMIT 1`)
	if err != nil {
		return nil, err
	}

	patientVisits, err := d.getPatientVisitFromRows(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(patientVisits); {
	case l == 0:
		return nil, NoRowsError
	case l == 1:
		return patientVisits[0], nil
	}

	return nil, fmt.Errorf("expected 1 patient visit but got %d", len(patientVisits))
}

func (d *DataService) PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error) {

	// get the creation time of the initial visit
	var creationDate time.Time
	err := d.db.QueryRow(`SELECT creation_date FROM patient_visit WHERE patient_case_id = ? ORDER BY id LIMIT 1`, caseID).Scan(&creationDate)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	// look for a pending followup visit created after the initial visit
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, health_condition_id,
		layout_version_id, creation_date, submitted_date, closed_date, 
		status, sku_id, followup
		FROM patient_visit
	 	WHERE patient_case_id = ? AND status = ? AND creation_date > ? 
	 	LIMIT 1
		`, caseID, common.PVStatusPending, creationDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) GetPatientVisitForSKU(patientID int64, skuType sku.SKU) (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, health_condition_id,
		layout_version_id, creation_date, submitted_date, closed_date,
		status, sku_id, followup
		FROM patient_visit
	 	WHERE patient_id = ? AND sku_id = ? 
	 	LIMIT 1
		`, patientID, d.skuMapping[skuType.String()])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) GetLastCreatedPatientVisit(patientID int64) (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, health_condition_id,
		layout_version_id, creation_date, submitted_date, closed_date, 
		status, sku_id, followup
		FROM patient_visit
	 	WHERE patient_id = ? AND creation_date IS NOT NULL 
	 	ORDER BY creation_date DESC LIMIT 1`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status, sku_id, followup
		FROM patient_visit 
		WHERE id = ?`, patientVisitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) GetPatientVisitFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT pv.id, pv.patient_id, pv.patient_case_id, pv.health_condition_id, 
		pv.layout_version_id, pv.creation_date, pv.submitted_date, pv.closed_date, 
		pv.status, pv.sku_id, pv.followup
		FROM patient_visit pv
		INNER JOIN treatment_plan_patient_visit_mapping m ON m.patient_visit_id = pv.id
		WHERE treatment_plan_id = ?`, treatmentPlanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) getSinglePatientVisit(rows *sql.Rows) (*common.PatientVisit, error) {
	patientVisits, err := d.getPatientVisitFromRows(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(patientVisits); {
	case l == 0:
		return nil, NoRowsError
	case l == 1:
		return patientVisits[0], nil
	}

	return nil, fmt.Errorf("expected 1 patient visit but got %d", len(patientVisits))
}

func (d *DataService) getPatientVisitFromRows(rows *sql.Rows) ([]*common.PatientVisit, error) {
	var patientVisits []*common.PatientVisit

	for rows.Next() {
		var patientVisit common.PatientVisit
		var submittedDate, closedDate mysql.NullTime
		var skuID int64
		err := rows.Scan(
			&patientVisit.PatientVisitID,
			&patientVisit.PatientID,
			&patientVisit.PatientCaseID,
			&patientVisit.HealthConditionID,
			&patientVisit.LayoutVersionID,
			&patientVisit.CreationDate,
			&submittedDate,
			&closedDate,
			&patientVisit.Status,
			&skuID,
			&patientVisit.IsFollowup)
		if err != nil {
			return nil, err
		}
		patientVisit.SubmittedDate = submittedDate.Time
		patientVisit.ClosedDate = closedDate.Time
		patientVisit.SKU, err = sku.GetSKU(d.skuIDToTypeMapping[skuID])
		if err != nil {
			return nil, err
		}

		patientVisits = append(patientVisits, &patientVisit)
	}

	return patientVisits, rows.Err()
}

func (d *DataService) GetPatientCaseIDFromPatientVisitID(patientVisitID int64) (int64, error) {
	var patientCaseID int64
	if err := d.db.QueryRow(`select patient_case_id from patient_visit where id=?`, patientVisitID).Scan(&patientCaseID); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}
	return patientCaseID, nil
}

func (d *DataService) CreatePatientVisit(visit *common.PatientVisit) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	caseID := visit.PatientCaseID.Int64()
	if caseID == 0 {
		// implicitly create a new case when creating a new visit for now
		// for now treating the creation of every new case as an unclaimed case because we don't have a notion of a
		// new case for which the patient returns (and thus can be potentially claimed)
		patientCase := &common.PatientCase{
			PatientID:         encoding.NewObjectID(visit.PatientID.Int64()),
			HealthConditionID: encoding.NewObjectID(visit.HealthConditionID.Int64()),
			Status:            common.PCStatusUnclaimed,
		}

		if err := d.createPatientCase(tx, patientCase); err != nil {
			tx.Rollback()
			return 0, err
		}

		caseID = patientCase.ID.Int64()
	}

	res, err := tx.Exec(`
		INSERT INTO patient_visit (patient_id, health_condition_id, layout_version_id, patient_case_id, status, sku_id, followup) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		visit.PatientID.Int64(), visit.HealthConditionID.Int64(), visit.LayoutVersionID.Int64(), caseID,
		visit.Status, d.skuMapping[visit.SKU.String()], visit.IsFollowup)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	visit.CreationDate = time.Now()
	visit.PatientVisitID = encoding.NewObjectID(lastID)
	visit.PatientCaseID = encoding.NewObjectID(caseID)
	return lastID, err
}

func (d *DataService) GetMessageForPatientVisit(patientVisitID int64) (string, error) {
	var message string
	if err := d.db.QueryRow(`SELECT message FROM patient_visit_message WHERE patient_visit_id = ?`, patientVisitID).Scan(&message); err == sql.ErrNoRows {
		return "", NoRowsError
	} else if err != nil {
		return "", err
	}
	return message, nil
}

func (d *DataService) SetMessageForPatientVisit(patientVisitID int64, message string) error {
	_, err := d.db.Exec(`REPLACE INTO patient_visit_message (patient_visit_id, message) VALUES (?,?) `, patientVisitID, message)
	return err
}

func (d *DataService) GetAbridgedTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`select id, doctor_id, patient_id, patient_case_id, status, creation_date from treatment_plan where id = ?`, treatmentPlanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	drTreatmentPlans, err := d.getAbridgedTreatmentPlanFromRows(rows, doctorID)
	if err != nil {
		return nil, err
	}

	switch l := len(drTreatmentPlans); {
	case l == 0:
		return nil, NoRowsError
	case l == 1:
		return drTreatmentPlans[0], nil
	}

	return nil, fmt.Errorf("Expected 1 drTreatmentPlan instead got %d", len(drTreatmentPlans))
}

// IsRevisedTreatmentPlan returns true if the treatmentPlan is a revision of another treatment
// plan in the case
func (d *DataService) IsRevisedTreatmentPlan(treatmentPlanID int64) (bool, error) {
	// get case id
	var caseID int64
	var creationDate time.Time
	if err := d.db.QueryRow(`SELECT patient_case_id, creation_date FROM treatment_plan WHERE id = ?`, treatmentPlanID).Scan(&caseID, &creationDate); err == sql.ErrNoRows {
		return false, NoRowsError
	} else if err != nil {
		return false, err
	}

	// check if there exist inactive treatment plans in the case that were created prior to this one
	var count int64
	if err := d.db.QueryRow(`SELECT count(*) FROM treatment_plan where creation_date < ? AND patient_case_id = ?`, creationDate, caseID).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d *DataService) UpdateTreatmentPlanStatus(treatmentPlanID int64, status common.TreatmentPlanStatus) error {
	_, err := d.db.Exec(`UPDATE treatment_plan 
		SET status = ? WHERE id = ?`, status.String(), treatmentPlanID)
	return err
}

func (d *DataService) GetTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error) {
	treatmentPlan, err := d.GetAbridgedTreatmentPlan(treatmentPlanID, doctorID)
	if err != nil {
		return nil, err
	}

	// get treatments
	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = d.GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID)
	if err != nil {
		return nil, err
	}

	// get regimen
	treatmentPlan.RegimenPlan, err = d.GetRegimenPlanForTreatmentPlan(treatmentPlanID)
	if err != nil {
		return nil, err
	}

	return treatmentPlan, nil
}

func (d *DataService) getAbridgedTreatmentPlanFromRows(rows *sql.Rows, doctorID int64) ([]*common.TreatmentPlan, error) {
	drTreatmentPlans := make([]*common.TreatmentPlan, 0)
	for rows.Next() {
		var drTreatmentPlan common.TreatmentPlan
		if err := rows.Scan(&drTreatmentPlan.ID, &drTreatmentPlan.DoctorID, &drTreatmentPlan.PatientID, &drTreatmentPlan.PatientCaseID, &drTreatmentPlan.Status, &drTreatmentPlan.CreationDate); err != nil {
			return nil, err
		}

		// parent information has to exist for every treatment plan; so if it doesn't that means something is logically inconsistent
		drTreatmentPlan.Parent = &common.TreatmentPlanParent{}
		err := d.db.QueryRow(`select parent_id, parent_type from treatment_plan_parent where treatment_plan_id = ?`, drTreatmentPlan.ID.Int64()).Scan(&drTreatmentPlan.Parent.ParentID, &drTreatmentPlan.Parent.ParentType)
		if err == sql.ErrNoRows {
			return nil, NoRowsError
		} else if err != nil {
			return nil, err
		}

		// get the creation date of the parent since this information is useful for the client
		var creationDate time.Time
		switch drTreatmentPlan.Parent.ParentType {
		case common.TPParentTypePatientVisit:
			if err := d.db.QueryRow(`select creation_date from patient_visit where id = ?`, drTreatmentPlan.Parent.ParentID.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, NoRowsError
			} else if err != nil {
				return nil, err
			}
		case common.TPParentTypeTreatmentPlan:
			if err := d.db.QueryRow(`select creation_date from treatment_plan where id = ?`, drTreatmentPlan.Parent.ParentID.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, NoRowsError
			} else if err != nil {
				return nil, err
			}
		}
		drTreatmentPlan.Parent.CreationDate = creationDate

		// only populate content source information if we are retrieving this information for the same doctor that created the treatment plan
		drTreatmentPlan.ContentSource = &common.TreatmentPlanContentSource{}
		err = d.db.QueryRow(`
			SELECT content_source_id, content_source_type, has_deviated
			FROM treatment_plan_content_source
			WHERE treatment_plan_id = ? and doctor_id = ?`,
			drTreatmentPlan.ID.Int64(), doctorID,
		).Scan(
			&drTreatmentPlan.ContentSource.ID,
			&drTreatmentPlan.ContentSource.Type,
			&drTreatmentPlan.ContentSource.HasDeviated)
		if err == sql.ErrNoRows {
			// treat content source as empty if non specified
			drTreatmentPlan.ContentSource = nil
		} else if err != nil {
			return nil, err
		}

		drTreatmentPlans = append(drTreatmentPlans, &drTreatmentPlan)
	}
	return drTreatmentPlans, rows.Err()
}

func (d *DataService) GetAbridgedTreatmentPlanList(doctorID, patientID int64, statuses []common.TreatmentPlanStatus) ([]*common.TreatmentPlan, error) {
	where := "patient_id = ?"
	vals := []interface{}{patientID}

	if l := len(statuses); l > 0 {
		where += " AND status in (" + nReplacements(l) + ")"
		for _, sItem := range statuses {
			vals = append(vals, sItem.String())
		}
	}

	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_id, patient_case_id, status, creation_date 
		FROM treatment_plan 
		WHERE `+where, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getAbridgedTreatmentPlanFromRows(rows, doctorID)
}

func (d *DataService) GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, patientID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_id, patient_case_id, status, creation_date 
		FROM treatment_plan 
		WHERE doctor_id = ? AND patient_id = ? AND status = ?`,
		doctorID, patientID, common.TPStatusDraft.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getAbridgedTreatmentPlanFromRows(rows, doctorID)
}

func (d *DataService) DeleteTreatmentPlan(treatmentPlanID int64) error {
	_, err := d.db.Exec(`delete from treatment_plan where id = ?`, treatmentPlanID)
	return err
}

func (d *DataService) GetPatientIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error) {
	var patientID int64
	err := d.db.QueryRow(`select patient_id from treatment_plan where id = ?`, treatmentPlanID).Scan(&patientID)

	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}

	return patientID, err
}

func (d *DataService) GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error) {
	var patientVisitID int64
	err := d.db.QueryRow(`select patient_visit_id from treatment_plan_patient_visit_mapping where treatment_plan_id = ?`, treatmentPlanID).Scan(&patientVisitID)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}

	return patientVisitID, nil
}

func (d *DataService) StartNewTreatmentPlan(patientID, patientVisitID, doctorID int64, parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	// Delete any existing draft treatment plan matching the doctor, patient, source, and parent
	_, err = tx.Exec(`
		DELETE FROM treatment_plan
		WHERE id = (SELECT treatment_plan_id FROM treatment_plan_parent WHERE parent_id = ? AND parent_type = ?)
			AND status = ? AND doctor_id = ?`,
		parent.ParentID.Int64(), parent.ParentType, common.TPStatusDraft.String(), doctorID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// get the case the treatment plan belongs to from the patient visit
	patientCaseID, err := d.GetPatientCaseIDFromPatientVisitID(patientVisitID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastID, err := tx.Exec(`
		INSERT INTO treatment_plan
		(patient_id, doctor_id, patient_case_id, status)
		VALUES (?,?,?,?)`, patientID, doctorID, patientCaseID, common.TPStatusDraft.String())
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	treatmentPlanID, err := lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the patient visit that is the reason for which the treatment plan is being created
	_, err = tx.Exec(`
		INSERT INTO treatment_plan_patient_visit_mapping
		(treatment_plan_id, patient_visit_id)
		VALUES (?,?)`, treatmentPlanID, patientVisitID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the parent information for treatment plan
	_, err = tx.Exec(`INSERT INTO treatment_plan_parent
		(treatment_plan_id,parent_id, parent_type) VALUES (?,?,?)`, treatmentPlanID, parent.ParentID.Int64(), parent.ParentType)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the original content source for the treatment plan
	if contentSource != nil {
		_, err := tx.Exec(`
			INSERT INTO treatment_plan_content_source
				(treatment_plan_id, doctor_id, content_source_id, content_source_type)
			VALUES (?,?,?,?)`,
			treatmentPlanID, doctorID, contentSource.ID.Int64(), contentSource.Type)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	err = tx.Commit()
	return treatmentPlanID, err
}

func (d *DataService) UpdatePatientVisit(id int64, update *PatientVisitUpdate) error {
	return updatePatientVisit(d.db, id, update)
}

func (d *DataService) UpdatePatientVisits(ids []int64, update *PatientVisitUpdate) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, visitID := range ids {
		if err := updatePatientVisit(tx, visitID, update); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func updatePatientVisit(d db, id int64, update *PatientVisitUpdate) error {
	cols := []string{}
	vals := []interface{}{}

	if update.Status != nil {
		cols = append(cols, "status = ?")
		vals = append(vals, *update.Status)
	}

	if update.LayoutVersionID != nil {
		cols = append(cols, "layout_version_id = ?")
		vals = append(vals, *update.LayoutVersionID)
	}

	if update.ClosedDate != nil {
		cols = append(cols, "closed_date = ?")
		vals = append(vals, *update.ClosedDate)
	}

	if len(cols) == 0 {
		return nil
	}

	vals = append(vals, id)

	_, err := d.Exec(`update patient_visit set `+strings.Join(cols, ",")+` where id = ?`, vals...)
	return err
}

func (d *DataService) ClosePatientVisit(patientVisitID int64, event string) error {
	_, err := d.db.Exec(`update patient_visit set status=?, closed_date=now() where id = ?`, event, patientVisitID)
	return err
}

func (d *DataService) ActivateTreatmentPlan(treatmentPlanID, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	treatmentPlan, err := d.GetAbridgedTreatmentPlan(treatmentPlanID, doctorID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// inactivate any previous treatment plan for this case
	_, err = tx.Exec(`update treatment_plan set status = ? where patient_case_id = ?`, common.TPStatusInactive.String(), treatmentPlan.PatientCaseID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	// mark the treatment plan as ACTIVE
	_, err = tx.Exec(`update treatment_plan set status = ? where id = ?`, common.TPStatusActive.String(), treatmentPlan.ID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) SubmitPatientVisitWithID(patientVisitID int64) error {
	_, err := d.db.Exec("update patient_visit set status='SUBMITTED', submitted_date=now() where id = ? and STATUS in ('OPEN', 'PHOTOS_REJECTED')", patientVisitID)
	return err
}

func (d *DataService) CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	err = func() error {
		// delete any previous steps and sections given that we have new ones coming in
		_, err := tx.Exec(`DELETE FROM regimen WHERE treatment_plan_id = ?`, regimenPlan.TreatmentPlanID.Int64())
		if err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM regimen_section WHERE treatment_plan_id = ?`, regimenPlan.TreatmentPlanID.Int64())
		if err != nil {
			return err
		}

		secStmt, err := tx.Prepare(`INSERT INTO regimen_section (treatment_plan_id, title) VALUES (?,?)`)
		if err != nil {
			return err
		}
		defer secStmt.Close()

		stepStmt, err := tx.Prepare(`INSERT INTO regimen (treatment_plan_id, regimen_section_id, dr_regimen_step_id, text, status) VALUES (?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer stepStmt.Close()

		// create new regimen steps within each section
		for _, section := range regimenPlan.Sections {
			res, err := secStmt.Exec(regimenPlan.TreatmentPlanID.Int64(), section.Name)
			if err != nil {
				return err
			}
			secID, err := res.LastInsertId()
			if err != nil {
				return err
			}
			for _, step := range section.Steps {
				_, err = stepStmt.Exec(regimenPlan.TreatmentPlanID.Int64(), secID, step.ParentID.Int64Ptr(), step.Text, STATUS_ACTIVE)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}()
	if err != nil {
		if e := tx.Rollback(); e != nil {
			golog.Errorf("Rollback failed: %s", e.Error())
		}
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetRegimenPlanForTreatmentPlan(treatmentPlanID int64) (*common.RegimenPlan, error) {
	rows, err := d.db.Query(`
		SELECT regimen.id, rs.title, dr_regimen_step_id, text
		FROM regimen
		INNER JOIN regimen_section rs ON rs.id = regimen_section_id
		WHERE regimen.treatment_plan_id = ?
			AND status = ?
		ORDER BY regimen.id`, treatmentPlanID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regimenPlan, err := getRegimenPlanFromRows(rows)
	if err != nil {
		return nil, err
	}
	regimenPlan.TreatmentPlanID = encoding.NewObjectID(treatmentPlanID)

	return regimenPlan, nil
}

func (d *DataService) AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorID, treatmentPlanID, patientID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("update treatment set status=? where treatment_plan_id = ?", common.TStatusInactive.String(), treatmentPlanID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, treatment := range treatments {
		treatment.TreatmentPlanID = encoding.NewObjectID(treatmentPlanID)
		err = d.addTreatment(treatmentForPatientType, treatment, nil, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

		if treatment.DoctorTreatmentTemplateID.Int64() != 0 {
			_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatment.ID.Int64(), treatment.DoctorTreatmentTemplateID.Int64())
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID int64) ([]*common.Treatment, error) {

	// get treatment plan information
	treatments := make([]*common.Treatment, 0)
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date, 
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			treatment_plan.patient_id, treatment_plan.doctor_id, is_controlled_substance from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment_plan_id=? and treatment.status=? and localized_text.language_id = ?`, treatmentPlanID, common.TStatusCreated.String(), EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatmentIds := make([]int64, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatment.TreatmentPlanID = encoding.NewObjectID(treatmentPlanID)
		treatments = append(treatments, treatment)
		treatmentIds = append(treatmentIds, treatment.ID.Int64())
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return treatments, nil
	}

	favoriteRows, err := d.db.Query(fmt.Sprintf(`select dr_treatment_template_id , treatment_dr_template_selection.treatment_id from treatment_dr_template_selection 
													inner join dr_treatment_template on dr_treatment_template.id = dr_treatment_template_id
														where treatment_dr_template_selection.treatment_id in (%s) and dr_treatment_template.status = ?`, enumerateItemsIntoString(treatmentIds)), common.TStatusCreated.String())
	treatmentIdToFavoriteIdMapping := make(map[int64]int64)
	if err != nil {
		return nil, err
	}
	defer favoriteRows.Close()

	for favoriteRows.Next() {
		var drFavoriteTreatmentId, treatmentID int64
		err = favoriteRows.Scan(&drFavoriteTreatmentId, &treatmentID)
		if err != nil {
			return nil, err
		}
		treatmentIdToFavoriteIdMapping[treatmentID] = drFavoriteTreatmentId
	}

	// assign the treatments the doctor favorite id if one exists
	for _, treatment := range treatments {
		if treatmentIdToFavoriteIdMapping[treatment.ID.Int64()] != 0 {
			treatment.DoctorTreatmentTemplateID = encoding.NewObjectID(treatmentIdToFavoriteIdMapping[treatment.ID.Int64()])
		}
	}

	return treatments, nil
}

func (d *DataService) GetTreatmentsForPatient(patientID int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			treatment_plan.patient_id, treatment_plan.doctor_id, is_controlled_substance from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment_plan.patient_id = ? and treatment.status=? and localized_text.language_id = ?`, patientID, common.TStatusCreated.String(), EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// get treatment plan information
	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}
		treatments = append(treatments, treatment)
	}

	return treatments, rows.Err()
}

func (d *DataService) GetTreatmentPlanForPatient(patientID, treatmentPlanID int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status
		FROM treatment_plan
		WHERE id = ?`, treatmentPlanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatmentPlans, err := getTreatmentPlansFromRows(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(treatmentPlans); {
	case l == 0:
		return nil, NoRowsError
	case l > 1:
		return nil, fmt.Errorf("Expected 1 treatment plan instead got %d", len(treatmentPlans))
	}

	tp := treatmentPlans[0]
	if tp.PatientID != patientID {
		return nil, NoRowsError
	}
	return tp, nil
}

func (d *DataService) GetActiveTreatmentPlansForPatient(patientID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status
		FROM treatment_plan
		WHERE patient_id = ?
			AND status = ?`, patientID, common.TPStatusActive.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getTreatmentPlansFromRows(rows)
}

func (d *DataService) GetTreatmentBasedOnPrescriptionID(erxID int64) (*common.Treatment, error) {
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			treatment_plan.patient_id, treatment_plan.doctor_id, is_controlled_substance from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where erx_id=? and localized_text.language_id = ?`, erxID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatments = append(treatments, treatment)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return nil, NoRowsError
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) GetTreatmentFromID(treatmentID int64) (*common.Treatment, error) {
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			treatment_plan.patient_id, treatment_plan.doctor_id, is_controlled_substance from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment.id=? and localized_text.language_id = ?`, treatmentID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatments = append(treatments, treatment)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return nil, nil
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) StartRXRoutingForTreatmentsAndTreatmentPlan(treatments []*common.Treatment, pharmacySentTo *pharmacyService.PharmacyData, treatmentPlanID, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	preparedStatement, err := tx.Prepare(`
		UPDATE treatment
		SET erx_id = ?, pharmacy_id = ?, erx_sent_date = now()
		WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer preparedStatement.Close()

	// update the treatments to add the prescription information
	for _, treatment := range treatments {
		if treatment.ERx != nil && treatment.ERx.PrescriptionID.Int64() != 0 {
			_, err = preparedStatement.Exec(treatment.ERx.PrescriptionID.Int64(), pharmacySentTo.LocalID, treatment.ID.Int64())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// update the status of the treatment plan
	_, err = tx.Exec(`
		UPDATE treatment_plan set status = ?
		WHERE id = ?`,
		common.TPStatusRXStarted.String(),
		treatmentPlanID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) UpdateTreatmentWithPharmacyAndErxID(treatments []*common.Treatment, pharmacySentTo *pharmacyService.PharmacyData, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		if treatment.ERx != nil && treatment.ERx.PrescriptionID.Int64() != 0 {
			_, err = tx.Exec(`update treatment set erx_id = ?, pharmacy_id = ?, erx_sent_date=now() where id = ?`, treatment.ERx.PrescriptionID.Int64(), pharmacySentTo.LocalID, treatment.ID.Int64())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (d *DataService) AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {

		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, STATUS_INACTIVE, treatment.ID.Int64(), STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		columnsAndData := make(map[string]interface{}, 0)
		columnsAndData["treatment_id"] = treatment.ID.Int64()
		columnsAndData["erx_status"] = prescriptionStatus.Status
		columnsAndData["status"] = STATUS_ACTIVE
		if !prescriptionStatus.ReportedTimestamp.IsZero() {
			columnsAndData["reported_timestamp"] = prescriptionStatus.ReportedTimestamp
		}
		if prescriptionStatus.StatusDetails != "" {
			columnsAndData["event_details"] = prescriptionStatus.StatusDetails
		}

		keys, values := getKeysAndValuesFromMap(columnsAndData)

		_, err = tx.Exec(fmt.Sprintf(`insert into erx_status_events (%s) values (%s)`, strings.Join(keys, ","), nReplacements(len(values))), values...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()

}

func (d *DataService) GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`select erx_status_events.treatment_id, treatment.erx_id, erx_status_events.erx_status, erx_status_events.creation_date from treatment
								inner join treatment_plan on treatment_plan_id = treatment_plan.id
								left outer join erx_status_events on erx_status_events.treatment_id = treatment.id
								inner join patient on patient.id = treatment_plan.patient_id
									where patient.erx_patient_id = ? and erx_status_events.status = ? order by erx_status_events.creation_date desc`, erxPatientID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptionStatuses := make([]common.StatusEvent, 0)
	for rows.Next() {
		var treatmentID int64
		var prescriptionID sql.NullInt64
		var status string
		var creationDate time.Time
		err = rows.Scan(&treatmentID, &prescriptionID, &status, &creationDate)
		if err != nil {
			return nil, err
		}

		prescriptionStatus := common.StatusEvent{
			Status:          status,
			ItemID:          treatmentID,
			StatusTimestamp: creationDate,
		}

		if prescriptionID.Valid {
			prescriptionStatus.PrescriptionID = prescriptionID.Int64
		}

		prescriptionStatuses = append(prescriptionStatuses, prescriptionStatus)
	}

	return prescriptionStatuses, rows.Err()
}

func (d *DataService) GetPrescriptionStatusEventsForTreatment(treatmentID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`select erx_status_events.treatment_id, erx_status_events.erx_status, erx_status_events.event_details, erx_status_events.creation_date
									  from erx_status_events where treatment_id = ? order by erx_status_events.creation_date desc`, treatmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptionStatuses := make([]common.StatusEvent, 0)
	for rows.Next() {
		var statusDetails sql.NullString
		var prescriptionStatus common.StatusEvent
		err = rows.Scan(&prescriptionStatus.ItemID, &prescriptionStatus.Status, &statusDetails, &prescriptionStatus.StatusTimestamp)

		if err != nil {
			return nil, err
		}
		prescriptionStatus.StatusDetails = statusDetails.String

		prescriptionStatuses = append(prescriptionStatuses, prescriptionStatus)
	}

	return prescriptionStatuses, rows.Err()
}

func (d *DataService) UpdateDateInfoForTreatmentId(treatmentID int64, erxSentDate time.Time) error {
	_, err := d.db.Exec(`update treatment set erx_sent_date = ? where treatment_id = ?`, erxSentDate, treatmentID)
	return err
}

func (d *DataService) MarkTPDeviatedFromContentSource(treatmentPlanID int64) error {
	_, err := d.db.Exec(`update treatment_plan_content_source set has_deviated = 1, deviated_date = now(6) where treatment_plan_id = ?`, treatmentPlanID)
	return err
}

func (d *DataService) GetOldestVisitsInStatuses(max int, statuses []string) ([]*ItemAge, error) {
	var whereClause string
	var params []interface{}

	if len(statuses) > 0 {
		whereClause = `WHERE status in (` + nReplacements(len(statuses)) + `)`
		params = appendStringsToInterfaceSlice(nil, statuses)
	}
	params = append(params, max)

	rows, err := d.db.Query(`
		SELECT id, last_modified_date 
		FROM patient_visit
		`+whereClause+`
		ORDER BY last_modified_date LIMIT ?`, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visitAges []*ItemAge
	for rows.Next() {
		var visitAge ItemAge
		var lastModifiedDate time.Time
		if err := rows.Scan(
			&visitAge.ID,
			&lastModifiedDate); err != nil {
			return nil, err
		}
		visitAge.Age = time.Since(lastModifiedDate)
		visitAges = append(visitAges, &visitAge)
	}

	return visitAges, rows.Err()
}

func (d *DataService) UpdateDiagnosisForVisit(id, doctorID int64, diagnosis string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// update any previous diagnosis for this case
	_, err = tx.Exec(`UPDATE visit_diagnosis SET status = ? WHERE patient_visit_id = ?`, STATUS_INACTIVE, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// track new diagnosis
	_, err = tx.Exec(`
		INSERT INTO visit_diagnosis (diagnosis, doctor_id, patient_visit_id, status) 
		VALUES (?,?,?,?)`, diagnosis, doctorID, id, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) DiagnosisForVisit(visitID int64) (string, error) {
	var diagnosis string
	err := d.db.QueryRow(`
		SELECT diagnosis 
		FROM visit_diagnosis 
		WHERE patient_visit_id = ? AND status = ?`, visitID, STATUS_ACTIVE).Scan(
		&diagnosis)

	if err == sql.ErrNoRows {
		return "", NoRowsError
	}

	return diagnosis, err
}

func (d *DataService) CreateDiagnosisSet(set *common.VisitDiagnosisSet) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// inactivate any previous diagnosis sets pertaining to this visit
	_, err = tx.Exec(`
		UPDATE visit_diagnosis_set
		SET active = 0
		WHERE patient_visit_id = ?
		AND active = 1
		`, set.VisitID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// create the new set
	res, err := tx.Exec(`
		INSERT INTO visit_diagnosis_set (patient_visit_id, doctor_id, notes, active, unsuitable, unsuitable_reason) 
		VALUES (?,?,?,?,?,?)`, set.VisitID, set.DoctorID, set.Notes, true, set.Unsuitable, set.UnsuitableReason)
	if err != nil {
		tx.Rollback()
		return err
	}

	set.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
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
			tx.Rollback()
			return err
		}
		defer insertItemStmt.Close()

		for _, item := range set.Items {
			res, err := insertItemStmt.Exec(set.ID, item.CodeID, item.LayoutVersionID)
			if err != nil {
				tx.Rollback()
				return err
			}

			item.ID, err = res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) ActiveDiagnosisSet(visitID int64) (*common.VisitDiagnosisSet, error) {
	var set common.VisitDiagnosisSet
	err := d.db.QueryRow(`
		SELECT id, doctor_id, patient_visit_id, notes, active, created, unsuitable, unsuitable_reason 
		FROM visit_diagnosis_set 
		WHERE patient_visit_id = ?
		AND active = 1`, visitID).Scan(
		&set.ID,
		&set.DoctorID,
		&set.VisitID,
		&set.Notes,
		&set.Active,
		&set.Created,
		&set.Unsuitable,
		&set.UnsuitableReason)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	// get the items in the set
	rows, err := d.db.Query(`
		SELECT id, diagnosis_code_id, layout_version_id 
		FROM visit_diagnosis_item
		WHERE visit_diagnosis_set_id = ?
		ORDER BY id`, set.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var setItems []*common.VisitDiagnosisItem
	for rows.Next() {
		var setItem common.VisitDiagnosisItem
		if err := rows.Scan(
			&setItem.ID,
			&setItem.CodeID,
			&setItem.LayoutVersionID); err != nil {
			return nil, err
		}
		setItems = append(setItems, &setItem)
	}
	set.Items = setItems
	return &set, rows.Err()
}

func (d *DataService) getTreatmentAndMetadataFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentID, treatmentPlanID, dispenseUnitId, patientID, prescriberId, prescriptionID, pharmacyID encoding.ObjectID
	var dispenseValue encoding.HighPrecisionFloat64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription string
	var status common.TreatmentStatus
	var substitutionsAllowed bool
	var refills, daysSupply encoding.NullInt64
	var creationDate time.Time
	var erxSentDate mysql.NullTime
	var isControlledSubstance sql.NullBool
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentID, &prescriptionID, &treatmentPlanID, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId,
		&dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyID,
		&pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &status, &drugName, &drugRoute, &drugForm, &patientID, &prescriberId, &isControlledSubstance)
	if err != nil {
		return nil, err
	}

	treatment := &common.Treatment{
		ID:                      treatmentID,
		PatientID:               patientID,
		DrugInternalName:        drugInternalName,
		DosageStrength:          dosageStrength,
		DispenseValue:           dispenseValue,
		DispenseUnitID:          dispenseUnitId,
		DispenseUnitDescription: dispenseUnitDescription,
		NumberRefills:           refills,
		SubstitutionsAllowed:    substitutionsAllowed,
		DaysSupply:              daysSupply,
		DrugName:                drugName.String,
		DrugForm:                drugForm.String,
		DrugRoute:               drugRoute.String,
		PatientInstructions:     patientInstructions,
		CreationDate:            &creationDate,
		Status:                  status,
		PharmacyNotes:           pharmacyNotes.String,
		DoctorID:                prescriberId,
		TreatmentPlanID:         treatmentPlanID,
		IsControlledSubstance:   isControlledSubstance.Bool,
	}
	if treatmentType == treatmentOTC {
		treatment.OTC = true
	}

	if pharmacyID.IsValid || prescriptionID.IsValid || erxSentDate.Valid {
		treatment.ERx = &common.ERxData{}
		treatment.ERx.PharmacyLocalID = pharmacyID
		treatment.ERx.PrescriptionID = prescriptionID
	}

	if erxSentDate.Valid {
		treatment.ERx.ErxSentDate = &erxSentDate.Time
	}

	err = d.fillInDrugDBIdsForTreatment(treatment, treatment.ID.Int64(), possibleTreatmentTables[treatmentForPatientType])
	if err != nil {
		return nil, err
	}

	err = d.fillInSupplementalInstructionsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	// if its null that means that there isn't any erx related information
	if treatment.ERx != nil {
		treatment.ERx.RxHistory, err = d.GetPrescriptionStatusEventsForTreatment(treatment.ID.Int64())
		if err != nil {
			return nil, err
		}

		treatment.ERx.Pharmacy, err = d.GetPharmacyFromID(treatment.ERx.PharmacyLocalID.Int64())
		if err != nil {
			return nil, err
		}

	}

	treatment.Doctor, err = d.GetDoctorFromID(treatment.DoctorID.Int64())
	if err != nil {
		return nil, err
	}

	treatment.Patient, err = d.GetPatientFromID(treatment.PatientID.Int64())
	if err != nil {
		return nil, err
	}
	return treatment, nil
}

func (d *DataService) fillInDrugDBIdsForTreatment(treatment *common.Treatment, id int64, tableName string) error {
	// for each of the drugs, populate the drug db ids
	drugDbIds := make(map[string]string)
	drugRows, err := d.db.Query(fmt.Sprintf(`select drug_db_id_tag, drug_db_id from %s_drug_db_id where %s_id = ? `, tableName, tableName), id)
	if err != nil {
		return err
	}
	defer drugRows.Close()

	for drugRows.Next() {
		var dbIdTag string
		var dbId string
		if err := drugRows.Scan(&dbIdTag, &dbId); err != nil {
			return err
		}
		drugDbIds[dbIdTag] = dbId
	}

	treatment.DrugDBIDs = drugDbIds
	return nil
}

func (d *DataService) fillInSupplementalInstructionsForTreatment(treatment *common.Treatment) error {
	// get the supplemental instructions for this treatment
	instructionsRows, err := d.db.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions 
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id 
													where treatment_instructions.status=? and treatment_id=?`, STATUS_ACTIVE, treatment.ID.Int64())
	if err != nil {
		return err
	}
	defer instructionsRows.Close()

	drugInstructions := make([]*common.DoctorInstructionItem, 0)
	for instructionsRows.Next() {
		var instructionId encoding.ObjectID
		var text string
		if err := instructionsRows.Scan(&instructionId, &text); err != nil {
			return err
		}
		drugInstruction := &common.DoctorInstructionItem{
			ID:       instructionId,
			Text:     text,
			Selected: true,
		}
		drugInstructions = append(drugInstructions, drugInstruction)
	}
	treatment.SupplementalInstructions = drugInstructions
	return nil
}

func getRegimenPlanFromRows(rows *sql.Rows) (*common.RegimenPlan, error) {
	// keep track of the ordering of the regimenSections
	var regimenSectionNames []string
	regimenSections := make(map[string][]*common.DoctorInstructionItem)
	for rows.Next() {
		var regimenType, regimenText string
		var regimenId, parentID encoding.ObjectID
		err := rows.Scan(&regimenId, &regimenType, &parentID, &regimenText)
		if err != nil {
			return nil, err
		}
		regimenStep := &common.DoctorInstructionItem{
			ID:       regimenId,
			Text:     regimenText,
			ParentID: parentID,
		}

		// keep track of the unique regimen sections as they appear
		if _, ok := regimenSections[regimenType]; !ok {
			regimenSectionNames = append(regimenSectionNames, regimenType)
		}
		regimenSections[regimenType] = append(regimenSections[regimenType], regimenStep)

	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	regimenSectionsArray := make([]*common.RegimenSection, 0)
	// create the regimen sections
	for _, regimenSectionName := range regimenSectionNames {
		regimenSection := &common.RegimenSection{
			Name:  regimenSectionName,
			Steps: regimenSections[regimenSectionName],
		}
		regimenSectionsArray = append(regimenSectionsArray, regimenSection)
	}

	return &common.RegimenPlan{Sections: regimenSectionsArray}, nil
}

func getAdvicePointsFromRows(rows *sql.Rows) ([]*common.DoctorInstructionItem, error) {
	advicePoints := make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id, parentID encoding.ObjectID
		var text string
		if err := rows.Scan(&id, &parentID, &text); err != nil {
			return nil, err
		}

		advicePoint := &common.DoctorInstructionItem{
			ID:       id,
			ParentID: parentID,
			Text:     text,
		}
		advicePoints = append(advicePoints, advicePoint)
	}
	return advicePoints, rows.Err()
}

func getTreatmentPlansFromRows(rows *sql.Rows) ([]*common.TreatmentPlan, error) {
	var treatmentPlans []*common.TreatmentPlan
	for rows.Next() {
		var treatmentPlan common.TreatmentPlan
		if err := rows.Scan(
			&treatmentPlan.ID, &treatmentPlan.DoctorID, &treatmentPlan.PatientCaseID,
			&treatmentPlan.PatientID, &treatmentPlan.CreationDate, &treatmentPlan.Status,
		); err != nil {
			return nil, err
		}
		treatmentPlans = append(treatmentPlans, &treatmentPlan)
	}

	return treatmentPlans, rows.Err()
}
