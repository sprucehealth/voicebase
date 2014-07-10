package api

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]common.Answer, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, object_storage.bucket, object_storage.storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (info_intake.question_id in (%s) or parent_question_id in (%s)) and role_id = ? and info_intake.status='ACTIVE' and role='PATIENT'`, enumeratedStrings, enumeratedStrings)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId)
}

func (d *DataService) GetPatientAnswersForQuestions(questionIds []int64, patientId int64, patientVisitId int64) (answerIntakes map[int64][]common.Answer, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, bucket, storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (info_intake.question_id in (%s) or parent_question_id in (%s)) and role_id = ? and context_id = ? and info_intake.status='ACTIVE' and role='PATIENT'`, enumeratedStrings, enumeratedStrings)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId, patientVisitId)
}

func (d *DataService) GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds []int64, roleId int64, patientVisitId int64) (answerIntakes map[int64][]common.Answer, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, bucket, storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (info_intake.question_id in (%s) or parent_question_id in (%s)) and role_id = ? and context_id = ? and info_intake.status='ACTIVE' and role='DOCTOR'`, enumeratedStrings, enumeratedStrings)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, roleId, patientVisitId)
}

func (d *DataService) StoreAnswersForQuestion(role string, roleId, patientVisitId, layoutVersionId int64, answersToStorePerQuestion map[int64][]*common.AnswerIntake) error {

	if len(answersToStorePerQuestion) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for questionId, answersToStore := range answersToStorePerQuestion {
		// keep track of all question ids for which we are storing answers.
		questionIds := make(map[int64]bool)
		questionIds[questionId] = true

		infoIdToAnswersWithSubAnswers := make(map[int64]*common.AnswerIntake)
		subAnswersFound := false
		for _, answerToStore := range answersToStore {
			res, err := insertAnswers(tx, []*common.AnswerIntake{answerToStore}, STATUS_CREATING)
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
			d.updateSubAnswersToPatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, STATUS_INACTIVE, STATUS_ACTIVE, tx)
			d.updatePatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, STATUS_INACTIVE, STATUS_ACTIVE, tx)

			// if there are no subanswers to store, our job is done with just the top level answers
			d.updatePatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, STATUS_ACTIVE, STATUS_CREATING, tx)
			// tx.Commit()
			continue
		}

		// tx.Commit()
		// create a query to batch insert all subanswers
		for infoIntakeId, answerToStore := range infoIdToAnswersWithSubAnswers {
			_, err = insertAnswersForSubQuestions(tx, answerToStore.SubAnswers, strconv.FormatInt(infoIntakeId, 10), strconv.FormatInt(answerToStore.QuestionId.Int64(), 10), STATUS_CREATING)
			if err != nil {
				tx.Rollback()
				return err
			}
			// keep track of all questions for which we are storing answers
			for _, subAnswer := range answerToStore.SubAnswers {
				questionIds[subAnswer.QuestionId.Int64()] = true
			}
		}

		// deactivate all answers to top level questions as well as their sub-questions
		// as we make the new answers the most current 	up-to-date patient info intake
		err = d.updateSubAnswersToPatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
			patientVisitId, layoutVersionId, STATUS_INACTIVE, STATUS_ACTIVE, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}

		err = d.updatePatientInfoIntakesWithStatus(role, createKeysArrayFromMap(questionIds), roleId,
			patientVisitId, layoutVersionId, STATUS_INACTIVE, STATUS_ACTIVE, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}

		// make all answers pertanining to the questionIds collected the new active set of answers for the
		// questions traversed
		err = d.updatePatientInfoIntakesWithStatus(role, createKeysArrayFromMap(questionIds), roleId,
			patientVisitId, layoutVersionId, STATUS_ACTIVE, STATUS_CREATING, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) RejectPatientVisitPhotos(patientVisitId int64) error {
	_, err := d.db.Exec(`update info_intake 
		inner join question on info_intake.question_id = question.id 
		inner join question_type on question_type.id = question.qtype_id 
		set info_intake.status='REJECTED' 
			where info_intake.context_id = ? and qtype='q_type_photo' and status='ACTIVE'`, patientVisitId)
	return err
}

func (d *DataService) StorePhotoSectionsForQuestion(questionId, patientId, patientVisitId int64, photoSections []*common.PhotoIntakeSection) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// mark any preexisting photosections to this question as inactive
	_, err = tx.Exec(`update photo_intake_section set status=? 
		where question_id=? and patient_id=? and patient_visit_id=?`, STATUS_INACTIVE, questionId, patientId, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// iterate through the photo sections to create new ones
	for _, photoSection := range photoSections {
		res, err := tx.Exec(`insert into photo_intake_section (section_name, question_id, patient_id, patient_visit_id, status) values (?,?,?,?,?)`, photoSection.Name, questionId, patientId, patientVisitId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		photoSectionId, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, photoSlot := range photoSection.Photos {
			// claim the photo that was uploaded via the generic photo uploader
			if err := d.claimPhoto(tx, photoSlot.PhotoId, common.ClaimerTypePhotoIntakeSection, photoSectionId); err != nil {
				tx.Rollback()
				return err
			}

			_, err = tx.Exec(`insert into photo_intake_slot (photo_slot_id, photo_id, photo_slot_name, photo_intake_section_id) values (?,?,?,?)`, photoSlot.SlotId, photoSlot.PhotoId, photoSlot.Name, photoSectionId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) GetPatientCreatedPhotoSectionsForQuestionId(questionId, patientId, patientVisitId int64) ([]common.Answer, error) {
	photoSectionsByQuestion, err := d.GetPatientCreatedPhotoSectionsForQuestionIds([]int64{questionId}, patientId, patientVisitId)
	return photoSectionsByQuestion[questionId], err
}

func (d *DataService) GetPatientCreatedPhotoSectionsForQuestionIds(questionIds []int64, patientId, patientVisitId int64) (map[int64][]common.Answer, error) {
	if len(questionIds) == 0 {
		return nil, nil
	}
	photoSectionsByQuestion := make(map[int64][]common.Answer)
	params := []interface{}{patientId}
	params = appendInt64sToInterfaceSlice(params, questionIds)
	params = append(params, patientVisitId)
	params = append(params, STATUS_ACTIVE)

	rows, err := d.db.Query(fmt.Sprintf(`select id, question_id, section_name, creation_date 
		from photo_intake_section where patient_id=? and question_id in (%s) and patient_visit_id = ? and status=?`, nReplacements(len(questionIds))), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photoIntakeSection common.PhotoIntakeSection
		if err := rows.Scan(&photoIntakeSection.Id, &photoIntakeSection.QuestionId, &photoIntakeSection.Name, &photoIntakeSection.CreationDate); err != nil {
			return nil, err
		}

		// get photos associated with each section
		rows2, err := d.db.Query(`select id, photo_slot_id, photo_id, photo_slot_name, creation_date from photo_intake_slot where photo_intake_section_id = ?`, photoIntakeSection.Id)
		if err != nil {
			return nil, err
		}
		defer rows2.Close()

		photoIntakeSlots := make([]*common.PhotoIntakeSlot, 0)
		for rows2.Next() {
			var photoIntakeSlot common.PhotoIntakeSlot
			if err := rows2.Scan(&photoIntakeSlot.Id, &photoIntakeSlot.SlotId, &photoIntakeSlot.PhotoId, &photoIntakeSlot.Name, &photoIntakeSlot.CreationDate); err != nil {
				return nil, err
			}
			photoIntakeSlots = append(photoIntakeSlots, &photoIntakeSlot)
		}
		if rows2.Err() != nil {
			return nil, err
		}

		photoIntakeSection.Photos = photoIntakeSlots

		photoSections := photoSectionsByQuestion[photoIntakeSection.QuestionId]
		if len(photoSections) == 0 {
			photoSections = []common.Answer{&photoIntakeSection}
		} else {
			photoSections = append(photoSections, &photoIntakeSection)
		}
		photoSectionsByQuestion[photoIntakeSection.QuestionId] = photoSections
	}

	return photoSectionsByQuestion, rows.Err()
}

func insertAnswers(tx *sql.Tx, answersToStore []*common.AnswerIntake, status string) (res sql.Result, err error) {

	for _, answerToStore := range answersToStore {

		if answerToStore.PotentialAnswerId.Int64() == 0 {
			res, err = tx.Exec(`insert into info_intake (role_id, context_id, 
			question_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId.Int64(), answerToStore.ContextId.Int64(),
				answerToStore.QuestionId.Int64(), answerToStore.AnswerText, answerToStore.LayoutVersionId.Int64(), answerToStore.Role, status)
		} else {
			res, err = tx.Exec(`insert into info_intake (role_id, context_id,  
			question_id, potential_answer_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId.Int64(), answerToStore.ContextId.Int64(),
				answerToStore.QuestionId.Int64(), answerToStore.PotentialAnswerId.Int64(), answerToStore.AnswerText, answerToStore.LayoutVersionId.Int64(), answerToStore.Role, status)
		}

		if err != nil {
			return
		}
	}

	return
}

func insertAnswersForSubQuestions(tx *sql.Tx, answersToStore []*common.AnswerIntake, parentInfoIntakeId string, parentQuestionId string, status string) (res sql.Result, err error) {

	for _, answerToStore := range answersToStore {

		if answerToStore.PotentialAnswerId.Int64() == 0 {
			res, err = tx.Exec(`insert into info_intake (role_id, context_id, parent_info_intake_id, parent_question_id, 
			question_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId.Int64(), answerToStore.ContextId.Int64(), parentInfoIntakeId, parentQuestionId,
				answerToStore.QuestionId.Int64(), answerToStore.AnswerText, answerToStore.LayoutVersionId.Int64(), answerToStore.Role, status)
		} else {
			res, err = tx.Exec(`insert into info_intake (role_id, context_id, parent_info_intake_id, parent_question_id, 
			question_id, potential_answer_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId.Int64(), answerToStore.ContextId.Int64(), parentInfoIntakeId, parentQuestionId,
				answerToStore.QuestionId.Int64(), answerToStore.PotentialAnswerId.Int64(), answerToStore.AnswerText, answerToStore.LayoutVersionId.Int64(), answerToStore.Role, status)
		}

		if err != nil {
			return
		}
	}

	return
}
func (d *DataService) deleteAnswersWithId(role string, answerIds []int64) error {
	// delete all ids that were in CREATING state since they were committed in that state
	query := fmt.Sprintf("delete from info_intake where id in (%s) and role=?", enumerateItemsIntoString(answerIds))
	_, err := d.db.Exec(query, role)
	return err
}

// This private helper method is to make it possible to update the status of sub answers
// only in combination with the top-level answer to the question. This method makes it possible
// to change the status of the entire set in an atomic fashion.
func (d *DataService) updateSubAnswersToPatientInfoIntakesWithStatus(role string, questionIds []int64, roleId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {

	if len(questionIds) == 0 {
		return
	}

	parentInfoIntakeIds := make([]int64, 0)
	queryStr := fmt.Sprintf(`select id from info_intake where role_id = ? and question_id in (%s) and context_id = ? and layout_version_id = ? and status=? and role=?`, enumerateItemsIntoString(questionIds))
	rows, err := tx.Query(queryStr, roleId, patientVisitId, layoutVersionId, previousStatus, role)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		parentInfoIntakeIds = append(parentInfoIntakeIds, id)
	}
	if rows.Err() != nil {
		return rows.Err()
	}

	if len(parentInfoIntakeIds) == 0 {
		return
	}

	updateStr := fmt.Sprintf(`update info_intake set status=? 
						where parent_info_intake_id in (%s) and role=?`, enumerateItemsIntoString(parentInfoIntakeIds))
	_, err = tx.Exec(updateStr, status, role)
	return err
}

func (d *DataService) updatePatientInfoIntakesWithStatus(role string, questionIds []int64, roleId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {
	updateStr := fmt.Sprintf(`update info_intake set status=? 
						where role_id = ? and question_id in (%s)
						and context_id = ? and layout_version_id = ? and status=? and role=?`, enumerateItemsIntoString(questionIds))
	_, err = tx.Exec(updateStr, status, roleId, patientVisitId, layoutVersionId, previousStatus, role)
	return err
}

func (d *DataService) getPatientAnswersForQuestionsBasedOnQuery(query string, args ...interface{}) (map[int64][]common.Answer, error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patientAnswers := make(map[int64][]common.Answer)
	queriedAnswers := make([]common.Answer, 0)
	for rows.Next() {
		var patientAnswerToQuestion common.AnswerIntake
		var answerText, answerSummaryText, storageBucket, storageKey, storageRegion, potentialAnswer sql.NullString
		if err := rows.Scan(&patientAnswerToQuestion.AnswerIntakeId, &patientAnswerToQuestion.QuestionId, &patientAnswerToQuestion.PotentialAnswerId, &potentialAnswer,
			&answerSummaryText, &answerText, &storageBucket, &storageKey, &storageRegion, &patientAnswerToQuestion.LayoutVersionId, &patientAnswerToQuestion.ParentQuestionId, &patientAnswerToQuestion.ParentAnswerId); err != nil {
			return nil, err
		}

		patientAnswerToQuestion.PotentialAnswer = potentialAnswer.String
		patientAnswerToQuestion.AnswerText = answerText.String
		patientAnswerToQuestion.AnswerSummary = answerSummaryText.String
		patientAnswerToQuestion.StorageBucket = storageBucket.String
		patientAnswerToQuestion.StorageRegion = storageRegion.String
		patientAnswerToQuestion.StorageKey = storageKey.String

		queriedAnswers = append(queriedAnswers, &patientAnswerToQuestion)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// populate all top-level answers into the map
	patientAnswers = make(map[int64][]common.Answer)
	for _, patientAnswerToQuestion := range queriedAnswers {
		answer := patientAnswerToQuestion.(*common.AnswerIntake)
		if answer.ParentQuestionId.Int64() == 0 {
			questionId := answer.QuestionId.Int64()
			if patientAnswers[questionId] == nil {
				patientAnswers[questionId] = make([]common.Answer, 0)
			}
			patientAnswers[questionId] = append(patientAnswers[questionId], patientAnswerToQuestion)
		}
	}

	// add all subanswers to the top-level answers by iterating through the queried answers
	// to identify any sub answers
	for _, patientAnswerToQuestion := range queriedAnswers {
		answer := patientAnswerToQuestion.(*common.AnswerIntake)
		if answer.ParentQuestionId.Int64() != 0 {
			questionId := answer.ParentQuestionId.Int64()
			// go through the list of answers to identify the particular answer we care about
			for _, patientAnswer := range patientAnswers[questionId] {
				pAnswer := patientAnswer.(*common.AnswerIntake)
				if pAnswer.AnswerIntakeId.Int64() == answer.ParentAnswerId.Int64() {
					// this is the top level answer to
					if pAnswer.SubAnswers == nil {
						pAnswer.SubAnswers = make([]*common.AnswerIntake, 0)
					}
					pAnswer.SubAnswers = append(pAnswer.SubAnswers, answer)
				}
			}
		}
	}
	return patientAnswers, nil
}
