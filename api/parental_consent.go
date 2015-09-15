package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

// GrantParentChildConsent creates a relationship between the patient accounts and consents to treatment.
// However, this doesn't update the patient because we can't allow the patient to do a visit until
// we've also collected the parent's identification photos. It returns true iff consent had not preivously
// been granted.
func (d *dataService) GrantParentChildConsent(parentPatientID, childPatientID common.PatientID, relationship string) (bool, error) {
	res, err := d.db.Exec(`INSERT IGNORE INTO patient_parent (patient_id, parent_patient_id, relationship, consented) VALUES (?, ?, ?, ?)`,
		childPatientID, parentPatientID, relationship, true)
	if err != nil {
		return false, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	return n != 0, errors.Trace(err)
}

// ParentalConsentCompletedForPatient updates the patient record and visits to reflect consent has been granted
// and all necessary information has been recorded (identification photos). It returns true iff consent had not
// previously been completed for the patient.
func (d *dataService) ParentalConsentCompletedForPatient(childPatientID common.PatientID) (bool, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return false, errors.Trace(err)
	}
	res, err := tx.Exec(`UPDATE patient SET has_parental_consent = ? WHERE id = ? AND has_parental_consent = ?`, true, childPatientID, false)
	if err != nil {
		tx.Rollback()
		return false, errors.Trace(err)
	}
	if n, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return false, errors.Trace(err)
	} else if n == 0 {
		tx.Rollback()
		return false, nil
	}
	_, err = tx.Exec(`UPDATE patient_visit SET status = ? WHERE patient_id = ? AND status = ?`,
		common.PVStatusReceivedParentalConsent, childPatientID, common.PVStatusPendingParentalConsent)
	if err != nil {
		tx.Rollback()
		return false, errors.Trace(err)
	}
	return true, errors.Trace(tx.Commit())
}

// ParentalConsent returns the consent status for a given child
func (d *dataService) ParentalConsent(childPatientID common.PatientID) ([]*common.ParentalConsent, error) {
	var consents []*common.ParentalConsent
	rows, err := d.db.Query(`SELECT parent_patient_id, consented, relationship FROM patient_parent WHERE patient_id = ?`,
		childPatientID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var consent common.ParentalConsent
		if err := rows.Scan(
			&consent.ParentPatientID,
			&consent.Consented,
			&consent.Relationship,
		); err != nil {
			return nil, errors.Trace(err)
		}
		consents = append(consents, &consent)
	}

	return consents, errors.Trace(rows.Err())
}

// AllParentalConsent returns the full set of parent/child consent relationships which
// is a mapping from child's patient ID to the ParentalConsent model.
func (d *dataService) AllParentalConsent(parentPatientID common.PatientID) (map[common.PatientID]*common.ParentalConsent, error) {
	rows, err := d.db.Query(`SELECT patient_id, consented, relationship FROM patient_parent WHERE parent_patient_id = ?`, parentPatientID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	consent := make(map[common.PatientID]*common.ParentalConsent)
	for rows.Next() {
		var childID common.PatientID
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
func (d *dataService) UpsertParentConsentProof(parentPatientID common.PatientID, proof *ParentalConsentProof) (int64, error) {
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
			parentPatientID.Int64(),
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
			parentPatientID.Int64(),
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
func (d *dataService) ParentConsentProof(parentPatientID common.PatientID) (*ParentalConsentProof, error) {
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
