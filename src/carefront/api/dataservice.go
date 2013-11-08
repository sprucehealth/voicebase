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

func (d *DataService) GetTreatmentInfo(treatmentTag string) (int64, error) {
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

func (d *DataService) GetOutcomeInfo(questionId int64, languageId int64) (ids []int64, outcomes []string, outcomeTypes []string, outcomeTags []string, orderings []int64, err error) {
	rows, err := d.DB.Query(`select potential_outcome.id, ltext, otype, potential_outcome_tag, ordering from potential_outcome 
								inner join localized_text on outcome_localized_text=app_text_id 
								inner join outcome_type on otype_id=outcome_type.id 
									where question_id = ? and language_id = ?`, questionId, languageId)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer rows.Close()
	ids = make([]int64, 0, 5)
	outcomes = make([]string, 0, 5)
	outcomeTypes = make([]string, 0, 5)
	orderings = make([]int64, 0, 5)
	outcomeTags = make([]string, 0, 5)
	for rows.Next() {
		var id, ordering int64
		var outcome, outcomeType, outcomeTag string
		err = rows.Scan(&id, &outcome, &outcomeType, &outcomeTag, &ordering)
		ids = append(ids, id)
		outcomes = append(outcomes, outcome)
		outcomeTypes = append(outcomeTypes, outcomeType)
		orderings = append(orderings, ordering)
		outcomeTags = append(outcomeTags, outcomeTag)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return ids, outcomes, outcomeTypes, outcomeTags, orderings, nil
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

func (d *DataService) GetActiveLayoutInfoForTreatment(treatmentTag string) (bucket, key, region string, err error) {
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

func (d *DataService) GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error) {
	rows, err := d.DB.Query(`select id,language from languages_supported`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	languagesSupported = make([]string, 0, 5)
	languagesSupportedIds = make([]int64, 0, 5)
	for rows.Next() {
		var languageId int64
		var language string
		err := rows.Scan(&languageId, &language)
		if err != nil {
			return nil, nil, err
		}
		languagesSupported = append(languagesSupported, language)
		languagesSupportedIds = append(languagesSupportedIds, languageId)
	}
	return languagesSupported, languagesSupportedIds, nil
}

func (d *DataService) CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error) {
	res, err := d.DB.Exec(`insert into object_storage (bucket, storage_key, status, region_id) 
								values (?, ?, 'CREATING', (select id from region where region_tag = ?))`, bucket, key, region)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) UpdateCloudObjectRecordToSayCompleted(id int64) error {
	_, err := d.DB.Exec("update object_storage set status='ACTIVE' where id = ?", id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DataService) MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, treatmentId int64, comment string) (int64, error) {
	res, err := d.DB.Exec(`insert into layout_version (object_storage_id, syntax_version, treatment_id, comment, status) 
							values (?, ?, ?, ?, 'CREATING')`, objectId, syntaxVersion, treatmentId, comment)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, treatmentId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_layout_version (object_storage_id, language_id, layout_version_id, treatment_id, status) 
								values (?, ?, ?, ?, 'CREATING')`, objectId, languageId, layoutVersionId, treatmentId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) UpdateActiveLayouts(layoutId int64, clientLayoutIds []int64, treatmentId int64) error {
	tx, _ := d.DB.Begin()
	// update the current active layouts to DEPRECATED
	_, err := d.DB.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and treatment_id = ?`, treatmentId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the current client active layouts to DEPRECATED
	_, err = d.DB.Exec(`update patient_layout_version set status='DEPCRECATED' where status='ACTIVE' and treatment_id = ?`, treatmentId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = d.DB.Exec(`update layout_version set status='ACTIVE' where id = ?`, layoutId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, clientLayoutId := range clientLayoutIds {
		_, err := d.DB.Exec(`update patient_layout_version set status='ACTIVE' where id = ?`, clientLayoutId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}
