package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
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

// DoctorIDsInCareProvidingState returns a slice of doctorIDs that are considered available
// and eligible to see patients in the state/pathway combination indicated by careProvidingStateID.
func (d *DataService) DoctorIDsInCareProvidingState(careProvidingStateID int64) ([]int64, error) {
	rows, err := d.db.Query(`
		SELECT provider_id 
		FROM care_provider_state_elligibility
		WHERE unavailable = 0
		AND role_type_id = ?
		AND care_providing_state_id = ?`, d.roleTypeMapping[DOCTOR_ROLE], careProvidingStateID)
	if err != nil {
		return nil, err
	}

	var doctorIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		doctorIDs = append(doctorIDs, id)
	}

	return doctorIDs, rows.Err()
}

// EligibleDoctorIDs returns a slice of doctor IDs (from the provided list) for the doctors that are eligible to see
// patients in the state/pathway combination indicated by the careProvidingStateID.
func (d *DataService) EligibleDoctorIDs(doctorIDs []int64, careProvidingStateID int64) ([]int64, error) {
	if len(doctorIDs) == 0 {
		return nil, nil
	}

	vals := []interface{}{d.roleTypeMapping[DOCTOR_ROLE], careProvidingStateID}
	vals = dbutil.AppendInt64sToInterfaceSlice(vals, doctorIDs)

	rows, err := d.db.Query(`
		SELECT provider_id 
		FROM care_provider_state_elligibility
		WHERE unavailable = 0
			AND role_type_id = ?
			AND care_providing_state_id = ?
			AND provider_id in (`+dbutil.MySQLArgs(len(doctorIDs))+`)`,
		vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	eligibleDoctorIDs := make([]int64, 0, len(doctorIDs))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		eligibleDoctorIDs = append(eligibleDoctorIDs, id)
	}

	return eligibleDoctorIDs, rows.Err()
}

// AvailableDoctorIDs returns a maximum of N available doctor IDs where N is capped at a 100.
func (d *DataService) AvailableDoctorIDs(n int) ([]int64, error) {
	if n == 0 {
		return nil, nil
	} else if n > 100 {
		n = 100
	}

	rows, err := d.db.Query(`
		SELECT provider_id 
		FROM care_provider_state_elligibility
		WHERE unavailable = 0
		AND role_type_id = ?
		LIMIT ?`, d.roleTypeMapping[DOCTOR_ROLE], n)
	if err != nil {
		return nil, err
	}

	var doctorIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		doctorIDs = append(doctorIDs, id)
	}

	return doctorIDs, rows.Err()
}
