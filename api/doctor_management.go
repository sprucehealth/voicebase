package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetCareProvidingStateID(stateAbbreviation string, healthConditionID int64) (int64, error) {
	var careProvidingStateID int64
	if err := d.db.QueryRow(`select id from care_providing_state where state = ? and health_condition_id = ?`, stateAbbreviation, healthConditionID).Scan(&careProvidingStateID); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return careProvidingStateID, nil
}

func (d *DataService) AddCareProvidingState(stateAbbreviation, fullStateName string, healthConditionID int64) (int64, error) {
	res, err := d.db.Exec(`insert into care_providing_state (state,long_state, health_condition_id) values (?,?,?)`, stateAbbreviation, fullStateName, healthConditionID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error {
	_, err := d.db.Exec(`REPLACE INTO care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) VALUES (?,?,?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorID, careProvidingStateID)
	return err
}

func (d *DataService) GetDoctorWithEmail(email string) (*common.Doctor, error) {
	var doctorID int64
	if err := d.db.QueryRow(`select id from doctor where account_id = (select id from account where email = ?)`, email).Scan(&doctorID); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	doctor, err := d.GetDoctorFromID(doctorID)
	if err != nil {
		return nil, err
	}

	return doctor, err
}
