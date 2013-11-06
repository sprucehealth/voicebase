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
								left outer join localized_text on app_text_id=qtext_app_text_id
	   							left outer join question_type on qtype_id=question_type.id 
	   								where question_tag = ? and (ltext is NULL or language_id = ?)`, questionTag, languageId)
	if err != nil {
		return 0, "", "", err
	}
	defer rows.Close()
	rows.Next()
	var byteQuestionTitle, byteQuestionType []byte
	err = rows.Scan(&id, &byteQuestionTitle, &byteQuestionType)
	if err != nil {
		return 0, "", "", err
	}
	questionTitle = string(byteQuestionTitle)
	questionType = string(byteQuestionType)
	return id, questionTitle, questionType, nil
}

func (d *DataService) GetOutcomeInfo(outcomeTag string, languageId int64) (id int64, outcome string, outcomeType string, err error) {
	rows, err := d.DB.Query(`select potential_outcome.id, ltext, outcome_type.otype from potential_outcome 
								inner join outcome_type on otype_id=outcome_type.id 
								inner join localized_text on outcome_localized_text=app_text_id 
									where potential_outcome_Tag=? and language_id=?`, outcomeTag, languageId)
	if err != nil {
		return 0, "", "", err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &outcome, &outcomeType)
	if err != nil {
		return 0, "", "", err
	}
	return id, outcome, outcomeType, nil
}

func (d *DataService) GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error) {
	rows, err := d.DB.Query(`select tips.id, ltext from tips
								inner join localized_text on app_text_id=tips_text_id 
									where tips_tag = ? and language_id = ?`, tipTag, languageId)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &tip)
	if err != nil {
		return 0, "", err
	}
	return id, tip, nil
}

func (d *DataService) GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error) {
	rows, err := d.DB.Query(`select tips_section.id, ltext1.ltext, ltext2.ltext from tips_section 
								inner join localized_text as ltext1 on tips_title_text_id=ltext1.app_text_id 
								inner join localized_text as ltext2 on tips_subtext_text_id=ltext2.app_text_id 
									where ltext1.language_id = ? and tips_section_tag = ?`, languageId, tipSectionTag)
	if err != nil {
		return 0, "", "", err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &tipSectionTitle, &tipSectionSubtext)
	if err != nil {
		return 0, "", "", err
	}
	return id, tipSectionTitle, tipSectionSubtext, nil
}

func (d *DataService) GetCurrentActiveLayoutInfoForTreatment(treatmentTag string) (bucket, key, region string, err error) {
	rows, err := d.DB.Query(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join treatment on treatment_id = treatment.id 
									where layout_version.status='ACTIVE' and treatment.treatment_tag = ?`, treatmentTag)
	if err != nil {
		return "", "", "", err
	}
	defer rows.Close()
	// if there are no rows to return, return empty values
	if !rows.Next() {
		return "", "", "", nil
	}
	err = rows.Scan(&bucket, &key, &region)
	if err != nil {
		return "", "", "", err
	}
	return bucket, key, region, nil
}
