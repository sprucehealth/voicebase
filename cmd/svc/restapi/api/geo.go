package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/errors"
)

func (d *dataService) State(stateName string) (*common.State, error) {
	state := &common.State{}
	err := d.db.QueryRow(
		`SELECT id, full_name, abbreviation, country FROM state 
			WHERE full_name = ? OR abbreviation = ?`,
		strings.Title(stateName), strings.ToUpper(stateName)).Scan(&state.ID, &state.Name, &state.Abbreviation, &state.Country)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(fmt.Sprintf("state not found for full_name or abbreviation %s", stateName)))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return state, nil
}

func (d *dataService) ListStates() ([]*common.State, error) {
	rows, err := d.db.Query(`SELECT id, full_name, abbreviation, country FROM state ORDER BY full_name`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	var states []*common.State
	for rows.Next() {
		state := &common.State{}
		if err := rows.Scan(&state.ID, &state.Name, &state.Abbreviation, &state.Country); err != nil {
			return nil, errors.Trace(err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}
	return states, nil
}
