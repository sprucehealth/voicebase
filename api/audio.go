package api

import (
	"database/sql"
	"github.com/sprucehealth/backend/common"
)

func (d *DataService) AddAudio(uploaderID int64, url, mimetype string) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO audio (uploader_id, url, mimetype) VALUES (?, ?, ?)`,
		uploaderID, url, mimetype)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) GetAudio(audioID int64) (*common.Audio, error) {
	audio := &common.Audio{
		Id: audioID,
	}
	var claimerType sql.NullString
	var claimerID sql.NullInt64
	if err := d.db.QueryRow(`
		SELECT uploaded, uploader_id, url, mimetype, claimer_type, claimer_id
		FROM audio
		WHERE id = ?`, audioID,
	).Scan(
		&audio.Uploaded, &audio.UploaderID, &audio.URL, &audio.Mimetype,
		&claimerType, &claimerID,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	audio.ClaimerType = claimerType.String
	audio.ClaimerID = claimerID.Int64
	return audio, nil
}

func (d *DataService) ClaimAudio(audioID int64, claimerType string, claimerID int64) error {
	return d.claimAudio(d.db, audioID, claimerType, claimerID)
}

func (d *DataService) claimAudio(db db, audioID int64, claimerType string, claimerID int64) error {
	_, err := db.Exec(`
		UPDATE audio
		SET claimer_type = ?, claimer_id = ?
		WHERE id = ?`, claimerType, claimerID, audioID)
	return err
}
