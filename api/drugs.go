package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) MultiQueryDrugDetailIDs(queries []*DrugDetailsQuery) ([]int64, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	names := make([]interface{}, len(queries))
	for i, q := range queries {
		if q.GenericName != "" {
			q.GenericName = strings.ToLower(q.GenericName)
			names[i] = q.GenericName
		}
	}

	if len(names) == 0 {
		// Return an empty slice of the same length since this is
		// more or less a "successful" query even though no queries
		// include the required generic name.
		return make([]int64, len(queries)), nil
	}

	// determine the ndcs for which drug details exist from the list
	rows, err := d.db.Query(`
		SELECT id, COALESCE(ndc, ''), generic_drug_name, drug_route, COALESCE(drug_form, '')
		FROM drug_details
		WHERE generic_drug_name IN (`+dbutil.MySQLArgs(len(names))+`)`, names...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, len(queries))
	for rows.Next() {
		var id int64
		var ndc, name, route, form string
		if err := rows.Scan(&id, &ndc, &name, &route, &form); err != nil {
			return nil, err
		}
		// See if the drug matches the query based on the following criteria.
		// - generic name and route must match exactly
		// - either the form is not set on the drug, or the form matches the query exactly
		// - either the ndc is not set on the drug, or the NDC matches the query exactly
		for i, q := range queries {
			if (ndc != "" && q.NDC == ndc) ||
				(ndc == "" &&
					q.GenericName == name &&
					q.Route == route &&
					(form == "" || q.Form == form)) {
				ids[i] = id
				break
			}
		}

	}

	return ids, rows.Err()
}

func (d *DataService) DrugDetails(id int64) (*common.DrugDetails, error) {
	row := d.db.QueryRow(`SELECT json FROM drug_details WHERE id = ?`, id)

	var js []byte
	if err := row.Scan(&js); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	var details common.DrugDetails
	if err := json.Unmarshal(js, &details); err != nil {
		return nil, err
	}
	details.ID = id
	return &details, nil
}

func (d *DataService) QueryDrugDetails(query *DrugDetailsQuery) (*common.DrugDetails, error) {
	if query.GenericName == "" || query.Route == "" {
		return nil, NoRowsError
	}
	query.Form = strings.ToLower(query.Form)

	rows, err := d.db.Query(`
		SELECT id, COALESCE(ndc, ''), COALESCE(drug_form, ''), json
		FROM drug_details
		WHERE generic_drug_name = ? AND drug_route = ?`,
		query.GenericName, query.Route)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Pick the most specific details available in the following order:
	// - exact match on NDC
	// - match name + route + form
	// - match name + route
	var haveForm bool
	var bestID int64
	var bestJS []byte
	for rows.Next() {
		var id int64
		var ndc, form string
		var js []byte
		if err := rows.Scan(&id, &ndc, &form, &js); err != nil {
			return nil, err
		}
		if ndc != "" && ndc == query.NDC {
			bestID = id
			bestJS = js
			break
		}
		if !haveForm && ndc == "" && (form == "" || query.Form == form) {
			bestID = id
			bestJS = js
			if form != "" {
				haveForm = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if bestJS == nil {
		return nil, NoRowsError
	}

	var details common.DrugDetails
	if err := json.Unmarshal(bestJS, &details); err != nil {
		return nil, err
	}
	details.ID = bestID
	return &details, nil
}

func (d *DataService) ListDrugDetails() ([]*common.DrugDetails, error) {
	rows, err := d.db.Query(`SELECT id, json FROM drug_details ORDER BY generic_drug_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drugs []*common.DrugDetails
	for rows.Next() {
		var id int64
		var js []byte
		if err := rows.Scan(&id, &js); err != nil {
			return nil, err
		}
		details := &common.DrugDetails{}
		if err := json.Unmarshal(js, &details); err != nil {
			return nil, err
		}
		details.ID = id
		drugs = append(drugs, details)
	}
	return drugs, nil
}

func (d *DataService) SetDrugDetails(details []*common.DrugDetails) error {
	if len(details) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM drug_details`); err != nil {
		tx.Rollback()
		return err
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	for _, det := range details {
		if err := enc.Encode(det); err != nil {
			tx.Rollback()
			return err
		}
		form := sql.NullString{
			String: strings.ToLower(det.Form),
			Valid:  det.Form != "",
		}
		res, err := tx.Exec(`
			INSERT INTO drug_details (ndc, generic_drug_name, drug_route, drug_form, json)
			VALUES (?, ?, ?, ?, ?)`,
			det.NDC, strings.ToLower(det.GenericName), strings.ToLower(det.Route), form,
			buf.Bytes(),
		)
		if err != nil {
			tx.Rollback()
			return err
		}
		det.ID, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
		buf.Reset()
	}

	return tx.Commit()
}
