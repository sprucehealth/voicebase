package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) AddMedia(uploaderID int64, url, mimetype string) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO media (uploader_id, url, mimetype) VALUES (?, ?, ?)`,
		uploaderID, url, mimetype)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) GetMedia(mediaID int64) (*common.Media, error) {
	media := &common.Media{
		ID: mediaID,
	}
	if err := d.db.QueryRow(`
		SELECT uploaded_date, uploader_id, url, mimetype
		FROM media
		WHERE id = ?`, mediaID,
	).Scan(
		&media.Uploaded, &media.UploaderID, &media.URL, &media.Mimetype,
	); err == sql.ErrNoRows {
		return nil, ErrNotFound("media")
	} else if err != nil {
		return nil, err
	}
	return media, nil
}

func (d *DataService) MediaHasClaim(mediaID int64, claimerType string, claimerID int64) (bool, error) {
	var x int
	row := d.db.QueryRow(`
		SELECT 1 FROM media_claim
		WHERE media_id = ? AND claimer_type = ? AND claimer_id = ?`,
		mediaID, claimerType, claimerID)
	err := row.Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DataService) ClaimMedia(mediaID int64, claimerType string, claimerID int64) error {
	return d.claimMedia(d.db, mediaID, claimerType, claimerID)
}

func (d *DataService) UnclaimMedia(mediaID int64, claimerType string, claimerID int64) error {
	return d.unclaimMedia(d.db, mediaID, claimerType, claimerID)
}

func (d *DataService) claimMedia(db db, mediaID int64, claimerType string, claimerID int64) error {
	_, err := db.Exec(`
		INSERT INTO media_claim (media_id, claimer_type, claimer_id)
		VALUES (?, ?, ?)`,
		mediaID, claimerType, claimerID)
	return err
}

func (d *DataService) unclaimMedia(db db, mediaID int64, claimerType string, claimerID int64) error {
	_, err := db.Exec(`
		DELETE FROM media_claim WHERE media_id = ? AND claimer_type = ? AND claimer_id = ?`,
		mediaID, claimerType, claimerID)
	return err
}
