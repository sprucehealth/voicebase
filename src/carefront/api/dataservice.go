package api

import (
	"bytes"
	"carefront/common"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
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

func (d *DataService) GetPatientFromId(patientId int64) (patient *common.Patient, err error) {
	var firstName, lastName, zipCode, status, gender string
	var dob mysql.NullTime
	var accountId int64
	err = d.DB.QueryRow(`select account_id, first_name, last_name, zip_code, gender, dob, status from patient where id = ?`, patientId).Scan(&accountId, &firstName, &lastName, &zipCode, &gender, &dob, &status)
	if err != nil {
		return
	}
	patient = &common.Patient{
		FirstName: firstName,
		LastName:  lastName,
		ZipCode:   zipCode,
		Status:    status,
		Gender:    gender,
		AccountId: accountId,
	}
	if dob.Valid {
		patient.Dob = dob.Time
	}
	patient.PatientId = patientId
	return
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

func (d *DataService) getPatientAnswersForQuestionsBasedOnQuery(query string, args ...interface{}) (patientAnswers map[int64][]*common.PatientAnswer, err error) {
	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patientAnswers = make(map[int64][]*common.PatientAnswer)
	queriedAnswers := make([]*common.PatientAnswer, 0)
	for rows.Next() {
		var answerId, questionId, potentialAnswerId, layoutVersionId int64
		var answerText, answerSummaryText, storageBucket, storageKey, storageRegion, potentialAnswer sql.NullString
		var parentQuestionId, parentInfoIntakeId sql.NullInt64
		err = rows.Scan(&answerId, &questionId, &potentialAnswerId, &potentialAnswer, &answerSummaryText, &answerText, &storageBucket, &storageKey, &storageRegion, &layoutVersionId, &parentQuestionId, &parentInfoIntakeId)
		if err != nil {
			return
		}
		patientAnswerToQuestion := &common.PatientAnswer{PatientAnswerId: answerId,
			QuestionId:        questionId,
			PotentialAnswerId: potentialAnswerId,
			LayoutVersionId:   layoutVersionId,
		}

		if potentialAnswer.Valid {
			patientAnswerToQuestion.PotentialAnswer = potentialAnswer.String
		}
		if answerText.Valid {
			patientAnswerToQuestion.AnswerText = answerText.String
		}
		if answerSummaryText.Valid {
			patientAnswerToQuestion.AnswerSummary = answerSummaryText.String
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
		if parentQuestionId.Valid {
			patientAnswerToQuestion.ParentQuestionId = parentQuestionId.Int64
		}
		if parentInfoIntakeId.Valid {
			patientAnswerToQuestion.ParentAnswerId = parentInfoIntakeId.Int64
		}
		queriedAnswers = append(queriedAnswers, patientAnswerToQuestion)
	}

	// populate all top-level answers into the map
	patientAnswers = make(map[int64][]*common.PatientAnswer)
	for _, patientAnswerToQuestion := range queriedAnswers {
		if patientAnswerToQuestion.ParentQuestionId == 0 {
			questionId := patientAnswerToQuestion.QuestionId
			if patientAnswers[questionId] == nil {
				patientAnswers[questionId] = make([]*common.PatientAnswer, 0)
			}
			patientAnswers[questionId] = append(patientAnswers[questionId], patientAnswerToQuestion)
		}
	}

	// add all subanswers to the top-level answers by iterating through the queried answers
	// to identify any sub answers
	for _, patientAnswerToQuestion := range queriedAnswers {
		if patientAnswerToQuestion.ParentQuestionId != 0 {
			questionId := patientAnswerToQuestion.ParentQuestionId
			// go through the list of answers to identify the particular answer we care about
			for _, patientAnswer := range patientAnswers[questionId] {
				if patientAnswer.PatientAnswerId == patientAnswerToQuestion.ParentAnswerId {
					// this is the top level answer to
					if patientAnswer.SubAnswers == nil {
						patientAnswer.SubAnswers = make([]*common.PatientAnswer, 0)
					}
					patientAnswer.SubAnswers = append(patientAnswer.SubAnswers, patientAnswerToQuestion)
				}
			}
		}
	}
	return
}

func (d *DataService) GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]*common.PatientAnswer, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select patient_info_intake.id, potential_answer.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, object_storage.bucket, object_storage.storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from patient_info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (potential_answer.question_id in (%s) or parent_question_id in (%s)) and patient_id = ? and patient_info_intake.status='ACTIVE'`, enumeratedStrings, enumeratedStrings)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId)
}

func (d *DataService) GetPatientAnswersForQuestionsInPatientVisit(questionIds []int64, patientId int64, patientVisitId int64) (patientAnswers map[int64][]*common.PatientAnswer, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select patient_info_intake.id, potential_answer.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, bucket, storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from patient_info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (potential_answer.question_id in (%s) or parent_question_id in (%s)) and patient_id = ? and patient_visit_id = ? and patient_info_intake.status='ACTIVE'`, enumeratedStrings, enumeratedStrings)
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

func (d *DataService) GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error) {
	var patientVisitId int64
	err := d.DB.QueryRow("select id from patient_visit where patient_id = ? and health_condition_id = ? and status='OPEN'", patientId, healthConditionId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, err
}

func (d *DataService) GetPatientVisitFromId(patientVisitId int64) (patientVisit *common.PatientVisit, err error) {
	var patientId, healthConditionId, layoutVersionId int64
	var creationDateBytes, openedDateBytes, closedDateBytes mysql.NullTime
	var status string
	row := d.DB.QueryRow(`select patient_id, health_condition_id, layout_version_id, 
		creation_date, opened_date, closed_date, status from patient_visit where id = ?`, patientVisitId)
	err = row.Scan(&patientId, &healthConditionId, &layoutVersionId, &creationDateBytes, &openedDateBytes, &closedDateBytes, &status)
	if err != nil {
		return nil, err
	}
	patientVisit = &common.PatientVisit{
		PatientVisitId:    patientVisitId,
		PatientId:         patientId,
		HealthConditionId: healthConditionId,
		Status:            status,
		LayoutVersionId:   layoutVersionId,
	}

	if creationDateBytes.Valid {
		patientVisit.CreationDate = creationDateBytes.Time
	}
	if openedDateBytes.Valid {
		patientVisit.OpenedDate = openedDateBytes.Time
	}
	if closedDateBytes.Valid {
		patientVisit.ClosedDate = closedDateBytes.Time
	}

	return patientVisit, err
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

func (d *DataService) GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where patient_layout_version.status='ACTIVE' and health_condition_id = ? and language_id = ?`, healthConditionId, languageId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetStorageInfoOfCurrentActiveDoctorLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from dr_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where dr_layout_version.status='ACTIVE' and health_condition_id = ?`, healthConditionId)
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

// This private helper method is to make it possible to update the status of sub answers
// only in combination with the top-level answer to the question. This method makes it possible
// to change the status of the entire set in an atomic fashion.
func (d *DataService) updateSubAnswersToPatientInfoIntakesWithStatus(questionIds []int64, patientId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {

	if len(questionIds) == 0 {
		return
	}

	parentInfoIntakeIds := make([]int64, 0)
	queryStr := fmt.Sprintf(`select id from patient_info_intake where patient_id = ? and question_id in (%s) and patient_visit_id = ? and layout_version_id = ? and status='%s'`, enumerateItemsIntoString(questionIds), previousStatus)
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

	if len(parentInfoIntakeIds) == 0 {
		return
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

func (d *DataService) deleteAnswersWithId(answerIds []int64) error {
	// delete all ids that were in CREATING state since they were committed in that state
	query := fmt.Sprintf("delete from patient_info_intake where id in (%s)", enumerateItemsIntoString(answerIds))
	_, err := d.DB.Exec(query)
	return err
}

func prepareQueryForAnswers(answersToStore []*common.PatientAnswer, parentInfoIntakeId string, parentQuestionId string, status string) string {
	var buffer bytes.Buffer
	insertStr := `insert into patient_info_intake (patient_id, patient_visit_id, parent_info_intake_id, parent_question_id, question_id, potential_answer_id, answer_text, layout_version_id, status) values`
	buffer.WriteString(insertStr)
	values := constructValuesToInsert(answersToStore, parentInfoIntakeId, parentQuestionId, status)
	buffer.WriteString(strings.Join(values, ","))
	return buffer.String()
}

func constructValuesToInsert(answersToStore []*common.PatientAnswer, parentInfoIntakeId, parentQuestionId, status string) []string {
	values := make([]string, 0)
	for _, answerToStore := range answersToStore {
		valueStr := fmt.Sprintf("(%d, %d, %s, %s, %d, %d, '%s', %d, '%s')", answerToStore.PatientId, answerToStore.PatientVisitId, parentInfoIntakeId, parentQuestionId,
			answerToStore.QuestionId, answerToStore.PotentialAnswerId, answerToStore.AnswerText, answerToStore.LayoutVersionId, status)
		values = append(values, valueStr)
	}
	return values
}

func (d *DataService) StoreAnswersForQuestion(questionId, patientId, patientVisitId, layoutVersionId int64, answersToStore []*common.PatientAnswer) (err error) {

	if len(answersToStore) == 0 {
		return
	}

	// keep track of all question ids for which we are storing answers.
	questionIds := make(map[int64]bool)
	questionIds[questionId] = true

	tx, err := d.DB.Begin()
	if err != nil {
		return
	}

	infoIdToAnswersWithSubAnswers := make(map[int64]*common.PatientAnswer)
	subAnswersFound := false
	for _, answerToStore := range answersToStore {
		insertStr := prepareQueryForAnswers([]*common.PatientAnswer{answerToStore}, "NULL", "NULL", status_creating)
		res, err := tx.Exec(insertStr)
		if err != nil {
			tx.Rollback()
			return err
		}

		if answerToStore.SubAnswers != nil {
			subAnswersFound = true

			lastInsertId, err := res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
			infoIdToAnswersWithSubAnswers[lastInsertId] = answerToStore
		}
	}

	// if there are no subanswers found, then we are pretty much done with the insertion of the
	// answers into the database.
	if !subAnswersFound {
		// ensure to update the status of any prior subquestions linked to the responses
		// of the top level questions that need to be inactivated, along with the answers
		// to the top level question itself.
		d.updateSubAnswersToPatientInfoIntakesWithStatus([]int64{questionId}, patientId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)
		d.updatePatientInfoIntakesWithStatus([]int64{questionId}, patientId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)

		// if there are no subanswers to store, our job is done with just the top level answers
		d.updatePatientInfoIntakesWithStatus([]int64{questionId}, patientId,
			patientVisitId, layoutVersionId, status_active, status_creating, tx)
		tx.Commit()
		return
	}

	tx.Commit()
	// create a query to batch insert all subanswers
	var buffer bytes.Buffer
	for infoIntakeId, answerToStore := range infoIdToAnswersWithSubAnswers {
		if buffer.Len() == 0 {
			buffer.WriteString(prepareQueryForAnswers(answerToStore.SubAnswers,
				strconv.FormatInt(infoIntakeId, 10),
				strconv.FormatInt(answerToStore.QuestionId, 10), status_creating))
		} else {
			values := constructValuesToInsert(answerToStore.SubAnswers,
				strconv.FormatInt(infoIntakeId, 10),
				strconv.FormatInt(answerToStore.QuestionId, 10), status_creating)
			buffer.WriteString(",")
			buffer.WriteString(strings.Join(values, ","))
		}
		// keep track of all questions for which we are storing answers
		for _, subAnswer := range answerToStore.SubAnswers {
			questionIds[subAnswer.QuestionId] = true
		}
	}

	// start a new transaction to store the answers to the sub questions
	tx, err = d.DB.Begin()
	if err != nil {
		d.deleteAnswersWithId(infoIdsFromMap(infoIdToAnswersWithSubAnswers))
		return
	}

	insertStr := buffer.String()
	_, err = tx.Exec(insertStr)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithId(infoIdsFromMap(infoIdToAnswersWithSubAnswers))
		return
	}

	// deactivate all answers to top level questions as well as their sub-questions
	// as we make the new answers the most current up-to-date patient info intake
	err = d.updateSubAnswersToPatientInfoIntakesWithStatus([]int64{questionId}, patientId,
		patientVisitId, layoutVersionId, status_inactive, status_active, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithId(infoIdsFromMap(infoIdToAnswersWithSubAnswers))
		return
	}

	err = d.updatePatientInfoIntakesWithStatus(createKeysArrayFromMap(questionIds), patientId,
		patientVisitId, layoutVersionId, status_inactive, status_active, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithId(infoIdsFromMap(infoIdToAnswersWithSubAnswers))
		return
	}

	// make all answers pertanining to the questionIds collected the new active set of answers for the
	// questions traversed
	err = d.updatePatientInfoIntakesWithStatus(createKeysArrayFromMap(questionIds), patientId,
		patientVisitId, layoutVersionId, status_active, status_creating, tx)
	if err != nil {
		tx.Rollback()
		d.deleteAnswersWithId(infoIdsFromMap(infoIdToAnswersWithSubAnswers))
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

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, questionSummary string, parentQuestionId int64, additionalFields map[string]string, err error) {
	var byteQuestionTitle, byteQuestionType, byteQuestionSummary []byte
	var nullParentQuestionId sql.NullInt64
	err = d.DB.QueryRow(
		`select question.id, l1.ltext, qtype, parent_question_id, l2.ltext from question 
			left outer join localized_text as l1 on app_text_id=qtext_app_text_id
			left outer join question_type on qtype_id=question_type.id
			left outer join localized_text as l2 on qtext_short_text_id = l2.app_text_id
				where question_tag = ? and (l1.ltext is NULL or l1.language_id = ?)`,
		questionTag, languageId).Scan(&id, &byteQuestionTitle, &byteQuestionType, &nullParentQuestionId, &byteQuestionSummary)
	if nullParentQuestionId.Valid {
		parentQuestionId = nullParentQuestionId.Int64
	}
	questionTitle = string(byteQuestionTitle)
	questionType = string(byteQuestionType)
	questionSummary = string(byteQuestionSummary)

	// get any additional fields pertaining to the question from the database
	rows, err := d.DB.Query(`select question_field, ltext from question_fields
								inner join localized_text on app_text_id = localized_text.app_text_id
								where question_id = ? and language_id = ?`, id, languageId)
	for rows.Next() {
		var questionField, fieldText string
		err = rows.Scan(&questionField, &fieldText)
		if err != nil {
			return
		}
		if additionalFields == nil {
			additionalFields = make(map[string]string)
		}
		additionalFields[questionField] = fieldText
	}

	return
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error) {
	rows, err := d.DB.Query(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering from potential_answer 
								left outer join localized_text as l1 on answer_localized_text_id=l1.app_text_id 
								left outer join answer_type on atype_id=answer_type.id 
								left outer join localized_text as l2 on answer_summary_text_id=l2.app_text_id
									where question_id = ? and (l1.language_id = ? or l1.ltext is null) and (l2.language_id = ? or l2.ltext is null)`, questionId, languageId, languageId)
	if err != nil {
		return
	}
	defer rows.Close()
	answerInfos = make([]PotentialAnswerInfo, 0)
	for rows.Next() {
		var id, ordering int64
		var answerType, answerTag string
		var answer, answerSummary sql.NullString
		err = rows.Scan(&id, &answer, &answerSummary, &answerType, &answerTag, &ordering)
		potentialAnswerInfo := PotentialAnswerInfo{}
		if answer.Valid {
			potentialAnswerInfo.Answer = answer.String
		}
		if answerSummary.Valid {
			potentialAnswerInfo.AnswerSummary = answerSummary.String
		}
		potentialAnswerInfo.PotentialAnswerId = id
		potentialAnswerInfo.AnswerTag = answerTag
		potentialAnswerInfo.Ordering = ordering
		potentialAnswerInfo.AnswerType = answerType
		answerInfos = append(answerInfos, potentialAnswerInfo)
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

func (d *DataService) GetActiveLayoutInfoForHealthCondition(healthConditionTag, role string) (bucket, key, region string, err error) {
	queryStr := fmt.Sprintf(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status='ACTIVE' and role = '%s' and health_condition.health_condition_tag = ?`, role)
	err = d.DB.QueryRow(queryStr, healthConditionTag).Scan(&bucket, &key, &region)
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

func (d *DataService) MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, comment string) (int64, error) {
	insertStr := fmt.Sprintf(`insert into layout_version (object_storage_id, syntax_version, health_condition_id,role, comment, status) 
							values (?, ?, ?, '%s', ?, 'CREATING')`, role)
	res, err := d.DB.Exec(insertStr, objectId, syntaxVersion, healthConditionId, comment)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into dr_layout_version (object_storage_id, layout_version_id, health_condition_id, status) 
							values (?, ?, ?, 'CREATING')`, objectId, layoutVersionId, healthConditionId)
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

func (d *DataService) UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error {
	tx, _ := d.DB.Begin()
	// update the current active layouts to DEPRECATED
	_, err := tx.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and role = 'PATIENT' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the current client active layouts to DEPRECATED
	_, err = tx.Exec(`update patient_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = tx.Exec(`update layout_version set status='ACTIVE' where id = ?`, layoutId)
	if err != nil {
		tx.Rollback()
		return err
	}

	updateStr := fmt.Sprintf(`update patient_layout_version set status='ACTIVE' where id in (%s)`, enumerateItemsIntoString(clientLayoutIds))
	_, err = tx.Exec(updateStr)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (d *DataService) UpdateDoctorActiveLayouts(layoutId int64, doctorLayoutId int64, healthConditionId int64) error {
	tx, _ := d.DB.Begin()
	// update the current active layouts to DEPRECATED
	_, err := tx.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and role = 'DOCTOR' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the current client active layouts to DEPRECATED
	_, err = tx.Exec(`update dr_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = tx.Exec(`update layout_version set status='ACTIVE' where id = ?`, layoutId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update dr_layout_version set status='ACTIVE' where id = ?`, doctorLayoutId)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func infoIdsFromMap(m map[int64]*common.PatientAnswer) []int64 {
	infoIds := make([]int64, 0)
	for key, _ := range m {
		infoIds = append(infoIds, key)
	}
	return infoIds
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
