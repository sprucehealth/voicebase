package api

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) State(state string) (full string, short string, err error) {
	err = d.db.QueryRow(
		`SELECT full_name, abbreviation FROM state WHERE full_name = ? OR abbreviation = ?`,
		strings.Title(state), strings.ToUpper(state)).Scan(&full, &short)
	if err == sql.ErrNoRows {
		return "", "", nil
	} else if err != nil {
		return "", "", err
	}
	return full, short, nil
}

func (d *DataService) ListStates() ([]*common.State, error) {
	rows, err := d.db.Query(`SELECT full_name, abbreviation, country FROM state ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []*common.State
	for rows.Next() {
		state := &common.State{}
		if err := rows.Scan(&state.Name, &state.Abbreviation, &state.Country); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}
