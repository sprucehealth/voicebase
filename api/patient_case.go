package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dbutil"
)

// ActiveCaseIDsForPathways returns a mapping of pathwayTag -> caseID of active
// cases for a patient.
func (d *DataService) ActiveCaseIDsForPathways(patientID int64) (map[string]int64, error) {
	rows, err := d.db.Query(`
		SELECT id, clinical_pathway_id
		FROM patient_case
		WHERE status = ? AND patient_id = ?`,
		STATUS_ACTIVE, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mp := make(map[string]int64)
	for rows.Next() {
		var caseID, pathwayID int64
		if err := rows.Scan(&caseID, &pathwayID); err != nil {
			return nil, err
		}
		pathwayTag, err := d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, err
		}
		mp[pathwayTag] = caseID
	}
	return mp, rows.Err()
}

func (d *DataService) GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`
		SELECT provider_id, status, creation_date, expires
		FROM patient_case_care_provider_assignment
		WHERE patient_case_id = ? AND role_type_id = ?`,
		patientCaseID, d.roleTypeMapping[DOCTOR_ROLE])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*common.CareProviderAssignment
	for rows.Next() {
		var assignment common.CareProviderAssignment
		if err := rows.Scan(&assignment.ProviderID, &assignment.Status, &assignment.CreationDate, &assignment.Expires); err != nil {
			return nil, err
		}
		assignment.ProviderRole = DOCTOR_ROLE
		assignments = append(assignments, &assignment)
	}
	return assignments, rows.Err()
}

// GetActiveMembersOfCareTeamForCase returns the care providers that are permanently part of the patient care team
// It also populates the actual provider object so as to make it possible for the client to use this information as is seen fit
func (d *DataService) GetActiveMembersOfCareTeamForCase(patientCaseID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_case_care_provider_assignment 
		inner join role_type on role_type_id = role_type.id
		where status = ? and patient_case_id = ?`, STATUS_ACTIVE, patientCaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getMembersOfCareTeam(rows, fillInDetails)
}

func (d *DataService) GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_case_care_provider_assignment
		inner join role_type on role_type_id = role_type.id
		where status = ? and role_type_tag = ? and patient_case_id = ?`, STATUS_ACTIVE, role, patientCaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments, err := d.getMembersOfCareTeam(rows, false)
	if err != nil {
		return nil, err
	}

	switch l := len(assignments); {
	case l == 0:
		return nil, ErrNotFound("patient_case_care_provider_assignment")
	case l == 1:
		return assignments[0], nil
	}

	return nil, errors.New("Expected 1 care provider assignment but got more than 1")

}

// AddDoctorToPatientCase adds the provided doctor to the care team of the case
// after ensuring that the doctor is registered in the patient's state
// for the pathway pertaining to the patient case.
func (d *DataService) AddDoctorToPatientCase(doctorID, caseID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	patientCase, err := d.GetPatientCaseFromID(caseID)
	if err != nil {
		return err
	}

	// ensure that the care provider is eligible to see patients for the specified pathway in the patient's state
	var patientState string
	if err := d.db.QueryRow(`
		SELECT state
		FROM patient_location
		WHERE patient_id = ?`, patientCase.PatientID.Int64()).Scan(&patientState); err != nil {
		return err
	}

	careProvidingStateID, err := d.GetCareProvidingStateID(patientState, patientCase.PathwayTag)
	if err != nil {
		return err
	}

	var eligibile bool
	if err := d.db.QueryRow(`
		SELECT 1 
		FROM care_provider_state_elligibility
		WHERE role_type_id = ?
		AND provider_id = ?
		AND care_providing_state_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, careProvidingStateID).
		Scan(&eligibile); err == sql.ErrNoRows {
		return fmt.Errorf("care_provider is not registered in %s to see patients for %s", patientState, patientCase.PathwayTag)
	} else if err != nil {
		return err
	}

	if err := d.assignCareProviderToPatientFileAndCase(tx, doctorID, d.roleTypeMapping[DOCTOR_ROLE], patientCase); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) assignCareProviderToPatientFileAndCase(db db, providerID, roleTypeID int64, patientCase *common.PatientCase) error {
	pathwayID, err := d.pathwayIDFromTag(patientCase.PathwayTag)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		REPLACE INTO patient_care_provider_assignment
			(provider_id, role_type_id, patient_id, status, clinical_pathway_id)
		VALUES (?, ?, ?, ?, ?)`,
		providerID, roleTypeID, patientCase.PatientID.Int64(), STATUS_ACTIVE, pathwayID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		REPLACE INTO patient_case_care_provider_assignment
			(provider_id, role_type_id, patient_case_id, status)
		VALUES (?,?,?,?)`,
		providerID, roleTypeID, patientCase.ID.Int64(), STATUS_ACTIVE)
	return err
}

func (d *DataService) CasesForPathway(patientID int64, pathwayTag string, states []string) ([]*common.PatientCase, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return nil, err
	}

	var whereClause string
	if len(states) > 0 {
		whereClause = `pc.status in (` + dbutil.MySQLArgs(len(states)) + `) AND`
	}

	vals := dbutil.AppendStringsToInterfaceSlice(nil, states)
	vals = append(vals, patientID, pathwayID)
	rows, err := d.db.Query(`
		SELECT pc.id, pc.patient_id, cp.tag, pc.name, pc.creation_date, pc.status, cp.medicine_branch
		FROM patient_case pc
		INNER JOIN clinical_pathway cp ON cp.id = clinical_pathway_id
		WHERE `+whereClause+` patient_id = ? AND clinical_pathway_id = ?`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getPatientCaseFromRows(rows)
}

func (d *DataService) GetPatientCaseFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`
		SELECT pc.id, pc.patient_id, cp.tag, pc.name, pc.creation_date, pc.status, cp.medicine_branch
		FROM patient_case pc
		INNER JOIN treatment_plan tp ON tp.patient_case_id = pc.id
		INNER JOIN clinical_pathway cp ON cp.id = clinical_pathway_id
		WHERE tp.id = ?`, treatmentPlanID)
	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`
		SELECT pc.id, pc.patient_id, cp.tag, pc.name, pc.creation_date, pc.status, cp.medicine_branch
		FROM patient_case pc
		INNER JOIN patient_visit pv ON pv.patient_case_id = pc.id
		INNER JOIN clinical_pathway cp ON cp.id = pc.clinical_pathway_id
		WHERE pv.id = ?`, patientVisitID)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`
		SELECT pc.id, pc.patient_id, cp.tag, pc.name, pc.creation_date, pc.status, cp.medicine_branch
		FROM patient_case pc
		INNER JOIN clinical_pathway cp ON cp.id = pc.clinical_pathway_id
		WHERE pc.id = ?`, patientCaseID)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetCasesForPatient(patientID int64) ([]*common.PatientCase, error) {
	rows, err := d.db.Query(`
		SELECT pc.id, pc.patient_id, cp.tag, pc.name, pc.creation_date, pc.status, cp.medicine_branch
		FROM patient_case pc
		INNER JOIN clinical_pathway cp ON cp.id = pc.clinical_pathway_id
		WHERE patient_id = ?
		ORDER BY creation_date DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getPatientCaseFromRows(rows)
}

func (d *DataService) DoesCaseExistForPatient(patientID, patientCaseID int64) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT id
		FROM patient_case
		WHERE patient_id = ? AND id = ?`,
		patientID, patientCaseID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (d *DataService) DoesActiveTreatmentPlanForCaseExist(patientCaseID int64) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT id
		FROM treatment_plan
		WHERE patient_case_id = ? AND status = ?`,
		patientCaseID, common.TPStatusActive.String()).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (d *DataService) GetActiveTreatmentPlanForCase(patientCaseID int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status, patient_viewed
		FROM treatment_plan
		WHERE patient_case_id = ? AND status = ?`, patientCaseID, common.TPStatusActive.String())
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
	case l == 1:
		return treatmentPlans[0], nil
	}

	return nil, fmt.Errorf("Expected just one active treatment plan for case instead got %d", len(treatmentPlans))
}

func (d *DataService) GetTreatmentPlansForCase(caseID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status,patient_viewed
		FROM treatment_plan
		WHERE patient_case_id = ?
			AND (status = ? OR status = ?)`, caseID, common.TPStatusActive.String(), common.TPStatusInactive.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getTreatmentPlansFromRows(rows)
}

func (d *DataService) GetVisitsForCase(patientCaseID int64, statuses []string) ([]*common.PatientVisit, error) {
	vals := []interface{}{patientCaseID}
	var whereClauseStatusFilter string
	if len(statuses) > 0 {
		whereClauseStatusFilter = " AND status in (" + dbutil.MySQLArgs(len(statuses)) + ")"
		vals = dbutil.AppendStringsToInterfaceSlice(vals, statuses)
	}

	rows, err := d.db.Query(`
		SELECT id, patient_id, patient_case_id, clinical_pathway_id, layout_version_id,
		creation_date, submitted_date, closed_date, status, sku_id, followup
		FROM patient_visit
		WHERE patient_case_id = ?`+whereClauseStatusFilter+`
		ORDER BY creation_date DESC`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getPatientVisitFromRows(rows)
}

func getPatientCaseFromRow(row *sql.Row) (*common.PatientCase, error) {
	var patientCase common.PatientCase
	err := row.Scan(
		&patientCase.ID,
		&patientCase.PatientID,
		&patientCase.PathwayTag,
		&patientCase.Name,
		&patientCase.CreationDate,
		&patientCase.Status,
		&patientCase.MedicineBranch)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_case")
	} else if err != nil {
		return nil, err
	}
	return &patientCase, nil
}

func getPatientCaseFromRows(rows *sql.Rows) ([]*common.PatientCase, error) {
	var patientCases []*common.PatientCase
	for rows.Next() {
		var patientCase common.PatientCase
		err := rows.Scan(
			&patientCase.ID,
			&patientCase.PatientID,
			&patientCase.PathwayTag,
			&patientCase.Name,
			&patientCase.CreationDate,
			&patientCase.Status,
			&patientCase.MedicineBranch)
		if err != nil {
			return nil, err
		}
		patientCases = append(patientCases, &patientCase)
	}

	return patientCases, rows.Err()
}

func (d *DataService) DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error {
	_, err := d.db.Exec(`
		DELETE FROM treatment_plan
		WHERE doctor_id = ? AND status = ? AND patient_case_id = ?`, doctorID, common.TPStatusDraft.String(), patientCaseID)
	return err
}

func (d *DataService) GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_case_id, notification_type, uid, creation_date, data 
		FROM case_notification 
		WHERE patient_case_id = ? 
		ORDER BY creation_date`, patientCaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notificationItems []*common.CaseNotification
	for rows.Next() {
		item, err := scanCaseNotification(rows, notificationTypeRegistry)
		if err != nil {
			return nil, err
		}

		notificationItems = append(notificationItems, item)
	}

	return notificationItems, rows.Err()
}

func (d *DataService) NotificationsForCases(
	patientID int64,
	notificationTypeRegistry map[string]reflect.Type) (map[int64][]*common.CaseNotification, error) {

	rows, err := d.db.Query(`
		SELECT cn.id, cn.patient_case_id, cn.notification_type, cn.uid, cn.creation_date, cn.data 
		FROM case_notification cn
		INNER JOIN patient_case ON patient_case_id = patient_case.id
		WHERE patient_id = ? 
		ORDER BY cn.creation_date`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cnMap := make(map[int64][]*common.CaseNotification, 0)
	for rows.Next() {
		item, err := scanCaseNotification(rows, notificationTypeRegistry)
		if err != nil {
			return nil, err
		}

		items := cnMap[item.PatientCaseID]
		cnMap[item.PatientCaseID] = append(items, item)
	}

	return cnMap, rows.Err()
}

func scanCaseNotification(rows *sql.Rows, typeRegistry map[string]reflect.Type) (*common.CaseNotification, error) {
	var item common.CaseNotification
	var notificationData []byte
	if err := rows.Scan(
		&item.ID,
		&item.PatientCaseID,
		&item.NotificationType,
		&item.UID,
		&item.CreationDate,
		&notificationData); err != nil {
		return nil, err
	}

	// based on the notification type, find the appropriate type to render the notification data
	nDataType, ok := typeRegistry[item.NotificationType]
	if !ok {
		// currently throwing an error if the notification type is not found as this should not happen right now
		return nil, fmt.Errorf("Unable to find notification type to render data into for item %s", item.NotificationType)
	}

	item.Data = reflect.New(nDataType).Interface().(common.Typed)
	if notificationData != nil {
		if err := json.Unmarshal(notificationData, &item.Data); err != nil {
			return nil, err
		}
	}

	return &item, nil
}

func (d *DataService) GetNotificationCountForCase(patientCaseID int64) (int64, error) {
	var notificationCount int64
	if err := d.db.QueryRow(`select count(*) from case_notification where patient_case_id = ?`, patientCaseID).Scan(&notificationCount); err == sql.ErrNoRows {
		return 0, ErrNotFound("case_notification")
	} else if err != nil {
		return 0, err
	}
	return notificationCount, nil
}

func (d *DataService) InsertCaseNotification(notificationItem *common.CaseNotification) error {
	notificationData, err := json.Marshal(notificationItem.Data)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`replace into case_notification (patient_case_id, notification_type, uid, data) values (?,?,?,?)`, notificationItem.PatientCaseID, notificationItem.NotificationType, notificationItem.UID, notificationData)
	return err
}

func (d *DataService) DeleteCaseNotification(uid string, patientCaseID int64) error {
	_, err := d.db.Exec(`delete from case_notification where uid = ? and patient_case_id = ?`, uid, patientCaseID)
	return err
}

func (d *DataService) createPatientCase(db db, patientCase *common.PatientCase) error {
	if patientCase.Name == "" {
		pathway, err := d.PathwayForTag(patientCase.PathwayTag, PONone)
		if err != nil {
			return err
		}
		patientCase.Name = pathway.Name
	}

	pathwayID, err := d.pathwayIDFromTag(patientCase.PathwayTag)
	if err != nil {
		return err
	}

	res, err := db.Exec(`
		INSERT INTO patient_case
			(patient_id, name, status, clinical_pathway_id)
		VALUES (?, ?, ?, ?)`,
		patientCase.PatientID.Int64(), patientCase.Name, patientCase.Status.String(), pathwayID)
	if err != nil {
		return err
	}

	patientCaseID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	patientCase.ID = encoding.NewObjectID(patientCaseID)

	// for now, automatically assign MA to be on the care team of the patient and the case
	ma, err := d.GetMAInClinic()
	if err == nil {
		if err := d.assignCareProviderToPatientFileAndCase(db, ma.DoctorID.Int64(), d.roleTypeMapping[MA_ROLE], patientCase); err != nil {
			return err
		}
	} else if !IsErrNotFound(err) {
		return err
	}

	return nil
}
