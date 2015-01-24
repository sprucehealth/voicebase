package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

// SpruceAvailableInState checks to see if atleast one doctor is registered in the state
// to see patient for any condition.
func (d *DataService) SpruceAvailableInState(state string) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT care_provider_state_elligibility.id 
		FROM care_provider_state_elligibility 
		INNER JOIN care_providing_state ON care_providing_state_id = care_providing_state.id 
		WHERE (state = ? OR long_state = ?) AND role_type_id = ? LIMIT 1`, state, state,
		d.roleTypeMapping[DOCTOR_ROLE]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}

func (d *DataService) GetCareProvidingStateID(stateAbbreviation, pathwayTag string) (int64, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return 0, err
	}
	var careProvidingStateID int64
	if err := d.db.QueryRow(
		`SELECT id FROM care_providing_state WHERE state = ? AND clinical_pathway_id = ?`,
		stateAbbreviation, pathwayID,
	).Scan(&careProvidingStateID); err == sql.ErrNoRows {
		return 0, ErrNotFound("care_providing_state")
	} else if err != nil {
		return 0, err
	}

	return careProvidingStateID, nil
}

func (d *DataService) AddCareProvidingState(stateAbbreviation, fullStateName, pathwayTag string) (int64, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return 0, err
	}

	res, err := d.db.Exec(
		`INSERT INTO care_providing_state (state, long_state, clinical_pathway_id)
		VALUES (?, ?, ?)`,
		stateAbbreviation, fullStateName, pathwayID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error {
	_, err := d.db.Exec(
		`REPLACE INTO care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) VALUES (?,?,?)`,
		d.roleTypeMapping[DOCTOR_ROLE], doctorID, careProvidingStateID)
	return err
}

func (d *DataService) GetDoctorWithEmail(email string) (*common.Doctor, error) {
	var doctorID int64
	if err := d.db.QueryRow(
		`SELECT id FROM doctor WHERE account_id = (SELECT id FROM account WHERE email = ?)`, email,
	).Scan(&doctorID); err == sql.ErrNoRows {
		return nil, ErrNotFound("doctor")
	} else if err != nil {
		return nil, err
	}

	doctor, err := d.GetDoctorFromID(doctorID)
	if err != nil {
		return nil, err
	}

	return doctor, err
}
