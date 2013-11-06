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

func (d *DataService) GetTreatmentInfo(treatmentTag string, languageId int64) (int64, error) {
	rows, err := d.DB.Query("select id from treatment where comment = ? ", treatmentTag)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int64
	rows.Next()
	err = rows.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d *DataService) GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error) {
	rows, err := d.DB.Query(`select section.id, ltext from section 
								inner join app_text on section_title_app_text_id = app_text.id 
								inner join localized_text on app_text_id = app_text.id 
									where language_id = ? and section_tag = ?`, languageId, sectionTag)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &title)
	if err != nil {
		return 0, "", err
	}
	return id, title, nil
}

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, err error) {
	rows, err := d.DB.Query(`select question.id, ltext, qtype from question 
								inner join app_text on qtext_app_text_id=app_text.id 
								inner join localized_text on app_text_id=app_text.id
	   							inner join question_type on qtype_id=question_type.id 
	   								where question_tag = ? and language_id = ?`, questionTag, languageId)
	if err != nil {
		return 0, "", "", err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &questionTitle, &questionType)
	if err != nil {
		return 0, "", "", err
	}
	return id, questionTitle, questionType, nil
}
