package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetCareProvidingStateId(stateAbbreviation string, healthConditionId int64) (int64, error) {
	var careProvidingStateId int64
	if err := d.db.QueryRow(`select id from care_providing_state where state = ? and health_condition_id = ?`, stateAbbreviation, healthConditionId).Scan(&careProvidingStateId); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return careProvidingStateId, nil
}

func (d *DataService) AddCareProvidingState(stateAbbreviation, fullStateName string, healthConditionId int64) (int64, error) {
	res, err := d.db.Exec(`insert into care_providing_state (state,long_state, health_condition_id) values (?,?,?)`, stateAbbreviation, fullStateName, healthConditionId)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId int64) error {
	_, err := d.db.Exec(`REPLACE INTO care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) VALUES (?,?,?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, careProvidingStateId)
	return err
}

func (d *DataService) GetDoctorWithEmail(email string) (*common.Doctor, error) {
	var doctorId int64
	if err := d.db.QueryRow(`select id from doctor where account_id = (select id from account where email = ?)`, email).Scan(&doctorId); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	doctor, err := d.GetDoctorFromId(doctorId)
	if err != nil {
		return nil, err
	}

	return doctor, err
}

func (d *DataService) ProvidersToNotifyOfVisitInCareProvidingState(careProvidingStateID int64) ([]*Provider, error) {
	rows, err := d.db.Query(`
		SELECT provider_id, role_type_tag from care_provider_state_elligibility 
		INNER JOIN role_type ON role_type.id = role_type_id
		WHERE care_providing_state_id = ?
		AND notify = 1`, careProvidingStateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		var provider Provider
		if err := rows.Scan(&provider.ProviderID, &provider.ProviderRole); err != nil {
			return nil, err
		}
		providers = append(providers, &provider)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return providers, nil
}
