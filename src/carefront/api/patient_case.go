package api

import (
	"database/sql"

	"carefront/common"
)

func (d *DataService) GetPatientCase(caseID int64) (*common.PatientCase, error) {
	cas := &common.PatientCase{
		Id: caseID,
	}
	row := d.db.QueryRow(`
		SELECT patient_id, health_condition_id, status, creation_date
		FROM patient_case WHERE id = ?`, caseID)
	if err := row.Scan(&cas.PatientId, &cas.HealthConditionId, &cas.Status, &cas.CreationDate); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return cas, nil
}
