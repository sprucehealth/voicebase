package api

import (
	"database/sql"
)

type DataService struct {
	DB *sql.DB
}

func (d *DataService) CreatePhotoForCase(caseId int64, photoType string) (int64, error) {
	// create a new photo for the case and mark it as pending upload
	res, err := d.DB.Exec("insert into CaseImage(case_id, photoType, status) values (?, ?, ?)", caseId, photoType, PHOTO_STATUS_PENDING_UPLOAD)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkPhotoUploadComplete(caseId, photoId int64) error {
	_, err := d.DB.Exec("update CaseImage set status = ? where id = ? and case_id = ?", PHOTO_STATUS_PENDING_APPROVAL, photoId, caseId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) GetPhotosForCase(caseId int64) ([]string, error) {
	return make([]string, 1), nil
}
