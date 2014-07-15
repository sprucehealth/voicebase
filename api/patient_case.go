package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetDoctorsAssignedToPatientCase(patientCaseId int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, status, creation_date, expires from patient_case_care_provider_assignment where patient_case_id = ? and role_type_id = ?`, patientCaseId, d.roleTypeMapping[DOCTOR_ROLE])
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
func (d *DataService) GetActiveMembersOfCareTeamForCase(patientCaseId int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_case_care_provider_assignment 
		inner join role_type on role_type_id = role_type.id
		where status = ? and patient_case_id = ?`, STATUS_ACTIVE, patientCaseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*common.CareProviderAssignment
	for rows.Next() {
		var assignment common.CareProviderAssignment
		if err := rows.Scan(&assignment.ProviderID, &assignment.ProviderRole, &assignment.Status, &assignment.CreationDate); err != nil {
			return nil, err
		}

		if assignment.ProviderRole == DOCTOR_ROLE {
			doctor, err := d.GetDoctorFromId(assignment.ProviderID)
			if err != nil {
				return nil, err
			}
			assignment.FirstName = doctor.FirstName
			assignment.LastName = doctor.LastName
			assignment.ShortTitle = doctor.ShortTitle
			assignment.LongTitle = doctor.LongTitle
			assignment.SmallThumbnailURL = doctor.SmallThumbnailURL
			assignment.LargeThumbnailURL = doctor.LargeThumbnailURL
			assignment.ProfileURL = doctor.ProfileURL
		}
		assignments = append(assignments, &assignment)
	}
	return assignments, rows.Err()
}

func (d *DataService) AssignDoctorToPatientFileAndCase(doctorId int64, patientCase *common.PatientCase) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`replace into patient_care_provider_assignment (provider_id, role_type_id, patient_id, status, health_condition_id) values (?,?,?,?,?)`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.PatientId.Int64(), STATUS_ACTIVE, patientCase.HealthConditionId.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`replace into patient_case_care_provider_assignment (provider_id, role_type_id, patient_case_id, status) values (?,?,?,?)`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.Id.Int64(), STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPatientCaseFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status, health_condition.medicine_branch from patient_case
							inner join treatment_plan on treatment_plan.patient_case_id = patient_case.id
							inner join health_condition on health_condition.id = health_condition_id
							where treatment_plan.id = ?`, treatmentPlanId)
	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromPatientVisitId(patientVisitId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status, health_condition.medicine_branch from patient_case
							inner join patient_visit on patient_case_id = patient_case.id
							inner join health_condition on health_condition.id = patient_case.health_condition_id
							where patient_visit.id = ?`, patientVisitId)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromId(patientCaseId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_id, health_condition_id, creation_date, status, health_condition.medicine_branch from patient_case
							inner join health_condition on health_condition.id = patient_case.health_condition_id
							where patient_case.id = ?`, patientCaseId)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetCasesForPatient(patientId int64) ([]*common.PatientCase, error) {
	rows, err := d.db.Query(`select patient_case.id, patient_id, health_condition_id, creation_date, status, health_condition.medicine_branch from patient_case 
								inner join health_condition on health_condition.id = patient_case.health_condition_id
								where patient_id=? order by creation_date desc`, patientId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patientCases []*common.PatientCase
	for rows.Next() {
		var patientCase common.PatientCase
		err := rows.Scan(
			&patientCase.Id,
			&patientCase.PatientId,
			&patientCase.HealthConditionId,
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

func (d *DataService) GetActiveTreatmentPlanForCase(patientCaseId int64) (*common.TreatmentPlan, error) {
	rows, err := d.db.Query(`select id, doctor_id, patient_case_id, patient_id, creation_date, status from treatment_plan where patient_case_id = ? and status = ?`, patientCaseId, STATUS_ACTIVE)
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

func (d *DataService) GetVisitsForCase(patientCaseId int64) ([]*common.PatientVisit, error) {
	rows, err := d.db.Query(`select id, patient_id, patient_case_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status, diagnosis from patient_visit 
		where patient_case_id = ? order by creation_date desc`, patientCaseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getPatientVisitFromRows(rows)
}

func getPatientCaseFromRow(row *sql.Row) (*common.PatientCase, error) {
	var patientCase common.PatientCase

	err := row.Scan(
		&patientCase.Id,
		&patientCase.PatientId,
		&patientCase.HealthConditionId,
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

func (d *DataService) DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId int64) error {
	_, err := d.db.Exec(`delete from treatment_plan where doctor_id = ? and status = ? and patient_case_id = ?`, doctorId, STATUS_DRAFT, patientCaseId)
	return err
}

func (d *DataService) GetNotificationsForCase(patientCaseId int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error) {
	rows, err := d.db.Query(`select id, patient_case_id, notification_type, uid, creation_date, data from case_notification where patient_case_id = ? order by creation_date desc`, patientCaseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notificationItems []*common.CaseNotification
	for rows.Next() {
		var notificationItem common.CaseNotification
		var notificationData []byte
		if err := rows.Scan(
			&notificationItem.Id,
			&notificationItem.PatientCaseId,
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

func (d *DataService) GetNotificationCountForCase(patientCaseId int64) (int64, error) {
	var notificationCount int64
	if err := d.db.QueryRow(`select count(*) from case_notification where patient_case_id = ?`, patientCaseId).Scan(&notificationCount); err == sql.ErrNoRows {
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

	_, err = d.db.Exec(`replace into case_notification (patient_case_id, notification_type, uid, data) values (?,?,?,?)`, notificationItem.PatientCaseId, notificationItem.NotificationType, notificationItem.UID, notificationData)
	return err
}

func (d *DataService) DeleteCaseNotification(uid string, patientCaseId int64) error {
	_, err := d.db.Exec(`delete from case_notification where uid = ? and patient_case_id = ?`, uid, patientCaseId)
	return err
}
