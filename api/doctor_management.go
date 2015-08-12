package api

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) AvailableStates() ([]*common.State, error) {
	rows, err := d.db.Query(`
		SELECT DISTINCT state, long_state
		FROM care_provider_state_elligibility cpse
		INNER JOIN care_providing_state cps ON cps.id = cpse.care_providing_state_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []*common.State
	for rows.Next() {
		s := &common.State{}
		if err := rows.Scan(&s.Abbreviation, &s.Name); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

// SpruceAvailableInState checks to see if atleast one doctor is registered in the state
// to see patient for any condition.
func (d *dataService) SpruceAvailableInState(state string) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT care_provider_state_elligibility.id
		FROM care_provider_state_elligibility
		INNER JOIN care_providing_state ON care_providing_state_id = care_providing_state.id
		WHERE (state = ? OR long_state = ?) AND role_type_id = ? LIMIT 1`, state, state,
		d.roleTypeMapping[RoleDoctor]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}

func (d *dataService) GetCareProvidingStateID(stateAbbreviation, pathwayTag string) (int64, error) {
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

func (d *dataService) AddCareProvidingState(stateAbbreviation, fullStateName, pathwayTag string) (int64, error) {
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

func (d *dataService) MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error {
	_, err := d.db.Exec(
		`REPLACE INTO care_provider_state_elligibility (role_type_id, provider_id, care_providing_state_id) VALUES (?,?,?)`,
		d.roleTypeMapping[RoleDoctor], doctorID, careProvidingStateID)
	return err
}

func (d *dataService) CareProviderEligible(careProviderID int64, role, state, pathwayTag string) (bool, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return false, errors.Trace(err)
	}

	var id int64
	err = d.db.QueryRow(`
		SELECT cpse.id 
		FROM care_provider_state_elligibility cpse
		INNER JOIN care_providing_state cps ON cps.id = cpse.care_providing_state_id
		WHERE cps.state = ?
			AND cpse.provider_id = ?
			AND cpse.role_type_id = ?
			AND cps.clinical_pathway_id = ?`,
		state, careProviderID, d.roleTypeMapping[role], pathwayID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return (id > 0), errors.Trace(err)
}

func (d *dataService) GetDoctorWithEmail(email string) (*common.Doctor, error) {
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
func (d *dataService) DoctorIDsInCareProvidingState(careProvidingStateID int64) ([]int64, error) {
	rows, err := d.db.Query(`
		SELECT provider_id
		FROM care_provider_state_elligibility
		WHERE unavailable = 0
		AND role_type_id = ?
		AND care_providing_state_id = ?`, d.roleTypeMapping[RoleDoctor], careProvidingStateID)
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
func (d *dataService) EligibleDoctorIDs(doctorIDs []int64, careProvidingStateID int64) ([]int64, error) {
	if len(doctorIDs) == 0 {
		return nil, nil
	}

	vals := []interface{}{d.roleTypeMapping[RoleDoctor], careProvidingStateID}
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
func (d *dataService) AvailableDoctorIDs(n int) ([]int64, error) {
	if n == 0 {
		return nil, nil
	} else if n > 100 {
		n = 100
	}

	rows, err := d.db.Query(`
		SELECT DISTINCT provider_id
		FROM care_provider_state_elligibility
		WHERE unavailable = 0
		AND role_type_id = ?
		LIMIT ?`, d.roleTypeMapping[RoleDoctor], n)
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

func (d *dataService) CareProviderStatePathwayMappings(query *CareProviderStatePathwayMappingQuery) ([]*CareProviderStatePathway, error) {
	var where []string
	var vals []interface{}
	if query != nil {
		if query.State != "" {
			where = append(where, "cps.state = ?")
			vals = append(vals, query.State)
		}
		if query.PathwayTag != "" {
			pathwayID, err := d.pathwayIDFromTag(query.PathwayTag)
			if err != nil {
				return nil, err
			}
			where = append(where, "cps.clinical_pathway_id = ?")
			vals = append(vals, pathwayID)
		}
		if query.Provider.Role != "" && query.Provider.ID != 0 {
			where = append(where, "cpse.role_type_id = ?", "cpse.provider_id = ?")
			vals = append(vals, d.roleTypeMapping[query.Provider.Role], query.Provider.ID)
		}
	}
	var whereClause string
	if len(where) != 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}
	rows, err := d.db.Query(`
		SELECT cpse.id, cpse.role_type_id, cpse.provider_id, cpse.notify, cpse.unavailable,
			cps.state, cps.clinical_pathway_id, d.short_display_name, COALESCE(d.large_thumbnail_id, '')
		FROM care_provider_state_elligibility cpse
		INNER JOIN care_providing_state cps ON cps.id = cpse.care_providing_state_id
		INNER JOIN doctor d ON d.id = provider_id
		`+whereClause+`
		LIMIT 1000`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*CareProviderStatePathway
	for rows.Next() {
		var roleID int64
		p := &CareProviderStatePathway{}
		var pathwayID int64
		if err := rows.Scan(
			&p.ID, &roleID, &p.Provider.ID, &p.Notify, &p.Unavailable,
			&p.StateCode, &pathwayID, &p.ShortDisplayName, &p.ThumbnailID,
		); err != nil {
			return nil, err
		}
		p.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, err
		}
		p.Provider.Role = d.roleIDMapping[roleID]
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (d *dataService) CareProviderStatePathwayMappingSummary() ([]*CareProviderStatePathwayMappingSummary, error) {
	rows, err := d.db.Query(`
		SELECT state, clinical_pathway_id,
			(SELECT COUNT(1)
			 FROM care_provider_state_elligibility cpse
			 WHERE cpse.care_providing_state_id = cps.id) AS doctor_count
		FROM care_providing_state cps`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary []*CareProviderStatePathwayMappingSummary
	for rows.Next() {
		s := &CareProviderStatePathwayMappingSummary{}
		var pathwayID int64
		if err := rows.Scan(&s.StateCode, &pathwayID, &s.DoctorCount); err != nil {
			return nil, err
		}
		s.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, err
		}
		summary = append(summary, s)
	}
	return summary, rows.Err()
}

func (d *dataService) UpdateCareProviderStatePathwayMapping(patch *CareProviderStatePathwayMappingPatch) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.careProviderStatePathwayMappingUpdate(tx, patch.Update); err != nil {
		tx.Rollback()
		return err
	}

	if len(patch.Delete) > 0 {
		_, err = tx.Exec(`
			DELETE FROM care_provider_state_elligibility
			WHERE id IN (`+dbutil.MySQLArgs(len(patch.Delete))+`)`,
			dbutil.AppendInt64sToInterfaceSlice(nil, patch.Delete)...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(patch.Create) > 0 {
		// TODO: this is inefficient but shouldn't matter since this whole function
		// is only used in admin and should have a relatively short list of creates
		for _, c := range patch.Create {
			spID, err := d.GetCareProvidingStateID(c.StateCode, c.PathwayTag)
			if IsErrNotFound(err) {
				name, code, err := d.State(c.StateCode)
				if err != nil {
					tx.Rollback()
					return err
				}
				spID, err = d.AddCareProvidingState(code, name, c.PathwayTag)
				if err != nil {
					tx.Rollback()
					return err
				}
			} else if err != nil {
				tx.Rollback()
				return err
			}
			_, err = tx.Exec(`
				INSERT INTO care_provider_state_elligibility
					(role_type_id, provider_id, care_providing_state_id, notify, unavailable)
				VALUES (?, ?, ?, ?, ?)`,
				d.roleTypeMapping[c.Provider.Role], c.Provider.ID, spID,
				c.Notify, c.Unavailable)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *dataService) careProviderStatePathwayMappingUpdate(tx *sql.Tx, updates []*CareProviderStatePathwayMappingUpdate) error {
	var cols []string
	var vals []interface{}
	for _, u := range updates {
		cols = cols[:0]
		vals = vals[:0]
		if u.Notify != nil {
			cols = append(cols, "notify = ?")
			vals = append(vals, *u.Notify)
		}
		if u.Unavailable != nil {
			cols = append(cols, "unavailable = ?")
			vals = append(vals, *u.Unavailable)
		}
		if len(vals) == 0 {
			continue
		}
		vals = append(vals, u.ID)
		_, err := tx.Exec(`
			UPDATE care_provider_state_elligibility
			SET `+strings.Join(cols, ", ")+`
			WHERE id = ?`, vals...)
		if err != nil {
			return err
		}
	}
	return nil
}
