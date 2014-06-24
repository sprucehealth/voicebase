package api

import (
	"carefront/common"
	"database/sql"
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
		if err := rows.Scan(&assignment.ProviderId, &assignment.Status, &assignment.CreationDate, &assignment.Expires); err != nil {
			return nil, err
		}
		assignment.ProviderRole = DOCTOR_ROLE
		assignments = append(assignments, &assignment)
	}
	return assignments, rows.Err()
}

func (d *DataService) AssignDoctorToPatientFileAndCase(doctorId, patientId, patientCaseId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`replace into patient_care_provider_assignment (provider_id, role_type_id, patient_id, status) values (?,?,?,?)`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`replace into patient_case_care_provider_assignment (provider_id, role_type_id, patient_case_id, status) values (?,?,?,?)`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCaseId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPatientCaseFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status from patient_case
							inner join treatment_plan on treatment_plan.patient_case_id = patient_case.id
							where treatment_plan.id = ?`, treatmentPlanId)
	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromPatientVisitId(patientVisitId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status from patient_case
							inner join patient_visit on patient_case_id = patient_case.id
							where patient_visit.id = ?`, patientVisitId)

	return getPatientCaseFromRow(row)
}

func (d *DataService) GetPatientCaseFromId(patientCaseId int64) (*common.PatientCase, error) {
	row := d.db.QueryRow(`select id, patient_id, health_condition_id, creation_date, status from patient_case
							where id = ?`, patientCaseId)

	return getPatientCaseFromRow(row)
}

func getPatientCaseFromRow(row *sql.Row) (*common.PatientCase, error) {
	var patientCase common.PatientCase

	err := row.Scan(
		&patientCase.Id,
		&patientCase.PatientId,
		&patientCase.HealthConditionId,
		&patientCase.CreationDate,
		&patientCase.Status)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &patientCase, nil
}

func (d *DataService) GetCareProvidingStateId(stateAbbreviation string, healthConditionId int64) (int64, error) {
	var careProvidingStateId int64
	if err := d.db.QueryRow(`select id from care_providing_state where state = ? and health_condition_id = ?`, stateAbbreviation, healthConditionId).Scan(&careProvidingStateId); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return careProvidingStateId, nil
}

func (d *DataService) DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId int64) error {
	_, err := d.db.Exec(`delete from treatment_plan where doctor_id = ? and status = ? and patient_case_id = ?`, doctorId, STATUS_DRAFT, patientCaseId)
	return err
}
