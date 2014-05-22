package api

import (
	"bytes"
	"carefront/common"
	"database/sql"
	"encoding/json"
)

func (d *DataService) DoesDrugDetailsExist(ndc string) (bool, error) {
	var id int64
	if err := d.dB.QueryRow(`select id from drug_details where ndc=?`, ndc).Scan(&id); err == sql.ErrNoRows {
		return false, nil
	} else if err == sql.ErrNoRows {
		return false, NoRowsError
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (d *DataService) DrugDetails(ndc string) (*common.DrugDetails, error) {
	var js []byte
	if err := d.dB.QueryRow(`select json from drug_details where ndc = ?`, ndc).Scan(&js); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	details := new(common.DrugDetails)
	if err := json.Unmarshal(js, details); err != nil {
		return nil, err
	}
	return details, nil
}

func (d *DataService) SetDrugDetails(details map[string]*common.DrugDetails) error {
	tx, err := d.dB.Begin()
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	for ndc, det := range details {
		if err := enc.Encode(det); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`replace into drug_details (ndc, json) values (?, ?)`, ndc, buf.Bytes()); err != nil {
			tx.Rollback()
			return err
		}
		buf.Reset()
	}

	return tx.Commit()
}
