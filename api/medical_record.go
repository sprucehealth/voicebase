package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) MedicalRecordsForPatient(patientID int64) ([]*common.MedicalRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, status, error, storage_url, requested_timestamp, completed_timestamp
		FROM patient_exported_medical_record
		WHERE patient_id = ?
		ORDER BY completed_timestamp DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*common.MedicalRecord
	for rows.Next() {
		var errMsg sql.NullString
		var storageURL sql.NullString
		r := &common.MedicalRecord{
			PatientID: patientID,
		}
		if err := rows.Scan(&r.ID, &r.Status, &errMsg, &storageURL, &r.Requested, &r.Completed); err != nil {
			return nil, err
		}
		r.Error = errMsg.String
		r.StorageURL = storageURL.String
		records = append(records, r)
	}

	return records, rows.Err()
}

func (d *DataService) MedicalRecord(id int64) (*common.MedicalRecord, error) {
	row := d.db.QueryRow(`
		SELECT patient_id, status, error, storage_url, requested_timestamp, completed_timestamp
		FROM patient_exported_medical_record
		WHERE id = ?`, id)

	r := &common.MedicalRecord{
		ID: id,
	}
	var errMsg sql.NullString
	var storageURL sql.NullString
	if err := row.Scan(&r.PatientID, &r.Status, &errMsg, &storageURL, &r.Requested, &r.Completed); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	r.Error = errMsg.String
	r.StorageURL = storageURL.String

	return r, nil
}

func (d *DataService) CreateMedicalRecord(patientID int64) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO patient_exported_medical_record (patient_id, status)
		VALUES (?, ?)`, patientID, common.MRPending.String())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) UpdateMedicalRecord(id int64, update *MedicalRecordUpdate) error {
	var cols []string
	var vals []interface{}

	if update.Status != nil {
		switch *update.Status {
		case common.MRError:
			if update.Error == nil {
				return fmt.Errorf("setting medical record status to error must also set an error message")
			}
		case common.MRSuccess:
			if update.StorageURL == nil {
				return fmt.Errorf("setting medical record status to success must also include a storage URL")
			}
		}

		cols = append(cols, "status = ?")
		vals = append(vals, update.Status.String())
	}
	if update.Error != nil {
		cols = append(cols, "error = ?")
		vals = append(vals, *update.Error)
	}
	if update.StorageURL != nil {
		cols = append(cols, "storage_url = ?")
		vals = append(vals, *update.StorageURL)
	}
	if update.Completed != nil {
		cols = append(cols, "completed_timestamp = ?")
		vals = append(vals, *update.Completed)
	}

	if len(cols) == 0 {
		return nil
	}
	vals = append(vals, id)

	colStr := strings.Join(cols, ", ")
	_, err := d.db.Exec(`UPDATE patient_exported_medical_record SET `+colStr+` WHERE id = ?`, vals...)
	return err
}
