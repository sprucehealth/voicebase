package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
	pharmacyService "github.com/sprucehealth/backend/pharmacy"
)

var treatmentQuery = `
	SELECT t.id, t.erx_id, t.treatment_plan_id, t.drug_internal_name,
		t.dosage_strength, t.type, t.dispense_value, t.dispense_unit_id,
		ltext, t.refills, t.substitutions_allowed, t.days_supply,
		t.pharmacy_id, COALESCE(t.pharmacy_notes, ''), t.patient_instructions,
		t.creation_date, t.erx_sent_date, t.status, dn.name,
		dr.name, df.name, tp.patient_id, tp.doctor_id,
		COALESCE(is_controlled_substance, false), COALESCE(dn2.name, '')
	FROM treatment t
	INNER JOIN treatment_plan tp ON tp.id = t.treatment_plan_id
	INNER JOIN dispense_unit du ON du.id = t.dispense_unit_id
	INNER JOIN localized_text lt ON lt.app_text_id = du.dispense_unit_text_id
	INNER JOIN drug_name dn ON dn.id = drug_name_id
	LEFT JOIN drug_name dn2 ON dn2.id = generic_drug_name_id
	INNER JOIN drug_route dr ON dr.id = drug_route_id
	INNER JOIN drug_form df ON df.id = drug_form_id
`

var visitSummaryQuery = `
	SELECT p.account_id, pv.id, pv.patient_case_id, pv.creation_date, pv.submitted_date, cpa.creation_date,
			role_type.role_type_tag, cpa.status, cp.name, pc.requested_doctor_id, p.first_name, p.last_name,
			pc.name, sku.type, pl.state, pv.status, doctor.id, doctor.first_name, doctor.last_name
		FROM patient_visit pv
		JOIN patient_case pc ON pv.patient_case_id = pc.id
		JOIN clinical_pathway cp ON pv.clinical_pathway_id = cp.id
		JOIN sku ON pv.sku_id = sku.id
		JOIN patient p ON pv.patient_id = p.id
		LEFT JOIN patient_location pl ON pv.patient_id = pl.patient_id
		LEFT JOIN patient_case_care_provider_assignment cpa ON pv.patient_case_id = cpa.patient_case_id
		LEFT JOIN doctor ON cpa.provider_id = doctor.id
		LEFT JOIN role_type ON role_type_id = role_type.id
	`

func (d *DataService) GetPatientIDFromPatientVisitID(patientVisitID int64) (int64, error) {
	var patientID int64
	err := d.db.QueryRow("select patient_id from patient_visit where id = ?", patientVisitID).Scan(&patientID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("patient_visit")
	}
	return patientID, err
}

// Adding this only to link the patient and the doctor app so as to show the doctor
// the patient visit review of the latest submitted patient visit
func (d *DataService) GetLatestSubmittedPatientVisit() (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, clinical_pathway_id, layout_version_id,
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
		return nil, ErrNotFound("patient_visit")
	case l == 1:
		return patientVisits[0], nil
	}

	return nil, fmt.Errorf("expected 1 patient visit but got %d", len(patientVisits))
}

func (d *DataService) PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error) {
	// get the creation time of the initial visit
	var creationDate time.Time
	err := d.db.QueryRow(`
		SELECT creation_date
		FROM patient_visit
		WHERE patient_case_id = ?
		ORDER BY id LIMIT 1`, caseID).Scan(&creationDate)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_visit")
	} else if err != nil {
		return nil, err
	}

	// look for a pending followup visit created after the initial visit
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, clinical_pathway_id,
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

func (d *DataService) GetPatientVisitForSKU(patientID int64, skuType string) (*common.PatientVisit, error) {
	skuID, err := d.skuIDFromType(skuType)
	if err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, clinical_pathway_id,
		layout_version_id, creation_date, submitted_date, closed_date,
		status, sku_id, followup
		FROM patient_visit
	 	WHERE patient_id = ? AND sku_id = ?
	 	LIMIT 1
		`, patientID, skuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getSinglePatientVisit(rows)
}

func (d *DataService) GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, clinical_pathway_id, layout_version_id,
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
		SELECT pv.id, pv.patient_id, pv.patient_case_id, pv.clinical_pathway_id,
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

func (d *DataService) VisitsSubmittedForPatientSince(patientID int64, since time.Time) ([]*common.PatientVisit, error) {
	vals := []interface{}{patientID, since}
	vals = dbutil.AppendStringsToInterfaceSlice(vals, common.NonOpenPatientVisitStates())

	rows, err := d.db.Query(`
		SELECT pv.id, pv.patient_id, pv.patient_case_id, pv.clinical_pathway_id,
		pv.layout_version_id, pv.creation_date, pv.submitted_date, pv.closed_date,
		pv.status, pv.sku_id, pv.followup
		FROM patient_visit as pv
		WHERE pv.patient_id = ?
		and submitted_date >= ?
		and pv.status IN (`+dbutil.MySQLArgs(len(common.NonOpenPatientVisitStates()))+`)
		ORDER BY submitted_date DESC`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getPatientVisitFromRows(rows)
}

func (d *DataService) getSinglePatientVisit(rows *sql.Rows) (*common.PatientVisit, error) {
	patientVisits, err := d.getPatientVisitFromRows(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(patientVisits); {
	case l == 0:
		return nil, ErrNotFound("patient_visit")
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
		var pathwayID int64
		err := rows.Scan(
			&patientVisit.PatientVisitID,
			&patientVisit.PatientID,
			&patientVisit.PatientCaseID,
			&pathwayID,
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
		patientVisit.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, err
		}
		patientVisit.SubmittedDate = submittedDate.Time
		patientVisit.ClosedDate = closedDate.Time
		patientVisit.SKUType, err = d.skuTypeFromID(skuID)
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
		return 0, ErrNotFound("patient_visit")
	} else if err != nil {
		return 0, err
	}
	return patientCaseID, nil
}

func (d *DataService) CreatePatientVisit(visit *common.PatientVisit, requestedDoctorID *int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	caseID := visit.PatientCaseID.Int64()
	if caseID == 0 {
		// implicitly create a new case when creating a new visit for now
		// for now treating the creation of every new case as an unclaimed case because we don't have a notion of a
		// new case for which the patient returns (and thus can be potentially claimed)
		patientCase := &common.PatientCase{
			PatientID:         encoding.NewObjectID(visit.PatientID.Int64()),
			PathwayTag:        visit.PathwayTag,
			Status:            common.PCStatusOpen,
			RequestedDoctorID: requestedDoctorID,
		}

		if err := d.createPatientCase(tx, patientCase); err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}

		caseID = patientCase.ID.Int64()
	}

	pathwayID, err := d.pathwayIDFromTag(visit.PathwayTag)
	if err != nil {
		return 0, err
	}

	skuID, err := d.skuIDFromType(visit.SKUType)
	if err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		INSERT INTO patient_visit (patient_id, clinical_pathway_id, layout_version_id, patient_case_id, status, sku_id, followup)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		visit.PatientID.Int64(), pathwayID, visit.LayoutVersionID.Int64(), caseID,
		visit.Status, &skuID, visit.IsFollowup)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	visit.SKUType, err = d.skuTypeFromID(skuID)
	if err != nil {
		return 0, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Trace(err)
	}

	visit.CreationDate = time.Now()
	visit.PatientVisitID = encoding.NewObjectID(lastID)
	visit.PatientCaseID = encoding.NewObjectID(caseID)
	return lastID, nil
}

func (d *DataService) GetMessageForPatientVisit(patientVisitID int64) (string, error) {
	var message string
	if err := d.db.QueryRow(`SELECT message FROM patient_visit_message WHERE patient_visit_id = ?`, patientVisitID).Scan(&message); err == sql.ErrNoRows {
		return "", ErrNotFound("patient_visit_message")
	} else if err != nil {
		return "", errors.Trace(err)
	}
	return message, nil
}

func (d *DataService) SetMessageForPatientVisit(patientVisitID int64, message string) error {
	_, err := d.db.Exec(`REPLACE INTO patient_visit_message (patient_visit_id, message) VALUES (?,?) `, patientVisitID, message)
	return errors.Trace(err)
}

func (d *DataService) VisitSummaries(visitStatuses []string, from, to time.Time) ([]*common.VisitSummary, error) {
	q := visitSummaryQuery
	var values []interface{}
	conditions := make([]string, 0, 3)
	if len(visitStatuses) > 0 {
		conditions = append(conditions, ` pv.status IN (`+dbutil.MySQLArgs(len(visitStatuses))+`)`)
		values = dbutil.AppendStringsToInterfaceSlice(values, visitStatuses)
	}
	if !from.IsZero() {
		conditions = append(conditions, ` pv.creation_date >= ?`)
		values = append(values, from)
	}
	if !to.IsZero() {
		conditions = append(conditions, ` pv.creation_date <= ?`)
		values = append(values, to)
	}
	if len(conditions) > 0 {
		q += `WHERE` + strings.Join(conditions, ` AND `)
	}
	rows, err := d.db.Query(q, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summariesMap, err := d.sanitizeVisitSummaryRows(rows)
	if err != nil {
		return nil, err
	}

	summaries := make([]*common.VisitSummary, len(summariesMap))
	i := 0
	for _, v := range summariesMap {
		summaries[i] = v
		i++
	}
	return summaries, rows.Err()
}

func (d *DataService) VisitSummary(visitID int64) (*common.VisitSummary, error) {
	rows, err := d.db.Query(visitSummaryQuery+` WHERE pv.id = ?`, visitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summariesMap, err := d.sanitizeVisitSummaryRows(rows)
	if err != nil {
		return nil, err
	}

	if len(summariesMap) != 1 {
		return nil, fmt.Errorf("Expected to find only 1 collapsed row for visit id %d but found %d", visitID, len(summariesMap))
	}

	for _, v := range summariesMap {
		return v, nil
	}

	return nil, fmt.Errorf("Expected to find at lease 1 element in summary map but apparently found 0")
}

// If we encounter the same visit twice then we just need to make sure we have the information related to the actual physician
func (d *DataService) sanitizeVisitSummaryRows(rows *sql.Rows) (map[int64]*common.VisitSummary, error) {
	summariesMap := make(map[int64]*common.VisitSummary)
	for rows.Next() {
		sm := &common.VisitSummary{}
		if err := rows.Scan(
			&sm.PatientAccountID, &sm.VisitID, &sm.CaseID, &sm.CreationDate, &sm.SubmittedDate,
			&sm.LockTakenDate, &sm.RoleTypeTag, &sm.LockType, &sm.PathwayName, &sm.RequestedDoctorID,
			&sm.PatientFirstName, &sm.PatientLastName, &sm.CaseName, &sm.SKUType, &sm.SubmissionState,
			&sm.Status, &sm.DoctorID, &sm.DoctorFirstName, &sm.DoctorLastName,
		); err != nil {
			return nil, err
		}

		if sm.RoleTypeTag != nil && *sm.RoleTypeTag != "DOCTOR" {
			sm.DoctorLastName = nil
			sm.DoctorFirstName = nil
			sm.DoctorID = nil
			sm.LockType = nil
			sm.RoleTypeTag = nil
			_, ok := summariesMap[sm.VisitID]
			if !ok {
				summariesMap[sm.VisitID] = sm
			}
		} else {
			summariesMap[sm.VisitID] = sm
		}
	}
	return summariesMap, rows.Err()
}

func (d *DataService) GetAbridgedTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_id, patient_case_id, status, creation_date, sent_date, note, patient_viewed
		FROM treatment_plan
		WHERE id = ?`, treatmentPlanID)
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
		return nil, ErrNotFound("treatment_plan")
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
		return false, ErrNotFound("treatment_plan")
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

func (d *DataService) UpdateTreatmentPlan(treatmentPlanID int64, update *TreatmentPlanUpdate) error {

	var cols []string
	var vals []interface{}

	if update.PatientViewed != nil {
		cols = append(cols, "patient_viewed = ?")
		vals = append(vals, *update.PatientViewed)
	}
	if update.Status != nil {
		cols = append(cols, "status = ?")
		vals = append(vals, update.Status.String())
	}

	if len(cols) == 0 {
		return nil
	}

	vals = append(vals, treatmentPlanID)

	_, err := d.db.Exec(`
		UPDATE treatment_plan
		SET `+strings.Join(cols, ",")+`
		WHERE id = ?`, vals...)
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

	// resource guides
	treatmentPlan.ResourceGuides, err = d.ListTreatmentPlanResourceGuides(treatmentPlanID)
	if err != nil {
		return nil, err
	}

	// scheduled messages
	treatmentPlan.ScheduledMessages, err = d.ListTreatmentPlanScheduledMessages(treatmentPlanID)
	if err != nil {
		return nil, err
	}

	return treatmentPlan, nil
}

func (d *DataService) getAbridgedTreatmentPlanFromRows(rows *sql.Rows, doctorID int64) ([]*common.TreatmentPlan, error) {
	var tpList []*common.TreatmentPlan
	for rows.Next() {
		var tp common.TreatmentPlan
		var note sql.NullString
		if err := rows.Scan(
			&tp.ID,
			&tp.DoctorID,
			&tp.PatientID,
			&tp.PatientCaseID,
			&tp.Status,
			&tp.CreationDate,
			&tp.SentDate,
			&note,
			&tp.PatientViewed); err != nil {
			return nil, err
		}
		tp.Note = note.String

		// parent information has to exist for every treatment plan; so if it doesn't that means something is logically inconsistent
		tp.Parent = &common.TreatmentPlanParent{}
		err := d.db.QueryRow(`
			SELECT parent_id, parent_type
			FROM treatment_plan_parent
			WHERE treatment_plan_id = ?`,
			tp.ID.Int64()).Scan(&tp.Parent.ParentID, &tp.Parent.ParentType)
		if err == sql.ErrNoRows {
			return nil, ErrNotFound("treatment_plan_parent")
		} else if err != nil {
			return nil, err
		}

		// get the creation date of the parent since this information is useful for the client
		var creationDate time.Time
		switch tp.Parent.ParentType {
		case common.TPParentTypePatientVisit:
			if err := d.db.QueryRow(`
				SELECT creation_date
				FROM patient_visit
				WHERE id = ?`, tp.Parent.ParentID.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, ErrNotFound("patient_visit")
			} else if err != nil {
				return nil, err
			}
		case common.TPParentTypeTreatmentPlan:
			if err := d.db.QueryRow(`
				SELECT creation_date
				FROM treatment_plan
				WHERE id = ?`, tp.Parent.ParentID.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, ErrNotFound("treatment_plan")
			} else if err != nil {
				return nil, err
			}
		}
		tp.Parent.CreationDate = creationDate

		tp.ContentSource = &common.TreatmentPlanContentSource{}
		err = d.db.QueryRow(`
			SELECT content_source_id, content_source_type, has_deviated
			FROM treatment_plan_content_source
			WHERE treatment_plan_id = ? and doctor_id = ?`,
			tp.ID.Int64(), doctorID,
		).Scan(
			&tp.ContentSource.ID,
			&tp.ContentSource.Type,
			&tp.ContentSource.HasDeviated)
		if err == sql.ErrNoRows {
			// treat content source as empty if non specified
			tp.ContentSource = nil
		} else if err != nil {
			return nil, err
		}

		tpList = append(tpList, &tp)
	}
	return tpList, rows.Err()
}

func (d *DataService) GetAbridgedTreatmentPlanList(doctorID, patientCaseID int64, statuses []common.TreatmentPlanStatus) ([]*common.TreatmentPlan, error) {
	where := "patient_case_id = ?"
	vals := []interface{}{patientCaseID}

	if l := len(statuses); l > 0 {
		where += " AND status in (" + dbutil.MySQLArgs(l) + ")"
		for _, sItem := range statuses {
			vals = append(vals, sItem.String())
		}
	}

	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_id, patient_case_id, status, creation_date, sent_date, note, patient_viewed
		FROM treatment_plan
		WHERE `+where, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getAbridgedTreatmentPlanFromRows(rows, doctorID)
}

func (d *DataService) GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, patientCaseID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_id, patient_case_id, status, creation_date, sent_date, note, patient_viewed
		FROM treatment_plan
		WHERE doctor_id = ? AND patient_case_id = ? AND status = ?`,
		doctorID, patientCaseID, common.TPStatusDraft.String())
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
		return 0, ErrNotFound("treatment_plan")
	}

	return patientID, err
}

func (d *DataService) GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error) {
	var patientVisitID int64
	err := d.db.QueryRow(`
		SELECT patient_visit_id
		FROM treatment_plan_patient_visit_mapping
		WHERE treatment_plan_id = ?`, treatmentPlanID).Scan(&patientVisitID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("treatment_plan_patient_visit_mapping")
	}

	return patientVisitID, nil
}

func (d *DataService) StartNewTreatmentPlan(patientVisitID int64, tp *common.TreatmentPlan) (int64, error) {
	// validation
	if tp == nil {
		return 0, errors.New("missing tp information")
	}
	if tp.Parent == nil {
		return 0, errors.New("missing tp parent information")
	}
	if tp.DoctorID.Int64() == 0 {
		return 0, errors.New("missing doctor_id")
	}
	if tp.PatientID == 0 {
		return 0, errors.New("missing patient_id")
	}
	if tp.PatientCaseID.Int64() == 0 {
		return 0, errors.New("missing patient_case_id")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	// Delete any existing draft treatment plan matching the doctor, patient, source, and parent
	_, err = tx.Exec(`
		DELETE FROM treatment_plan
		WHERE id = (SELECT treatment_plan_id FROM treatment_plan_parent WHERE parent_id = ? AND parent_type = ?)
			AND status = ? AND doctor_id = ?`,
		tp.Parent.ParentID.Int64(), tp.Parent.ParentType, common.TPStatusDraft.String(), tp.DoctorID.Int64())
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastID, err := tx.Exec(`
		INSERT INTO treatment_plan
		(patient_id, doctor_id, patient_case_id, status, note)
		VALUES (?,?,?,?,?)`, tp.PatientID, tp.DoctorID.Int64(), tp.PatientCaseID.Int64(), common.TPStatusDraft.String(), tp.Note)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	treatmentPlanID, err := lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	tp.ID = encoding.NewObjectID(treatmentPlanID)

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
	_, err = tx.Exec(`
		INSERT INTO treatment_plan_parent
			(treatment_plan_id, parent_id, parent_type) VALUES (?,?,?)`,
		treatmentPlanID, tp.Parent.ParentID.Int64(), tp.Parent.ParentType)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the original content source for the treatment plan
	if tp.ContentSource != nil {
		_, err := tx.Exec(`
			INSERT INTO treatment_plan_content_source
				(treatment_plan_id, doctor_id, content_source_id, content_source_type)
			VALUES (?,?,?,?)`,
			treatmentPlanID, tp.DoctorID.Int64(), tp.ContentSource.ID.Int64(), tp.ContentSource.Type)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if tp.TreatmentList != nil {
		if err := d.addTreatmentsForTreatmentPlan(tx,
			tp.TreatmentList.Treatments,
			tp.DoctorID.Int64(),
			tp.ID.Int64(),
			tp.PatientID); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if tp.RegimenPlan != nil {
		tp.RegimenPlan.TreatmentPlanID = encoding.NewObjectID(treatmentPlanID)
		if err := createRegimenPlan(
			tx,
			tp.RegimenPlan); err != nil {
			return 0, err
		}
	}

	// create scheduled messages
	for _, tpSchedMsg := range tp.ScheduledMessages {
		tpSchedMsg.TreatmentPlanID = treatmentPlanID
		if _, err := d.createTreatmentPlanScheduledMessage(
			tx,
			"treatment_plan",
			common.ClaimerTypeTreatmentPlanScheduledMessage,
			0,
			tpSchedMsg); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// create resource guides
	guideIDs := make([]int64, len(tp.ResourceGuides))
	for i, resourceGuide := range tp.ResourceGuides {
		guideIDs[i] = resourceGuide.ID
	}

	if err := addResourceGuidesToTreatmentPlan(tx, treatmentPlanID, guideIDs); err != nil {
		tx.Rollback()
		return 0, err
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
	_, err = tx.Exec(`update treatment_plan set status = ?, sent_date = now() where id = ?`, common.TPStatusActive.String(), treatmentPlan.ID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) SubmitPatientVisitWithID(patientVisitID int64) error {
	_, err := d.db.Exec("UPDATE patient_visit SET status='SUBMITTED', submitted_date=now() WHERE id = ? AND STATUS IN ('OPEN', 'PHOTOS_REJECTED')", patientVisitID)
	return err
}

func (d *DataService) CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := createRegimenPlan(tx, regimenPlan); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func createRegimenPlan(tx *sql.Tx, regimenPlan *common.RegimenPlan) error {
	tpID := regimenPlan.TreatmentPlanID.Int64()
	// delete any previous steps and sections given that we have new ones coming in
	_, err := tx.Exec(`
		DELETE FROM regimen
		WHERE treatment_plan_id = ?`, tpID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		DELETE FROM regimen_section
		WHERE treatment_plan_id = ?`, tpID)
	if err != nil {
		return err
	}

	secStmt, err := tx.Prepare(`
		INSERT INTO regimen_section
		(treatment_plan_id, title) VALUES (?,?)`)
	if err != nil {
		return err
	}
	defer secStmt.Close()

	stepStmt, err := tx.Prepare(`
		INSERT INTO regimen
		(treatment_plan_id, regimen_section_id, dr_regimen_step_id, text, status) VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stepStmt.Close()

	// create new regimen steps within each section
	for _, section := range regimenPlan.Sections {
		res, err := secStmt.Exec(tpID, section.Name)
		if err != nil {
			return err
		}
		secID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		for _, step := range section.Steps {
			_, err = stepStmt.Exec(
				tpID,
				secID,
				step.ParentID.Int64Ptr(),
				step.Text,
				StatusActive)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *DataService) GetRegimenPlanForTreatmentPlan(treatmentPlanID int64) (*common.RegimenPlan, error) {
	rows, err := d.db.Query(`
		SELECT regimen.id, rs.title, dr_regimen_step_id, text
		FROM regimen
		INNER JOIN regimen_section rs ON rs.id = regimen_section_id
		WHERE regimen.treatment_plan_id = ?
			AND status = ?
		ORDER BY regimen.id`, treatmentPlanID, StatusActive)
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

	if err := d.addTreatmentsForTreatmentPlan(tx, treatments, doctorID, treatmentPlanID, patientID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) addTreatmentsForTreatmentPlan(tx *sql.Tx, treatments []*common.Treatment, doctorID, tpID, patientID int64) error {
	_, err := tx.Exec(`
		UPDATE treatment
		SET status = ?
		WHERE treatment_plan_id = ?`, common.TStatusInactive.String(), tpID)
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		treatment.TreatmentPlanID = encoding.NewObjectID(tpID)
		err = d.addTreatment(treatmentForPatientType, treatment, nil, tx)
		if err != nil {
			return err
		}

		if treatment.DoctorTreatmentTemplateID.Int64() != 0 {
			_, err = tx.Exec(`
				INSERT INTO treatment_dr_template_selection
				(treatment_id, dr_treatment_template_id) VALUES (?,?)`, treatment.ID.Int64(), treatment.DoctorTreatmentTemplateID.Int64())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *DataService) GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID int64) ([]*common.Treatment, error) {
	// get treatment plan information
	rows, err := d.db.Query(
		treatmentQuery+`
		WHERE treatment_plan_id = ?
			AND t.status = ?
			AND lt.language_id = ?`,
		treatmentPlanID, common.TStatusCreated.String(), LanguageIDEnglish)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var treatments []*common.Treatment
	var treatmentIDs []int64
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatment.TreatmentPlanID = encoding.NewObjectID(treatmentPlanID)
		treatments = append(treatments, treatment)
		treatmentIDs = append(treatmentIDs, treatment.ID.Int64())
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return treatments, nil
	}

	favoriteRows, err := d.db.Query(fmt.Sprintf(`
		SELECT dr_treatment_template_id, treatment_dr_template_selection.treatment_id
		FROM treatment_dr_template_selection
		INNER JOIN dr_treatment_template ON dr_treatment_template.id = dr_treatment_template_id
		WHERE treatment_dr_template_selection.treatment_id IN (%s)
			AND dr_treatment_template.status = ?`,
		enumerateItemsIntoString(treatmentIDs)), common.TStatusCreated.String())
	treatmentIDToFavoriteIDMapping := make(map[int64]int64)
	if err != nil {
		return nil, err
	}
	defer favoriteRows.Close()

	for favoriteRows.Next() {
		var drFavoriteTreatmentID, treatmentID int64
		err = favoriteRows.Scan(&drFavoriteTreatmentID, &treatmentID)
		if err != nil {
			return nil, err
		}
		treatmentIDToFavoriteIDMapping[treatmentID] = drFavoriteTreatmentID
	}

	// assign the treatments the doctor favorite id if one exists
	for _, treatment := range treatments {
		if treatmentIDToFavoriteIDMapping[treatment.ID.Int64()] != 0 {
			treatment.DoctorTreatmentTemplateID = encoding.NewObjectID(treatmentIDToFavoriteIDMapping[treatment.ID.Int64()])
		}
	}

	return treatments, nil
}

func (d *DataService) GetTreatmentsForPatient(patientID int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(
		treatmentQuery+`
		WHERE tp.patient_id = ?
			AND t.status = ?
			AND lt.language_id = ?`,
		patientID, common.TStatusCreated.String(), LanguageIDEnglish)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// get treatment plan information
	var treatments []*common.Treatment
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
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status, patient_viewed, sent_date
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
		return nil, ErrNotFound("treatment_plan")
	case l > 1:
		return nil, fmt.Errorf("Expected 1 treatment plan instead got %d", len(treatmentPlans))
	}

	tp := treatmentPlans[0]
	if tp.PatientID != patientID {
		return nil, ErrNotFound("treatment_plan")
	}
	return tp, nil
}

func (d *DataService) GetActiveTreatmentPlansForPatient(patientID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status, patient_viewed, sent_date
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
	rows, err := d.db.Query(
		treatmentQuery+`
		WHERE erx_id = ? AND lt.language_id = ?`,
		erxID, LanguageIDEnglish)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var treatments []*common.Treatment
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
		return nil, ErrNotFound("treatment")
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) GetTreatmentFromID(treatmentID int64) (*common.Treatment, error) {
	rows, err := d.db.Query(
		treatmentQuery+`
		WHERE t.id = ? AND lt.language_id = ?`,
		treatmentID, LanguageIDEnglish)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var treatments []*common.Treatment
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

		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, StatusInactive, treatment.ID.Int64(), StatusActive)
		if err != nil {
			tx.Rollback()
			return err
		}

		columnsAndData := map[string]interface{}{
			"treatment_id": treatment.ID.Int64(),
			"erx_status":   prescriptionStatus.Status,
			"status":       StatusActive,
		}
		if !prescriptionStatus.ReportedTimestamp.IsZero() {
			columnsAndData["reported_timestamp"] = prescriptionStatus.ReportedTimestamp
		}
		if prescriptionStatus.StatusDetails != "" {
			columnsAndData["event_details"] = prescriptionStatus.StatusDetails
		}

		keys, values := getKeysAndValuesFromMap(columnsAndData)

		_, err = tx.Exec(fmt.Sprintf(`insert into erx_status_events (%s) values (%s)`, strings.Join(keys, ","), dbutil.MySQLArgs(len(values))), values...)
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
									where patient.erx_patient_id = ? and erx_status_events.status = ? order by erx_status_events.creation_date desc`, erxPatientID, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prescriptionStatuses []common.StatusEvent
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
	rows, err := d.db.Query(`
		SELECT erx_status_events.treatment_id, erx_status_events.erx_status, erx_status_events.event_details, erx_status_events.creation_date
		FROM erx_status_events 
		WHERE treatment_id = ? 
		ORDER BY erx_status_events.creation_date desc, erx_status_events.id DESC`, treatmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prescriptionStatuses []common.StatusEvent
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

func (d *DataService) UpdateDateInfoForTreatmentID(treatmentID int64, erxSentDate time.Time) error {
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
		whereClause = `WHERE status in (` + dbutil.MySQLArgs(len(statuses)) + `)`
		params = dbutil.AppendStringsToInterfaceSlice(nil, statuses)
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
	_, err = tx.Exec(`UPDATE visit_diagnosis SET status = ? WHERE patient_visit_id = ?`, StatusInactive, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// track new diagnosis
	_, err = tx.Exec(`
		INSERT INTO visit_diagnosis (diagnosis, doctor_id, patient_visit_id, status)
		VALUES (?,?,?,?)`, diagnosis, doctorID, id, StatusActive)
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
		WHERE patient_visit_id = ? AND status = ?`, visitID, StatusActive).Scan(
		&diagnosis)

	if err == sql.ErrNoRows {
		return "", ErrNotFound("visit_diagnosis")
	}

	return diagnosis, err
}

func (d *DataService) AddAlertsForVisit(visitID int64, alerts []*common.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	fields := make([]string, 0, len(alerts))
	values := make([]interface{}, 0, 3*len(alerts))
	for _, alert := range alerts {
		values = append(values, alert.VisitID, alert.Message, alert.QuestionID)
		fields = append(fields, "(?,?,?)")
	}

	_, err := d.db.Exec(`INSERT INTO patient_alerts (patient_visit_id, alert, question_id) VALUES `+strings.Join(fields, ","), values...)
	return err
}

func (d *DataService) AlertsForVisit(visitID int64) ([]*common.Alert, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_visit_id, creation_date, alert, question_id
		FROM patient_alerts WHERE patient_visit_id = ?`, visitID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var alerts []*common.Alert
	for rows.Next() {
		alert := &common.Alert{}
		if err := rows.Scan(&alert.ID, &alert.VisitID, &alert.CreationDate, &alert.Message, &alert.QuestionID); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
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
		return nil, ErrNotFound("visit_diagnosis_set")
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
	treatment := &common.Treatment{}

	var prescriptionID, pharmacyID encoding.ObjectID
	var treatmentType string
	var erxSentDate mysql.NullTime
	err := rows.Scan(
		&treatment.ID, &prescriptionID, &treatment.TreatmentPlanID, &treatment.DrugInternalName,
		&treatment.DosageStrength, &treatmentType, &treatment.DispenseValue, &treatment.DispenseUnitID,
		&treatment.DispenseUnitDescription, &treatment.NumberRefills, &treatment.SubstitutionsAllowed,
		&treatment.DaysSupply, &pharmacyID, &treatment.PharmacyNotes, &treatment.PatientInstructions,
		&treatment.CreationDate, &erxSentDate, &treatment.Status, &treatment.DrugName, &treatment.DrugRoute,
		&treatment.DrugForm, &treatment.PatientID, &treatment.DoctorID, &treatment.IsControlledSubstance,
		&treatment.GenericDrugName)
	if err != nil {
		return nil, err
	}
	treatment.OTC = treatmentType == treatmentOTC

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
	drugDBIDs := make(map[string]string)
	drugRows, err := d.db.Query(fmt.Sprintf(`select drug_db_id_tag, drug_db_id from %s_drug_db_id where %s_id = ? `, tableName, tableName), id)
	if err != nil {
		return err
	}
	defer drugRows.Close()

	for drugRows.Next() {
		var dbIDTag string
		var dbID string
		if err := drugRows.Scan(&dbIDTag, &dbID); err != nil {
			return err
		}
		drugDBIDs[dbIDTag] = dbID
	}

	treatment.DrugDBIDs = drugDBIDs
	return nil
}

func (d *DataService) fillInSupplementalInstructionsForTreatment(treatment *common.Treatment) error {
	// get the supplemental instructions for this treatment
	instructionsRows, err := d.db.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id
													where treatment_instructions.status=? and treatment_id=?`, StatusActive, treatment.ID.Int64())
	if err != nil {
		return err
	}
	defer instructionsRows.Close()

	var drugInstructions []*common.DoctorInstructionItem
	for instructionsRows.Next() {
		var instructionID encoding.ObjectID
		var text string
		if err := instructionsRows.Scan(&instructionID, &text); err != nil {
			return err
		}
		drugInstruction := &common.DoctorInstructionItem{
			ID:       instructionID,
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
		var regimenID, parentID encoding.ObjectID
		err := rows.Scan(&regimenID, &regimenType, &parentID, &regimenText)
		if err != nil {
			return nil, err
		}
		regimenStep := &common.DoctorInstructionItem{
			ID:       regimenID,
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

	var regimenSectionsArray []*common.RegimenSection
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
	var advicePoints []*common.DoctorInstructionItem
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
			&treatmentPlan.PatientViewed, &treatmentPlan.SentDate); err != nil {
			return nil, err
		}
		treatmentPlans = append(treatmentPlans, &treatmentPlan)
	}

	return treatmentPlans, rows.Err()
}
