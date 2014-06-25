package api

import "database/sql"

func (d *DataService) GetCareProvidingStateId(stateAbbreviation string, healthConditionId int64) (int64, error) {
	var careProvidingStateId int64
	if err := d.db.QueryRow(`select id from care_providing_state where state = ? and health_condition_id = ?`, stateAbbreviation, healthConditionId).Scan(&careProvidingStateId); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return careProvidingStateId, nil
}

func (d *DataService) AddCareProvidingState(stateAbbreviation string, healthConditionId int64) (int64, error) {
	res, err := d.db.Exec(`insert into care_providing_state (state, health_condition_id) values (?,?)`, stateAbbreviation, healthConditionId)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId int64) error {
	_, err := d.db.Exec(`insert into care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) values (?,?,?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, careProvidingStateId)
	return err
}
