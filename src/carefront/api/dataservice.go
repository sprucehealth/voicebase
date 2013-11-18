package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	NoRowsError = errors.New("No rows exist")
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
	var patientId int64
	err := d.DB.QueryRow("select id from patient where account_id = ?", accountId).Scan(&patientId)
	return patientId, err
}
func (d *DataService) GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error) {
	var patientId int64
	err := d.DB.QueryRow("select patient_id from patient_visit where id = ?", patientVisitId).Scan(&patientId)
	return patientId, err
}

func (d *DataService) getPatientAnswersForQuestionsBasedOnQuery(query string, args ...interface{}) (patientAnswers map[int64][]PatientAnswerToQuestion, err error) {
	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patientAnswers = make(map[int64][]PatientAnswerToQuestion)
	for rows.Next() {
		var answerId, questionId, potentialAnswerId, layoutVersionId int64
		var answerText, storageBucket, storageKey, storageRegion sql.NullString
		err = rows.Scan(&answerId, &questionId, &potentialAnswerId, &answerText, &storageBucket, &storageKey, &storageRegion, &layoutVersionId)
		if err != nil {
			return
		}
		if patientAnswers[questionId] == nil {
			patientAnswers[questionId] = make([]PatientAnswerToQuestion, 0)
		}
		patientAnswerToQuestion := PatientAnswerToQuestion{PatientInfoIntakeId: answerId,
			QuestionId:        questionId,
			PotentialAnswerId: potentialAnswerId,
			LayoutVersionId:   layoutVersionId,
		}
		if answerText.Valid {
			patientAnswerToQuestion.AnswerText = answerText.String
		}
		if storageBucket.Valid {
			patientAnswerToQuestion.StorageBucket = storageBucket.String
		}
		if storageRegion.Valid {
			patientAnswerToQuestion.StorageRegion = storageRegion.String
		}
		if storageKey.Valid {
			patientAnswerToQuestion.StorageKey = storageKey.String
		}
		patientAnswers[questionId] = append(patientAnswers[questionId], patientAnswerToQuestion)
	}
	return
}

func (d *DataService) GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]PatientAnswerToQuestion, err error) {
	queryStr := fmt.Sprintf(`select patient_info_intake.id, question_id, potential_answer_id, answer_text, object_storage.bucket, object_storage.storage_key, region_tag,  
								layout_version_id from patient_info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								where question_id in (%s) and patient_id = ? and patient_info_intake.status='ACTIVE'`, enumerateItemsIntoString(questionIds))
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId)
}

func (d *DataService) GetPatientAnswersForQuestionsInPatientVisit(questionIds []int64, patientId int64, patientVisitId int64) (patientAnswers map[int64][]PatientAnswerToQuestion, err error) {
	queryStr := fmt.Sprintf(`select patient_info_intake.id, question_id, potential_answer_id, answer_text, bucket, storage_key, region_tag, 
								layout_version_id from patient_info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								where question_id in (%s) and patient_id = ? and patient_visit_id = ? and patient_info_intake.status='ACTIVE'`, enumerateItemsIntoString(questionIds))
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId, patientVisitId)
}

func (d *DataService) GetGlobalSectionIds() (globalSectionIds []int64, err error) {
	rows, err := d.DB.Query(`select id from section where health_condition_id is null`)
	if err != nil {
		return nil, err
	}

	globalSectionIds = make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		globalSectionIds = append(globalSectionIds, sectionId)
	}
	return
}

func (d *DataService) GetSectionIdsForHealthCondition(healthConditionId int64) (sectionIds []int64, err error) {
	rows, err := d.DB.Query(`select id from section where health_condition_id = ?`, healthConditionId)
	if err != nil {
		return nil, err
	}

	sectionIds = make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		sectionIds = append(sectionIds, sectionId)
	}
	return
}

func (d *DataService) GetActivePatientVisitForHealthCondition(patientId, healthConditionId int64) (int64, error) {
	var patientVisitId int64
	err := d.DB.QueryRow("select id from patient_visit where patient_id = ? and health_condition_id = ? and status='OPEN'", patientId, healthConditionId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, err
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

func (d *DataService) GetQuestionType(questionId int64) (string, error) {
	var questionType string
	err := d.DB.QueryRow(`select qtype from question
						inner join question_type on question_type.id = qtype_id
						where question.id = ?`, questionId).Scan(&questionType)
	return questionType, err
}

func (d *DataService) GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where layout_version_id = ? and language_id = ?`, layoutVersionId, languageId).Scan(&bucket, &key, &region)
	return
}

func (d *DataService) GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where patient_layout_version.status='ACTIVE' and health_condition_id = ? and language_id = ?`, healthConditionId, languageId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error) {
	err = d.DB.QueryRow("select layout_version_id from patient_visit where id = ?", patientVisitId).Scan(&layoutVersionId)
	return
}

func (d *DataService) inactivatePreviousAnswersToQuestion(patientId, questionId, patientVisitId, layoutVersionId int64, tx *sql.Tx) (err error) {
	_, err = tx.Exec(`update patient_info_intake set status='INACTIVE' 
						where patient_id = ? and question_id = ?
						and patient_visit_id = ? and layout_version_id = ?`, patientId, questionId, patientVisitId, layoutVersionId)
	return err
}

func (d *DataService) getPatientAnswersForQuestion(patientId, questionId, patientVisitId int64) (patientAnswers []int64, err error) {
	patientAnswers = make([]int64, 0)
	rows, err := d.DB.Query(`select id from patient_info_intake 
								where patient_id = ? and patient_visit_id = ? and question_id = ? and status='ACTIVE'`, patientId, patientVisitId, questionId)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		patientAnswers = append(patientAnswers, id)
	}
	return
}

func (d *DataService) StoreFreeTextAnswersForQuestion(patientId, questionId, patientVisitId, layoutVersionId int64, answerIds []int64, answerTexts []string) (patientInfoIntakeIds []int64, err error) {
	tx, err := d.DB.Begin()

	err = d.inactivatePreviousAnswersToQuestion(patientId, questionId, patientVisitId, layoutVersionId, tx)
	if err != nil {
		tx.Rollback()
		return
	}

	insertStr := "insert into patient_info_intake (patient_id, patient_visit_id, question_id, potential_answer_id, answer_text, layout_version_id, status)  values"
	for i, answerId := range answerIds {
		valueStr := fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, 'ACTIVE')", patientId, patientVisitId, questionId, answerId, answerTexts[i], layoutVersionId)
		insertStr = insertStr + valueStr

		if i < (len(answerIds) - 1) {
			insertStr = insertStr + ","
		}
	}

	_, err = tx.Exec(insertStr)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	patientInfoIntakeIds, err = d.getPatientAnswersForQuestion(patientId, questionId, patientVisitId)
	return
}

func (d *DataService) StoreChoiceAnswersForQuestion(patientId, questionId, patientVisitId, layoutVersionId int64, answerIds []int64) (patientInfoIntakeIds []int64, err error) {
	tx, err := d.DB.Begin()

	err = d.inactivatePreviousAnswersToQuestion(patientId, questionId, patientVisitId, layoutVersionId, tx)
	if err != nil {
		tx.Rollback()
		return
	}

	insertStr := "insert into patient_info_intake (patient_id, patient_visit_id, question_id, potential_answer_id, layout_version_id, status) values"
	for i, answerId := range answerIds {
		valueStr := fmt.Sprintf("(%d, %d, %d, %d, %d, 'ACTIVE')", patientId, patientVisitId, questionId, answerId, layoutVersionId)
		insertStr = insertStr + valueStr

		if i < (len(answerIds) - 1) {
			insertStr = insertStr + ","
		}
	}

	_, err = tx.Exec(insertStr)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()

	patientInfoIntakeIds, err = d.getPatientAnswersForQuestion(patientId, questionId, patientVisitId)
	return
}

func (d *DataService) CreatePhotoAnswerForQuestionRecord(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error) {
	res, err := d.DB.Exec(`insert into patient_info_intake (patient_id, patient_visit_id, question_id, potential_answer_id, layout_version_id, status) 
							values (?, ?, ?, ?, ?, 'PENDING_UPLOAD')`, patientId, patientVisitId, questionId, potentialAnswerId, layoutVersionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastId, nil
}

func (d *DataService) UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error {
	_, err := d.DB.Exec(`update patient_info_intake set object_storage_id = ?, status='ACTIVE' where id = ?`, objectStorageId, patientInfoIntakeId)
	return err
}

func (d *DataService) MakeCurrentPhotoAnswerInactive(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) error {
	_, err := d.DB.Exec(`update patient_info_intake set status='INACTIVE' where patient_id = ? and question_id = ? 
							and patient_visit_id = ? and potential_answer_id = ? 
							and layout_version_id = ?`, patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId)
	return err
}

func (d *DataService) GetHealthConditionInfo(healthConditionTag string) (int64, error) {
	var id int64
	err := d.DB.QueryRow("select id from health_condition where comment = ? ", healthConditionTag).Scan(&id)
	return id, err
}

func (d *DataService) GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error) {
	err = d.DB.QueryRow(`select section.id, ltext from section 
					inner join app_text on section_title_app_text_id = app_text.id 
					inner join localized_text on app_text_id = app_text.id 
						where language_id = ? and section_tag = ?`, languageId, sectionTag).Scan(&id, &title)
	return
}

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, err error) {
	var byteQuestionTitle, byteQuestionType []byte
	err = d.DB.QueryRow(
		`select question.id, ltext, qtype from question 
		 left outer join localized_text on app_text_id=qtext_app_text_id
			left outer join question_type on qtype_id=question_type.id 
				where question_tag = ? and (ltext is NULL or language_id = ?)`,
		questionTag, languageId).Scan(&id, &byteQuestionTitle, &byteQuestionType)
	questionTitle = string(byteQuestionTitle)
	questionType = string(byteQuestionType)
	return
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error) {
	rows, err := d.DB.Query(`select potential_answer.id, ltext, atype, potential_answer_tag, ordering from potential_answer 
								inner join localized_text on answer_localized_text=app_text_id 
								inner join answer_type on atype_id=answer_type.id 
									where question_id = ? and language_id = ?`, questionId, languageId)
	if err != nil {
		return
	}
	defer rows.Close()
	answerInfos = make([]PotentialAnswerInfo, 0)
	for rows.Next() {
		var id, ordering int64
		var answer, answerType, answerTag string
		err = rows.Scan(&id, &answer, &answerType, &answerTag, &ordering)
		answerInfos = append(answerInfos, PotentialAnswerInfo{PotentialAnswerId: id,
			Answer:     answer,
			AnswerType: answerType,
			AnswerTag:  answerTag,
			Ordering:   ordering})
		if err != nil {
			return
		}
	}
	return
}

func (d *DataService) GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error) {
	err = d.DB.QueryRow(`select tips.id, ltext from tips
								inner join localized_text on app_text_id=tips_text_id 
									where tips_tag = ? and language_id = ?`, tipTag, languageId).Scan(&id, &tip)
	return
}

func (d *DataService) GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error) {
	err = d.DB.QueryRow(`select tips_section.id, ltext1.ltext, ltext2.ltext from tips_section 
								inner join localized_text as ltext1 on tips_title_text_id=ltext1.app_text_id 
								inner join localized_text as ltext2 on tips_subtext_text_id=ltext2.app_text_id 
									where ltext1.language_id = ? and tips_section_tag = ?`, languageId, tipSectionTag).Scan(&id, &tipSectionTitle, &tipSectionSubtext)
	return
}

func (d *DataService) GetActiveLayoutInfoForHealthCondition(healthConditionTag string) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status='ACTIVE' and health_condition.health_condition_tag = ?`, healthConditionTag).Scan(&bucket, &key, &region)
	return
}

func (d *DataService) GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error) {
	rows, err := d.DB.Query(`select id,language from languages_supported`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	languagesSupported = make([]string, 0)
	languagesSupportedIds = make([]int64, 0)
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

	updateStr := fmt.Sprintf(`update patient_layout_version set status='ACTIVE' where id in (%s)`, enumerateItemsIntoString(clientLayoutIds))
	_, err = d.DB.Exec(updateStr)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func enumerateItemsIntoString(ids []int64) string {
	if ids == nil || len(ids) == 0 {
		return ""
	}
	idsStr := make([]string, 0)
	for _, id := range ids {
		idsStr = append(idsStr, strconv.FormatInt(id, 10))
	}
	return strings.Join(idsStr, ",")
}
