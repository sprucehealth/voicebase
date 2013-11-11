package api

import (
	"database/sql"
	"log"
	"time"
)

type DataService struct {
	DB *sql.DB
}

func (d *DataService) RegisterPatient(accountId int64, firstName, lastName, gender, zipCode string, dob time.Time) (int64, error) {
	res, err := d.DB.Exec(`insert into patient (account_id, first_name, last_name, zip_code, gender, dob, status) 
								values (?, ?, ?, ?, ?, ? , 'REGISTERED')`, accountId, firstName, lastName, zipCode, gender, dob)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return 0, err
	}
	return lastId, err
}

func (d *DataService) GetPatientIdFromAccountId(accountId int64) (int64, error) {
	rows, err := d.DB.Query("select id from patient where account_id = ?", accountId)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var patientId int64
	rows.Next()
	rows.Scan(&patientId)
	return patientId, nil
}

func (d *DataService) GetActivePatientVisitForHealthCondition(patientId, healthConditionId int64) (int64, error) {
	rows, err := d.DB.Query("select id from patient_visit where patient_id = ? and health_condition_id = ? and status='OPEN'", patientId, healthConditionId)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var patientVisitId int64

	if !rows.Next() {
		return -1, nil
	}

	rows.Scan(&patientVisitId)
	return patientVisitId, nil
}

func (d *DataService) CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_visit (patient_id, opened_date, health_condition_id, layout_version_id, status) 
								values (?, now(), ?, ?, 'OPEN')`, patientId, healthConditionId, layoutVersionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return 0, err
	}
	return lastId, err
}

func (d *DataService) GetQuestionType(questionId int64) (questionType string, err error) {
	rows, err := d.DB.Query(`select qtype from question
								inner join question_type on question_type.id = qtype_id
								where question.id = ?`, questionId)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		return "", nil
	}

	rows.Scan(&questionType)
	return questionType, nil
}

func (d *DataService) GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	rows, err := d.DB.Query(` select bucket, storage_key, region_tag, layout_version_id from patient_layout_version 
								inner join object_storage on object_storage_id=object_storage.id 
								inner join region on region_id=region.id 
									where patient_layout_version.status='ACTIVE' and health_condition_id = ? and language_id = ?`, healthConditionId, languageId)
	if err != nil {
		return "", "", "", 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return "", "", "", 0, nil
	}

	rows.Scan(&bucket, &storage, &region, &layoutVersionId)
	return bucket, storage, region, layoutVersionId, nil
}

func (d *DataService) GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error) {
	rows, err := d.DB.Query("select layout_version_id from patient_visit where id = ?", patientVisitId)
	if err != nil {
		return 0, err
	}

	if !rows.Next() {
		return 0, nil
	}

	rows.Scan(&layoutVersionId)

	return layoutVersionId, nil
}

func (d *DataService) inactivatePreviousAnswersToQuestion(patientId, questionId, sectionId, patientVisitId, layoutVersionId int64, tx *sql.Tx) (err error) {
	_, err = tx.Exec(`update patient_info_intake set status='INACTIVE' 
						where patient_id = ? and question_id = ? and section_id = ? 
						and patient_visit_id = ? and layout_version_id = ?`, patientId, questionId, sectionId, patientVisitId, layoutVersionId)
	return err
}

func (d *DataService) StoreFreeTextAnswersForQuestion(patientId, questionId, sectionId, patientVisitId, layoutVersionId int64, answerIds []int64, answerTexts []string, toUpdate bool) (patientInfoIntakeIds []int64, err error) {
	tx, err := d.DB.Begin()

	if toUpdate {
		err = d.inactivatePreviousAnswersToQuestion(patientId, questionId, sectionId, patientVisitId, layoutVersionId, tx)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	patientInfoIntakeIds = make([]int64, len(answerIds))
	for i, answerId := range answerIds {
		res, err := tx.Exec(`insert into patient_info_intake (patient_id, patient_visit_id, question_id, section_id, potential_answer_id, answer_text, layout_version_id, status) 
							values (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`, patientId, patientVisitId, questionId, sectionId, answerId, answerTexts[i], layoutVersionId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		lastId, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		patientInfoIntakeIds[i] = lastId
	}

	tx.Commit()
	return patientInfoIntakeIds, nil
}

func (d *DataService) StoreChoiceAnswersForQuestion(patientId, questionId, sectionId, patientVisitId, layoutVersionId int64, answerIds []int64, toUpdate bool) (patientInfoIntakeIds []int64, err error) {
	tx, err := d.DB.Begin()

	if toUpdate {
		err = d.inactivatePreviousAnswersToQuestion(patientId, questionId, sectionId, patientVisitId, layoutVersionId, tx)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	patientInfoIntakeIds = make([]int64, len(answerIds))
	for i, answerId := range answerIds {
		res, err := tx.Exec(`insert into patient_info_intake (patient_id, patient_visit_id, question_id, section_id, potential_answer_id, layout_version_id, status) 
							values (?, ?, ?, ?, ?, ?, 'ACTIVE')`, patientId, patientVisitId, questionId, sectionId, answerId, layoutVersionId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		lastId, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		patientInfoIntakeIds[i] = lastId
	}

	tx.Commit()
	return patientInfoIntakeIds, nil
}

func (d *DataService) CreatePhotoForCase(caseId int64, photoType string) (int64, error) {
	// create a new photo for the case and mark it as pending upload
	res, err := d.DB.Exec("insert into case_image(case_id, photoType, status) values (?, ?, ?)", caseId, photoType, PHOTO_STATUS_PENDING_UPLOAD)
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
	_, err := d.DB.Exec("update case_image set status = ? where id = ? and case_id = ?", PHOTO_STATUS_PENDING_APPROVAL, photoId, caseId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) GetPhotosForCase(caseId int64) ([]string, error) {
	return make([]string, 1), nil
}

func (d *DataService) GetHealthConditionInfo(healthConditionTag string) (int64, error) {
	rows, err := d.DB.Query("select id from health_condition where comment = ? ", healthConditionTag)
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

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) (ids []int64, answers []string, answerTypes []string, answerTags []string, orderings []int64, err error) {
	rows, err := d.DB.Query(`select potential_answer.id, ltext, atype, potential_answer_tag, ordering from potential_answer 
								inner join localized_text on answer_localized_text=app_text_id 
								inner join answer_type on atype_id=answer_type.id 
									where question_id = ? and language_id = ?`, questionId, languageId)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer rows.Close()
	ids = make([]int64, 0, 5)
	answers = make([]string, 0, 5)
	answerTypes = make([]string, 0, 5)
	orderings = make([]int64, 0, 5)
	answerTags = make([]string, 0, 5)
	for rows.Next() {
		var id, ordering int64
		var answer, answerType, answerTag string
		err = rows.Scan(&id, &answer, &answerType, &answerTag, &ordering)
		ids = append(ids, id)
		answers = append(answers, answer)
		answerTypes = append(answerTypes, answerType)
		orderings = append(orderings, ordering)
		answerTags = append(answerTags, answerTag)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return ids, answers, answerTypes, answerTags, orderings, nil
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

func (d *DataService) GetActiveLayoutInfoForHealthCondition(healthConditionTag string) (bucket, key, region string, err error) {
	rows, err := d.DB.Query(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status='ACTIVE' and health_condition.health_condition_tag = ?`, healthConditionTag)
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

func (d *DataService) MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, comment string) (int64, error) {
	res, err := d.DB.Exec(`insert into layout_version (object_storage_id, syntax_version, health_condition_id, comment, status) 
							values (?, ?, ?, ?, 'CREATING')`, objectId, syntaxVersion, healthConditionId, comment)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_layout_version (object_storage_id, language_id, layout_version_id, health_condition_id, status) 
								values (?, ?, ?, ?, 'CREATING')`, objectId, languageId, layoutVersionId, healthConditionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) UpdateActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error {
	tx, _ := d.DB.Begin()
	// update the current active layouts to DEPRECATED
	_, err := d.DB.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the current client active layouts to DEPRECATED
	_, err = d.DB.Exec(`update patient_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ?`, healthConditionId)
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
