package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/patient_case/model"
)

func (d *dataService) CaseIDForTreatmentPlan(treatmentPlanID int64) (int64, error) {
	row := d.db.QueryRow(`SELECT patient_case_id FROM treatment_plan WHERE id = ?`, treatmentPlanID)
	var caseID int64
	if err := row.Scan(&caseID); err == sql.ErrNoRows {
		return 0, errors.Trace(ErrNotFound("treatment_plan"))
	} else if err != nil {
		return 0, errors.Trace(err)
	}
	return caseID, nil
}

func (d *dataService) GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`
		SELECT provider_id, status, creation_date, expires
		FROM patient_case_care_provider_assignment
		WHERE patient_case_id = ? AND role_type_id = ?`,
		patientCaseID, d.roleTypeMapping[RoleDoctor])
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
		assignment.ProviderRole = RoleDoctor
		assignments = append(assignments, &assignment)
	}
	return assignments, rows.Err()
}

func (d *dataService) TimedOutCases() ([]*common.PatientCase, error) {
	rows, err := d.db.Query(`
		SELECT pc.id, pc.patient_id, pc.clinical_pathway_id, pc.name, pc.creation_date, pc.closed_date, pc.timeout_date, pc.status, pc.claimed
		FROM patient_case pc
		WHERE timeout_date IS NOT null AND timeout_date < ?`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patientCases []*common.PatientCase
	for rows.Next() {
		pc, err := d.getPatientCaseFromRow(rows)
		if err != nil {
			return nil, err
		}

		patientCases = append(patientCases, pc)
	}

	return patientCases, rows.Err()
}

// GetActiveMembersOfCareTeamForCase returns the care providers that are permanently part of the patient care team
// It also populates the actual provider object so as to make it possible for the client to use this information as is seen fit
func (d *dataService) GetActiveMembersOfCareTeamForCase(patientCaseID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`
		SELECT provider_id, role_type_tag, status, creation_date
		FROM patient_case_care_provider_assignment
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE status = ? AND patient_case_id = ?`, StatusActive, patientCaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getMembersOfCareTeam(rows, fillInDetails)
}

func (d *dataService) GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`
		SELECT provider_id, role_type_tag, status, creation_date
		FROM patient_case_care_provider_assignment
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE status = ? AND role_type_tag = ? AND patient_case_id = ?`,
		StatusActive, role, patientCaseID)
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
func (d *dataService) AddDoctorToPatientCase(doctorID, caseID int64) error {
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
		AND care_providing_state_id = ?`, d.roleTypeMapping[RoleDoctor], doctorID, careProvidingStateID).
		Scan(&eligibile); err == sql.ErrNoRows {
		return fmt.Errorf("care_provider is not registered in %s to see patients for %s", patientState, patientCase.PathwayTag)
	} else if err != nil {
		return err
	}

	if err := d.assignCareProviderToPatientFileAndCase(tx, doctorID, d.roleTypeMapping[RoleDoctor], patientCase); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *dataService) assignCareProviderToPatientFileAndCase(tx *sql.Tx, providerID, roleTypeID int64, patientCase *common.PatientCase) error {
	pathwayID, err := d.pathwayIDFromTag(patientCase.PathwayTag)
	if err != nil {
		return err
	}

	// update the case to indicate that its claimed
	// only if a doctor is assigned to the case
	if roleTypeID == d.roleTypeMapping[RoleDoctor] {
		_, err = tx.Exec(`
		UPDATE patient_case
		SET claimed = 1
		WHERE id = ?`, patientCase.ID.Int64())
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		REPLACE INTO patient_care_provider_assignment
			(provider_id, role_type_id, patient_id, status, clinical_pathway_id)
		VALUES (?, ?, ?, ?, ?)`,
		providerID, roleTypeID, patientCase.PatientID.Int64(), StatusActive, pathwayID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		REPLACE INTO patient_case_care_provider_assignment
			(provider_id, role_type_id, patient_case_id, status)
		VALUES (?,?,?,?)`,
		providerID, roleTypeID, patientCase.ID.Int64(), StatusActive)
	return err
}

func (d *dataService) CasesForPathway(patientID common.PatientID, pathwayTag string, states []string) ([]*common.PatientCase, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var whereClause string
	if len(states) > 0 {
		whereClause = `pc.status in (` + dbutil.MySQLArgs(len(states)) + `) AND`
	}

	vals := dbutil.AppendStringsToInterfaceSlice(nil, states)
	vals = append(vals, patientID, pathwayID)
	rows, err := d.db.Query(`
		SELECT pc.id, pc.patient_id, pc.clinical_pathway_id, pc.name, pc.creation_date, pc.closed_date, pc.timeout_date, pc.status, pc.claimed
		FROM patient_case pc
		WHERE `+whereClause+` patient_id = ? AND clinical_pathway_id = ?`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var patientCases []*common.PatientCase
	for rows.Next() {
		pc, err := d.getPatientCaseFromRow(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}

		patientCases = append(patientCases, pc)
	}

	return patientCases, errors.Trace(rows.Err())
}

func (d *dataService) GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`
		SELECT pc.id, pc.patient_id, pc.clinical_pathway_id, pc.name, pc.creation_date, pc.closed_date, pc.timeout_date, pc.status, pc.claimed
		FROM patient_case pc
		INNER JOIN patient_visit pv ON pv.patient_case_id = pc.id
		WHERE pv.id = ?`, patientVisitID)

	return d.getPatientCaseFromRow(row)
}

func (d *dataService) GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`
		SELECT pc.id, pc.patient_id, pc.clinical_pathway_id, pc.name, pc.creation_date, pc.closed_date, pc.timeout_date, pc.status, pc.claimed
		FROM patient_case pc
		WHERE pc.id = ?`, patientCaseID)

	return d.getPatientCaseFromRow(row)
}

func (d *dataService) GetCasesForPatient(patientID common.PatientID, states []string) ([]*common.PatientCase, error) {
	vals := []interface{}{patientID}
	var whereClause string
	if len(states) > 0 {
		whereClause = "AND pc.status IN (" + dbutil.MySQLArgs(len(states)) + ")"
		vals = dbutil.AppendStringsToInterfaceSlice(vals, states)
	} else {
		// filter out any deleted case by default
		whereClause = "AND pc.status NOT IN (" + dbutil.MySQLArgs(len(common.DeletedPatientCaseStates())) + ")"
		vals = dbutil.AppendStringsToInterfaceSlice(vals, common.DeletedPatientCaseStates())
	}
	rows, err := d.db.Query(`
		SELECT pc.id, pc.patient_id, pc.clinical_pathway_id, pc.name, pc.creation_date, pc.closed_date, pc.timeout_date, pc.status, pc.claimed
		FROM patient_case pc
		WHERE patient_id = ? `+whereClause+`
		ORDER BY creation_date DESC`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var patientCases []*common.PatientCase
	for rows.Next() {
		pc, err := d.getPatientCaseFromRow(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}

		patientCases = append(patientCases, pc)
	}

	return patientCases, errors.Trace(rows.Err())
}

// Utility function for populating assignment refernces with their provider's data
func (d *dataService) populateAssignmentInfoFromProviderID(assignment *common.CareProviderAssignment, providerID int64) error {
	doctor, err := d.Doctor(assignment.ProviderID, true)
	if err != nil {
		return err
	}
	assignment.FirstName = doctor.FirstName
	assignment.LastName = doctor.LastName
	assignment.ShortTitle = doctor.ShortTitle
	assignment.LongTitle = doctor.LongTitle
	assignment.ShortDisplayName = doctor.ShortDisplayName
	assignment.LongDisplayName = doctor.LongDisplayName
	assignment.SmallThumbnailID = doctor.SmallThumbnailID
	assignment.LargeThumbnailID = doctor.LargeThumbnailID
	return nil
}

// CaseCareTeams returns care teams for a given set of cases.
func (d *dataService) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	if len(caseIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
			SELECT role_type_tag, pccpa.creation_date, expires, provider_id, pccpa.status, patient_case_id
			FROM patient_case_care_provider_assignment AS pccpa
			INNER JOIN role_type ON role_type.id = role_type_id
			WHERE patient_case_id IN (`+dbutil.MySQLArgs(len(caseIDs))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, caseIDs)...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patientCaseID int64
	careTeams := make(map[int64]*common.PatientCareTeam)
	for rows.Next() {
		var assignment common.CareProviderAssignment
		err := rows.Scan(&assignment.ProviderRole,
			&assignment.CreationDate,
			&assignment.Expires,
			&assignment.ProviderID,
			&assignment.Status,
			&patientCaseID)
		if err != nil {
			return nil, err
		}

		if err := d.populateAssignmentInfoFromProviderID(&assignment, assignment.ProviderID); err != nil {
			return nil, err
		}

		if _, ok := careTeams[patientCaseID]; !ok {
			careTeams[patientCaseID] = &common.PatientCareTeam{}
		}

		careTeam := careTeams[patientCaseID]
		careTeam.Assignments = append(careTeam.Assignments, &assignment)
	}

	return careTeams, rows.Err()
}

func (d *dataService) DoesCaseExistForPatient(patientID common.PatientID, patientCaseID int64) (bool, error) {
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

func (d *dataService) DoesActiveTreatmentPlanForCaseExist(patientCaseID int64) (bool, error) {
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

func (d *dataService) GetActiveTreatmentPlanForCase(patientCaseID int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status, patient_viewed, sent_date
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

func (d *dataService) GetTreatmentPlansForCase(caseID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status,patient_viewed, sent_date
		FROM treatment_plan
		WHERE patient_case_id = ?
			AND (status = ? OR status = ?)`, caseID, common.TPStatusActive.String(), common.TPStatusInactive.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getTreatmentPlansFromRows(rows)
}

func (d *dataService) GetVisitsForCase(patientCaseID int64, statuses []string) ([]*common.PatientVisit, error) {
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
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	return d.getPatientVisitFromRows(rows)
}

func (d *dataService) getPatientCaseFromRow(s scannable) (*common.PatientCase, error) {
	var patientCase common.PatientCase
	var pathwayID int64
	err := s.Scan(
		&patientCase.ID,
		&patientCase.PatientID,
		&pathwayID,
		&patientCase.Name,
		&patientCase.CreationDate,
		&patientCase.ClosedDate,
		&patientCase.TimeoutDate,
		&patientCase.Status,
		&patientCase.Claimed)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_case")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	patientCase.PathwayTag, err = d.pathwayTagFromID(pathwayID)
	if err != nil {
		return nil, err
	}

	return &patientCase, nil
}

func (d *dataService) DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error {
	_, err := d.db.Exec(`
		DELETE FROM treatment_plan
		WHERE doctor_id = ? AND status = ? AND patient_case_id = ?`, doctorID, common.TPStatusDraft.String(), patientCaseID)
	return err
}

func (d *dataService) GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error) {
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

func (d *dataService) NotificationsForCases(
	patientID common.PatientID,
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

func (d *dataService) GetNotificationCountForCase(patientCaseID int64) (int64, error) {
	var notificationCount int64
	if err := d.db.QueryRow(`select count(*) from case_notification where patient_case_id = ?`, patientCaseID).Scan(&notificationCount); err == sql.ErrNoRows {
		return 0, ErrNotFound("case_notification")
	} else if err != nil {
		return 0, err
	}
	return notificationCount, nil
}

func (d *dataService) InsertCaseNotification(notificationItem *common.CaseNotification) error {
	notificationData, err := json.Marshal(notificationItem.Data)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`replace into case_notification (patient_case_id, notification_type, uid, data) values (?,?,?,?)`,
		notificationItem.PatientCaseID, notificationItem.NotificationType, notificationItem.UID, notificationData)
	return err
}

func (d *dataService) DeleteCaseNotification(uid string, patientCaseID int64) error {
	_, err := d.db.Exec(`delete from case_notification where uid = ? and patient_case_id = ?`, uid, patientCaseID)
	return err
}

func (d *dataService) createPatientCase(tx *sql.Tx, patientCase *common.PatientCase) error {
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

	res, err := tx.Exec(`
		INSERT INTO patient_case
			(patient_id, name, status, clinical_pathway_id, requested_doctor_id)
		VALUES (?, ?, ?, ?, ?)`,
		patientCase.PatientID.Int64(), patientCase.Name, patientCase.Status.String(), pathwayID, patientCase.RequestedDoctorID)
	if err != nil {
		return err
	}

	patientCaseID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	patientCase.ID = encoding.DeprecatedNewObjectID(patientCaseID)

	// Assign a random primary CC to the case care team
	cc, err := d.ListCareProviders(LCPOptPrimaryCCOnly)
	if err != nil {
		return err
	}
	if len(cc) == 0 {
		return nil
	}
	return d.assignCareProviderToPatientFileAndCase(tx, cc[rand.Intn(len(cc))].ID.Int64(), d.roleTypeMapping[RoleCC], patientCase)
}

func (d *dataService) UpdatePatientCase(id int64, update *PatientCaseUpdate) error {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.ClosedDate != nil {
		args.Append("closed_date", *update.ClosedDate)
	}
	if update.TimeoutDate.Valid {
		args.Append("timeout_date", update.TimeoutDate.Time)
	}
	if args.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE patient_case set `+args.Columns()+` WHERE id = ?`, append(args.Values(), id)...)
	return err
}

// InsertPatientCaseNote inserts the provided case note record
func (d *dataService) InsertPatientCaseNote(n *model.PatientCaseNote) (int64, error) {
	res, err := d.db.Exec(`INSERT INTO patient_case_note (case_id, author_doctor_id, note_text) VALUES (?, ?, ?)`,
		n.CaseID, n.AuthorDoctorID, n.NoteText)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.LastInsertId()
}

// UpdatePatientCaseNote updates the record for the indicated note
func (d *dataService) UpdatePatientCaseNote(nu *model.PatientCaseNoteUpdate) (int64, error) {
	res, err := d.db.Exec(`UPDATE patient_case_note SET note_text = ? WHERE id = ?`, nu.NoteText, nu.ID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

// DeletePatientCaseNote deletes the record for the indicated note
func (d *dataService) DeletePatientCaseNote(id int64) (int64, error) {
	res, err := d.db.Exec(`DELETE FROM patient_case_note WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

// PatientCaseNote retrieves the record for the indicated note
func (d *dataService) PatientCaseNote(id int64) (*model.PatientCaseNote, error) {
	caseNote := &model.PatientCaseNote{}
	if err := d.db.QueryRow(
		`SELECT id, case_id, created, modified, author_doctor_id, note_text
			FROM patient_case_note WHERE id = ?`, id).Scan(&caseNote.ID, &caseNote.CaseID, &caseNote.Created, &caseNote.Modified, &caseNote.AuthorDoctorID, &caseNote.NoteText); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(`patient_case_note`))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return caseNote, nil
}

// PatientCaseNotes returns a map from case id to list of notes for the given case
func (d *dataService) PatientCaseNotes(caseIDs []int64) (map[int64][]*model.PatientCaseNote, error) {
	if len(caseIDs) == 0 {
		return nil, nil
	}
	rows, err := d.db.Query(`SELECT id, case_id, created, modified, author_doctor_id, note_text FROM patient_case_note WHERE case_id IN (`+dbutil.MySQLArgs(len(caseIDs))+`) ORDER BY created ASC, id ASC`, dbutil.AppendInt64sToInterfaceSlice(nil, caseIDs)...)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(`patient_case_note`))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	caseNotes := make(map[int64][]*model.PatientCaseNote, len(caseIDs))
	for rows.Next() {
		caseNote := &model.PatientCaseNote{}
		if err := rows.Scan(&caseNote.ID, &caseNote.CaseID, &caseNote.Created, &caseNote.Modified, &caseNote.AuthorDoctorID, &caseNote.NoteText); err != nil {
			return nil, errors.Trace(err)
		}
		caseNotes[caseNote.CaseID] = append(caseNotes[caseNote.CaseID], caseNote)
	}

	return caseNotes, errors.Trace(rows.Err())
}
