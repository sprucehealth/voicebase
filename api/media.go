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
		Id: mediaID,
	}
	var claimerType sql.NullString
	var claimerID sql.NullInt64
	if err := d.db.QueryRow(`
		SELECT uploaded, uploader_id, url, mimetype, claimer_type, claimer_id
		FROM media
		WHERE id = ?`, mediaID,
	).Scan(
		&media.Uploaded, &media.UploaderID, &media.URL, &media.Mimetype,
		&claimerType, &claimerID,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	media.ClaimerType = claimerType.String
	media.ClaimerID = claimerID.Int64
	return media, nil
}

func (d *DataService) ClaimMedia(mediaID int64, claimerType string, claimerID int64) error {
	return d.claimMedia(d.db, mediaID, claimerType, claimerID)
}

func (d *DataService) claimMedia(db db, mediaID int64, claimerType string, claimerID int64) error {
	_, err := db.Exec(`
		UPDATE media
		SET claimer_type = ?, claimer_id = ?
		WHERE id = ?`, claimerType, claimerID, mediaID)
	return err
}
