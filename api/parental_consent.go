package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

// LinkParentChild creates a relationship between the patient accounts but does not grant consent to be treated
func (d *DataService) LinkParentChild(parentPatientID, childPatientID int64, relationship string) error {
	_, err := d.db.Exec(`INSERT IGNORE INTO patient_parent (patient_id, parent_patient_id, relationship) VALUES (?, ?, ?)`,
		childPatientID, parentPatientID, relationship)
	return errors.Trace(err)
}

// GrantParentChildConsent records that the parent consented to their child being treated
func (d *DataService) GrantParentChildConsent(parentPatientID, childPatientID int64) error {
	// Make sure relationship exists
	_, err := d.ParentChildConsent(parentPatientID, childPatientID)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = tx.Exec(`UPDATE patient_parent SET consented = ? WHERE patient_id = ? AND parent_patient_id = ?`,
		true, childPatientID, parentPatientID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	_, err = tx.Exec(`UPDATE patient SET has_parental_consent = ? WHERE id = ?`, true, childPatientID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	// Update any visits pending consent
	// TODO: This doesn't feel appropriate here but it's the only way to get it into the transaction.
	//       Once we have a better non-transactional story (background repair) this seems safest.
	_, err = tx.Exec(`UPDATE patient_visit SET status = ? WHERE patient_id = ? AND status = ?`,
		common.PVStatusReceivedParentalConsent, childPatientID, common.PVStatusPendingParentalConsent)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

// ParentChildConsent returns the consent status between parent and child
func (d *DataService) ParentChildConsent(parentPatientID, childPatientID int64) (bool, error) {
	var consent bool
	row := d.db.QueryRow(`SELECT consented FROM patient_parent WHERE patient_id = ? AND parent_patient_id = ?`,
		childPatientID, parentPatientID)
	if err := row.Scan(&consent); err == sql.ErrNoRows {
		return false, errors.Trace(ErrNotFound("patient_parent"))
	} else if err != nil {
		return false, errors.Trace(err)
	}
	return consent, nil
}

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
