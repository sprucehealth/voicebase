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

func (d *DataService) GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, status, creation_date, expires from patient_case_care_provider_assignment where patient_case_id = ? and role_type_id = ?`, patientCaseID, d.roleTypeMapping[DOCTOR_ROLE])
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
		return nil, NoRowsError
	case l == 1:
		return assignments[0], nil
	}

	return nil, errors.New("Expected 1 care provider assignment but got more than 1")

}

func (d *DataService) AssignDoctorToPatientFileAndCase(doctorID int64, patientCase *common.PatientCase) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.assignCareProviderToPatientFileAndCase(tx, doctorID, d.roleTypeMapping[DOCTOR_ROLE], patientCase); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) assignCareProviderToPatientFileAndCase(db db, providerId, roleTypeId int64, patientCase *common.PatientCase) error {
	_, err := db.Exec(`replace into patient_care_provider_assignment (provider_id, role_type_id, patient_id, status, health_condition_id) values (?,?,?,?,?)`, providerId, roleTypeId, patientCase.PatientID.Int64(), STATUS_ACTIVE, patientCase.HealthConditionID.Int64())
	if err != nil {
		return err
	}

	_, err = db.Exec(`replace into patient_case_care_provider_assignment (provider_id, role_type_id, patient_case_id, status) values (?,?,?,?)`, providerId, roleTypeId, patientCase.ID.Int64(), STATUS_ACTIVE)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) GetPatientCaseFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status, health_condition.medicine_branch from patient_case
							inner join treatment_plan on treatment_plan.patient_case_id = patient_case.id
							inner join health_condition on health_condition.id = health_condition_id
							where treatment_plan.id = ?`, treatmentPlanID)
	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status, health_condition.medicine_branch from patient_case
							inner join patient_visit on patient_case_id = patient_case.id
							inner join health_condition on health_condition.id = patient_case.health_condition_id
							where patient_visit.id = ?`, patientVisitID)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_id, health_condition_id, creation_date, status, health_condition.medicine_branch from patient_case
							inner join health_condition on health_condition.id = patient_case.health_condition_id
							where patient_case.id = ?`, patientCaseID)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetCasesForPatient(patientID int64) ([]*common.PatientCase, error) {
	rows, err := d.db.Query(`
		SELECT pc.id, patient_id, health_condition_id, creation_date, status, h.medicine_branch 
		FROM patient_case pc
		INNER JOIN health_condition h ON h.id = pc.health_condition_id
		WHERE patient_id = ? 
		ORDER BY creation_date DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patientCases []*common.PatientCase
	for rows.Next() {
		var patientCase common.PatientCase
		err := rows.Scan(
			&patientCase.ID,
			&patientCase.PatientID,
			&patientCase.HealthConditionID,
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
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status
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
		return nil, NoRowsError
	case l == 1:
		return treatmentPlans[0], nil
	}

	return nil, fmt.Errorf("Expected just one active treatment plan for case instead got %d", len(treatmentPlans))
}

func (d *DataService) GetTreatmentPlansForCase(caseID int64) ([]*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`
		SELECT id, doctor_id, patient_case_id, patient_id, creation_date, status
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
		SELECT id, patient_id, patient_case_id, health_condition_id, layout_version_id,
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
		&patientCase.HealthConditionID,
		&patientCase.CreationDate,
		&patientCase.Status,
		&patientCase.MedicineBranch)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &patientCase, nil
}

func (d *DataService) DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error {
	_, err := d.db.Exec(`
		DELETE FROM treatment_plan
		WHERE doctor_id = ? AND status = ? AND patient_case_id = ?`, doctorID, common.TPStatusDraft.String(), patientCaseID)
	return err
}

func (d *DataService) GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error) {
	rows, err := d.db.Query(`select id, patient_case_id, notification_type, uid, creation_date, data from case_notification where patient_case_id = ? order by creation_date`, patientCaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notificationItems []*common.CaseNotification
	for rows.Next() {
		var notificationItem common.CaseNotification
		var notificationData []byte
		if err := rows.Scan(
			&notificationItem.ID,
			&notificationItem.PatientCaseID,
			&notificationItem.NotificationType,
			&notificationItem.UID,
			&notificationItem.CreationDate,
			&notificationData); err != nil {
			return nil, err
		}

		// based on the notification type, find the appropriate type to render the notification data
		nDataType, ok := notificationTypeRegistry[notificationItem.NotificationType]
		if !ok {
			// currently throwing an error if the notification type is not found as this should not happen right now
			return nil, fmt.Errorf("Unable to find notification type to render data into for item %s", notificationItem.NotificationType)
		}

		notificationItem.Data = reflect.New(nDataType).Interface().(common.Typed)
		if notificationData != nil {
			if err := json.Unmarshal(notificationData, &notificationItem.Data); err != nil {
				return nil, err
			}
		}

		notificationItems = append(notificationItems, &notificationItem)
	}

	return notificationItems, rows.Err()
}

func (d *DataService) GetNotificationCountForCase(patientCaseID int64) (int64, error) {
	var notificationCount int64
	if err := d.db.QueryRow(`select count(*) from case_notification where patient_case_id = ?`, patientCaseID).Scan(&notificationCount); err == sql.ErrNoRows {
		return 0, NoRowsError
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
	res, err := db.Exec(`insert into patient_case (patient_id, health_condition_id, status) values (?,?,?)`, patientCase.PatientID.Int64(),
		patientCase.HealthConditionID.Int64(), patientCase.Status)
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
	if err != NoRowsError && err != nil {
		return err
	} else if err != NoRowsError {
		if err := d.assignCareProviderToPatientFileAndCase(db, ma.DoctorID.Int64(), d.roleTypeMapping[MA_ROLE], patientCase); err != nil {
			return err
		}
	}

	return nil
}
