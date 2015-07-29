package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

// GrantParentChildConsent creates a relationship between the patient accounts and consents to treatment.
// However, this doesn't update the patient because we can't allow the patient to do a visit until
// we've also collected the parent's identification photos.
func (d *DataService) GrantParentChildConsent(parentPatientID, childPatientID int64, relationship string) error {
	_, err := d.db.Exec(`INSERT IGNORE INTO patient_parent (patient_id, parent_patient_id, relationship, consented) VALUES (?, ?, ?, ?)`,
		childPatientID, parentPatientID, relationship, true)
	return errors.Trace(err)
}

// ParentalConsentCompletedForPatient updates the patient record and visits to reflect consent has been granted
// and all necessary information has been recorded (identification photos).
func (d *DataService) ParentalConsentCompletedForPatient(childPatientID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	_, err = tx.Exec(`UPDATE patient SET has_parental_consent = ? WHERE id = ?`, true, childPatientID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	_, err = tx.Exec(`UPDATE patient_visit SET status = ? WHERE patient_id = ? AND status = ?`,
		common.PVStatusReceivedParentalConsent, childPatientID, common.PVStatusPendingParentalConsent)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

// ParentalConsent returns the consent status between parent and child
func (d *DataService) ParentalConsent(parentPatientID, childPatientID int64) (*common.ParentalConsent, error) {
	var consent common.ParentalConsent
	row := d.db.QueryRow(`SELECT consented, relationship FROM patient_parent WHERE patient_id = ? AND parent_patient_id = ?`,
		childPatientID, parentPatientID)
	if err := row.Scan(&consent.Consented, &consent.Relationship); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("patient_parent"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &consent, nil
}

// AllParentalConsent returns the full set of parent/child consent relationships which
// is a mapping from child's patient ID to the ParentalConsent model.
func (d *DataService) AllParentalConsent(parentPatientID int64) (map[int64]*common.ParentalConsent, error) {
	rows, err := d.db.Query(`SELECT patient_id, consented, relationship FROM patient_parent WHERE parent_patient_id = ?`, parentPatientID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	consent := make(map[int64]*common.ParentalConsent)
	for rows.Next() {
		var childID int64
		c := &common.ParentalConsent{}
		if err := rows.Scan(&childID, &c.Consented, &c.Relationship); err != nil {
			return nil, errors.Trace(err)
		}
		consent[childID] = c
	}
	return consent, errors.Trace(rows.Err())
}

// UpsertParentConsentProof performs and INSERT ON DUPLICATE KEY UPDATE. In which it INSERTS a new record if one does not already exist for the provided patient ID and
// 	UPDATES any existing record to match the provided record.
func (d *DataService) UpsertParentConsentProof(parentPatientID int64, proof *ParentalConsentProof) (int64, error) {
	if proof.GovernmentIDPhotoID == nil && proof.SelfiePhotoID == nil {
		return 0, errors.Trace(errors.New("Atleast governmentIDPhotoID or selfiePhotoID must be specified for upsert."))
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	args := dbutil.MySQLVarArgs()
	if proof.SelfiePhotoID != nil {
		args.Append("selfie_media_id", proof.SelfiePhotoID)
		if err := d.claimMedia(
			tx,
			*proof.SelfiePhotoID,
			common.ClaimerTypeParentalConsentProof,
			parentPatientID,
		); err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	}

	if proof.GovernmentIDPhotoID != nil {
		args.Append("governmentid_media_id", proof.GovernmentIDPhotoID)
		if err := d.claimMedia(
			tx,
			*proof.GovernmentIDPhotoID,
			common.ClaimerTypeParentalConsentProof,
			parentPatientID,
		); err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	}

	res, err := tx.Exec(`
			INSERT INTO parent_consent_proof (governmentid_media_id, selfie_media_id, patient_id)
			VALUES (?,?,?)
			ON DUPLICATE KEY UPDATE `+args.Columns(),
		append([]interface{}{proof.GovernmentIDPhotoID, proof.SelfiePhotoID, parentPatientID}, args.Values()...)...)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	return rowsAffected, errors.Trace(tx.Commit())
}

// ParentConsentProof returns the ParentalConsentProof record mapped to the provided patient_id
func (d *DataService) ParentConsentProof(parentPatientID int64) (*ParentalConsentProof, error) {
	var proof ParentalConsentProof
	if err := d.db.QueryRow(`
		SELECT governmentid_media_id, selfie_media_id
		FROM parent_consent_proof
		WHERE patient_id = ?`, parentPatientID).Scan(
		&proof.GovernmentIDPhotoID,
		&proof.SelfiePhotoID); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("parent_consent_proof"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &proof, nil
}

// PatientParentID returns the patient id mapped to the provided patient's parent
func (d *DataService) PatientParentID(childPatientID int64) (int64, error) {
	var parentID int64
	err := d.db.QueryRow(`SELECT parent_patient_id FROM patient_parent WHERE patient_id = ?`, childPatientID).Scan(&parentID)
	if err == sql.ErrNoRows {
		return 0, errors.Trace(ErrNotFound(`patient_parent`))
	}
	return parentID, errors.Trace(err)
}
