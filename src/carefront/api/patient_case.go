package api

import (
	"carefront/common"
	"database/sql"
)

func (d *DataService) GetDoctorsAssignedToPatientCase(patientCaseId int64) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, creation_date from patient_case_care_provider_assignment where patient_case_id = ? and role_type_id = ?`, patientCaseId, d.roleTypeMapping[DOCTOR_ROLE])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*common.CareProviderAssignment
	for rows.Next() {
		var assignment common.CareProviderAssignment
		if err := rows.Scan(&assignment.ProviderId, &assignment.CreationDate); err != nil {
			return nil, err
		}
		assignment.ProviderRole = DOCTOR_ROLE
		assignments = append(assignments, &assignment)
	}
	return assignments, rows.Err()
}

func (d *DataService) GetPatientCaseFromPatientVisitId(patientVisitId int64) (*common.PatientCase, error) {
	var patientCase common.PatientCase
	err := d.db.QueryRow(`select patient_case.id, patient_case.patient_id, patient_case.health_condition_id, patient_case.creation_date, patient_case.status from patient_case
							inner join patient_visit on patient_case_id = patient_case.id
							where patient_visit.id = ?`, patientVisitId).Scan(
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
