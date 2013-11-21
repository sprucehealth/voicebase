package api

import (
	"bytes"
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

const (
	status_creating = "CREATING"
	status_active   = "ACTIVE"
	status_inactive = "INACTIVE"
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

func (d *DataService) updatePatientInfoIntakesWithStatus(questionIds []int64, patientId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {
	updateStr := fmt.Sprintf(`update patient_info_intake set status='%s' 
						where patient_id = ? and question_id in (%s)
						and patient_visit_id = ? and layout_version_id = ? and status='%s'`, status, enumerateItemsIntoString(questionIds), previousStatus)
	_, err = tx.Exec(updateStr, patientId, patientVisitId, layoutVersionId)
	return err
}

// This private helper method is to make it possible to update the status of sub answers to any
// top level question as specified, so as to atomically update the state of the answers that are relevant
// only in combination with the top-level answer to the question. This method makes it possible
// to change the status of the entire set in an atomic fashion.
func (d *DataService) updateSubAnswersToPatientInfoIntakesWithStatus(questionIds []int64, patientId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {
	parentInfoIntakeIds := make([]int64, 0)
	queryStr := fmt.Sprintf(`select id from patient_info_intake where patient_id = ? and question_id in (%s)
								and patient_visit_id = ? and layout_version_id = ? and status='%s'`, enumerateItemsIntoString(questionIds), previousStatus)
	rows, err := tx.Query(queryStr, patientId, patientVisitId, layoutVersionId)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		rows.Scan(&id)
		parentInfoIntakeIds = append(parentInfoIntakeIds, id)
	}

	updateStr := fmt.Sprintf(`update patient_info_intake set status='%s' 
						where parent_info_intake_id in (%s)`, status, enumerateItemsIntoString(parentInfoIntakeIds))
	_, err = tx.Exec(updateStr)
	return err
}

func (d *DataService) getPatientAnswersForQuestions(questionIds []int64, patientId, patientVisitId int64, status string) (answerIdToInfoIntakeIdMap map[int64]int64, err error) {
	queryStr := fmt.Sprintf(`select id, potential_answer_id from patient_info_intake
					where patient_id = ? and patient_visit_id = ? and question_id in (%s) and status='%s'`, enumerateItemsIntoString(questionIds), status)
	rows, err := d.DB.Query(queryStr, patientId, patientVisitId)
	if err != nil {
		return
	}
	defer rows.Close()

	answerIdToInfoIntakeIdMap = make(map[int64]int64)
	for rows.Next() {
		var id, potentialAnswerId int64
		rows.Scan(&id, &potentialAnswerId)
		answerIdToInfoIntakeIdMap[potentialAnswerId] = id
	}
	return
}

func (d *DataService) deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap map[int64]int64) error {
	// delete all ids that were in CREATING state since they were committed in that state
	values := createValuesArrayFromMap(answerIdToInfoIntakeIdMap)
	query := fmt.Sprintf("delete from patient_info_intake where id in (%s)", enumerateItemsIntoString(values))
	_, err := d.DB.Exec(query)
	return err
}

func prepareQueryForAnswers(answersToStore []AnswerToStore, parentInfoIntakeId string, status string) string {
	var buffer bytes.Buffer
	insertStr := `insert into patient_info_intake (patient_id, patient_visit_id, parent_info_intake_id, question_id, potential_answer_id, answer_text, layout_version_id, status) values`
	buffer.WriteString(insertStr)
	values := constructValuesToInsert(answersToStore, parentInfoIntakeId, status)
	buffer.WriteString(strings.Join(values, ","))
	return buffer.String()
}

func constructValuesToInsert(answersToStore []AnswerToStore, parentInfoIntakeId, status string) []string {
	values := make([]string, 0)
	for _, answerToStore := range answersToStore {
		valueStr := fmt.Sprintf("(%d, %d, %s, %d, %d, '%s', %d, '%s')", answerToStore.PatientId, answerToStore.PatientVisitId, parentInfoIntakeId,
			answerToStore.QuestionId, answerToStore.PotentialAnswerId, answerToStore.AnswerText, answerToStore.LayoutVersionId, status)
		values = append(values, valueStr)
	}
	return values
}

func (d *DataService) StoreAnswersForQuestion(questionId, patientId, patientVisitId, layoutVersionId int64, answersToStore []AnswerToStore) (err error) {

	if len(answersToStore) == 0 {
		return
	}

	// are there any answers to subquestions that we have to store.
	// the reason to determine this early on is because it changes the nature of the inserts/updates
	// to be done. If there are no subquestions, then we don't have to first commit the top level answers
	// to get the patient info intake ids to give as the parent info intake ids for the subquestions.
	// If there are subquestions, then we have to commit the top level answers first, get their ids, and then
	// commit the sub questions along with the parent info intake ids to indicate that the answers don't make sense
	// by themselve, but have to be linked to the parent answer to make them useful
	subAnswersFound := false
	for _, answerToStore := range answersToStore {
		if answerToStore.SubAnswers != nil {
			subAnswersFound = true
			break
		}
	}

	//keep track of all question ids for which we are storing answers. Note that while we are dealing
	// with a single top level question support for now, there are still subquestions pertanining to the question
	questionIds := make(map[int64]bool)
	questionIds[questionId] = true

	tx, err := d.DB.Begin()
	if err != nil {
		return
	}

	status := status_creating
	// if there are no subanswers found, then we are pretty much done with the insertion of the
	// answers into the database.
	if !subAnswersFound {
		// ensure to update the status of any prior subquestions linked to the responses
		// of the top level questions that need to be inactivated, along with the answers
		// to the top level question itself.
		d.updateSubAnswersToPatientInfoIntakesWithStatus([]int64{questionId}, patientId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)
		// if there are no subanswers to store, our job is done with just the top level answers
		d.updatePatientInfoIntakesWithStatus([]int64{questionId}, patientId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)
		status = status_active
	}

	insertStr := prepareQueryForAnswers(answersToStore, "NULL", status)
	_, err = tx.Exec(insertStr)
	if err != nil {
		return
	}
	tx.Commit()

	if !subAnswersFound {
		// nothing more to do after we have committed the top level answers
		return
	}

	// populate a map from potentialAnswerId to the infoIntakeId for the top level answers that were just
	// stored in the database. The reason for doing this is to be able to map each subquestion to its parent
	// patient info intake in the database, which is only possible by getting the ids from the database
	answerIdToInfoIntakeIdMap, err := d.getPatientAnswersForQuestions([]int64{questionId}, patientId, patientVisitId, status_creating)
	if err != nil {
		// clean up in the case of error given that we committed the last change
		d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
		return
	}

	var buffer bytes.Buffer
	for _, answerToStore := range answersToStore {
		if answerToStore.SubAnswers != nil {
			if buffer.Len() == 0 {
				buffer.WriteString(prepareQueryForAnswers(answerToStore.SubAnswers, strconv.FormatInt(answerIdToInfoIntakeIdMap[answerToStore.PotentialAnswerId], 10), status_creating))
			} else {
				values := constructValuesToInsert(answerToStore.SubAnswers, strconv.FormatInt(answerIdToInfoIntakeIdMap[answerToStore.PotentialAnswerId], 10), status_creating)
				buffer.WriteString(",")
				buffer.WriteString(strings.Join(values, ","))
			}
			// keep track of all questions for which we are storing answers
			for _, subAnswer := range answerToStore.SubAnswers {
				questionIds[subAnswer.QuestionId] = true
			}
		}
	}

	// start a new transaction to store the answers to the sub questions
	tx, err = d.DB.Begin()
	if err != nil {
		d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
		return
	}
	if subAnswersFound {
		insertStr = buffer.String()
		_, err = tx.Exec(insertStr)
		if err != nil {
			tx.Rollback()
			d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
			return
		}
	}

	// deactivate all answers to top level questions as well as their sub-questions
	// as we make new the answers the most current up-to-date patient info intake
	// Note: first update the answers to the sub questions to be inactive since we need the
	// answers to be able to identify which top level answers we care about
	err = d.updateSubAnswersToPatientInfoIntakesWithStatus([]int64{questionId}, patientId,
		patientVisitId, layoutVersionId, status_inactive, status_active, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
		return
	}

	err = d.updatePatientInfoIntakesWithStatus(createKeysArrayFromMap(questionIds), patientId,
		patientVisitId, layoutVersionId, status_inactive, status_active, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
		return
	}

	// make all answers pertanining to the questionIds collected the new active set of answers for the
	// questions traversed
	err = d.updatePatientInfoIntakesWithStatus(createKeysArrayFromMap(questionIds), patientId,
		patientVisitId, layoutVersionId, status_active, status_creating, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithIdInMap(answerIdToInfoIntakeIdMap)
		return
	}
	tx.Commit()
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

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, parentQuestionId int64, err error) {
	var byteQuestionTitle, byteQuestionType []byte
	var nullParentQuestionId sql.NullInt64
	err = d.DB.QueryRow(
		`select question.id, ltext, qtype, parent_question_id from question 
		 left outer join localized_text on app_text_id=qtext_app_text_id
			left outer join question_type on qtype_id=question_type.id 
				where question_tag = ? and (ltext is NULL or language_id = ?)`,
		questionTag, languageId).Scan(&id, &byteQuestionTitle, &byteQuestionType, &nullParentQuestionId)
	if nullParentQuestionId.Valid {
		parentQuestionId = nullParentQuestionId.Int64
	}
	questionTitle = string(byteQuestionTitle)
	questionType = string(byteQuestionType)
	return
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error) {
	rows, err := d.DB.Query(`select potential_answer.id, ltext, atype, potential_answer_tag, ordering from potential_answer 
								inner join localized_text on answer_localized_text_id=app_text_id 
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

func createKeysArrayFromMap(m map[int64]bool) []int64 {
	keys := make([]int64, 0)
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

func createValuesArrayFromMap(m map[int64]int64) []int64 {
	values := make([]int64, 0)
	for _, value := range m {
		values = append(values, value)
	}
	return values
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
