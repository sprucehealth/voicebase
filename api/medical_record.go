package api

import (
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
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
		return nil, ErrNotFound("patient_exported_medical_record")
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
	args := dbutil.MySQLVarArgs()
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
		args.Append("status", update.Status.String())
	}
	if update.Error != nil {
		args.Append("error", *update.Error)
	}
	if update.StorageURL != nil {
		args.Append("storage_url", *update.StorageURL)
	}
	if update.Completed != nil {
		args.Append("completed_timestamp", *update.Completed)
	}
	if args.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE patient_exported_medical_record SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id)...)
	return err
}
