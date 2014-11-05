package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]common.Answer, err error) {
	questionIdParams := nReplacements(len(questionIds))
	vals := appendInt64sToInterfaceSlice(nil, questionIds)
	vals = appendInt64sToInterfaceSlice(vals, questionIds)
	vals = append(vals, patientId, STATUS_ACTIVE)

	return d.getAnswersForQuestionsBasedOnQuery(`
		SELECT info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text,
			layout_version_id, parent_question_id, parent_info_intake_id 
		FROM info_intake  
		LEFT OUTER JOIN potential_answer ON potential_answer_id = potential_answer.id
		LEFT OUTER JOIN localized_text as l1 ON potential_answer.answer_localized_text_id = l1.app_text_id
		LEFT OUTER JOIN localized_text as l2 ON potential_answer.answer_summary_text_id = l2.app_text_id
		WHERE (info_intake.question_id IN (`+questionIdParams+`) OR parent_question_id IN (`+questionIdParams+`)) 
		AND patient_id = ? AND info_intake.status=?`, vals...)
}

func (d *DataService) AnswersForQuestions(questionIds []int64, info IntakeInfo) (answerIntakes map[int64][]common.Answer, err error) {
	questionIdParams := nReplacements(len(questionIds))
	vals := appendInt64sToInterfaceSlice(nil, questionIds)
	vals = appendInt64sToInterfaceSlice(vals, questionIds)
	vals = append(vals, info.Role().Value, info.Context().Value, STATUS_ACTIVE)

	return d.getAnswersForQuestionsBasedOnQuery(`
		SELECT i.id, i.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text,
			layout_version_id, parent_question_id, parent_info_intake_id 
		FROM `+info.TableName()+` as i  
		LEFT OUTER JOIN potential_answer ON potential_answer_id = potential_answer.id
		LEFT OUTER JOIN localized_text as l1 ON potential_answer.answer_localized_text_id = l1.app_text_id
		LEFT OUTER JOIN localized_text as l2 ON potential_answer.answer_summary_text_id = l2.app_text_id
		WHERE (i.question_id in (`+questionIdParams+`) OR parent_question_id in (`+questionIdParams+`)) 
		AND `+info.Role().Column+` = ? and `+info.Context().Column+` = ? and i.status=?`, vals...)
}

func (d *DataService) StoreAnswersForQuestion(info IntakeInfo) error {

	if len(info.Answers()) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.storeAnswers(tx, info); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) storeAnswers(tx *sql.Tx, info IntakeInfo) error {

	for questionId, answersToStore := range info.Answers() {
		// keep track of all question ids for which we are storing answers.
		questionIds := make(map[int64]bool)
		questionIds[questionId] = true

		infoIdToAnswersWithSubAnswers := make(map[int64]*common.AnswerIntake)
		subAnswersFound := false
		for _, answerToStore := range answersToStore {
			intakeID, err := insertAnswer(tx, info, answerToStore, STATUS_CREATING)
			if err != nil {
				return err
			}

			if answerToStore.SubAnswers != nil {
				subAnswersFound = true
				infoIdToAnswersWithSubAnswers[intakeID] = answerToStore
			}
		}

		// if there are no subanswers found, then we are pretty much done with the insertion of the
		// answers into the database.
		if !subAnswersFound {
			// ensure to update the status of any prior subquestions linked to the responses
			// of the top level questions that need to be inactivated, along with the answers
			// to the top level question itself.
			if err := d.updateSubAnswersWithStatus([]int64{questionId}, info, STATUS_INACTIVE, STATUS_ACTIVE, tx); err != nil {
				return err
			}

			if err := d.updateAnswersWithStatus([]int64{questionId}, info, STATUS_INACTIVE, STATUS_ACTIVE, tx); err != nil {
				return err
			}

			// if there are no subanswers to store, our job is done with just the top level answers
			if err := d.updateAnswersWithStatus([]int64{questionId}, info, STATUS_ACTIVE, STATUS_CREATING, tx); err != nil {
				return err
			}
			continue
		}

		// create a query to batch insert all subanswers
		for infoIntakeId, answerToStore := range infoIdToAnswersWithSubAnswers {
			if err := insertAnswersForSubQuestions(tx, info, answerToStore.SubAnswers,
				infoIntakeId, answerToStore.QuestionId.Int64(), STATUS_CREATING); err != nil {
				return err
			}

			// keep track of all questions for which we are storing answers
			for _, subAnswer := range answerToStore.SubAnswers {
				questionIds[subAnswer.QuestionId.Int64()] = true
			}
		}

		// deactivate all answers to top level questions as well as their sub-questions
		// as we make the new answers the most current 	up-to-date patient info intake
		if err := d.updateSubAnswersWithStatus([]int64{questionId}, info, STATUS_INACTIVE, STATUS_ACTIVE, tx); err != nil {
			return err
		}

		if err := d.updateAnswersWithStatus(createKeysArrayFromMap(questionIds),
			info, STATUS_INACTIVE, STATUS_ACTIVE, tx); err != nil {
			return err
		}

		// make all answers pertanining to the questionIds collected the new active set of answers for the
		// questions traversed
		if err := d.updateAnswersWithStatus(createKeysArrayFromMap(questionIds),
			info, STATUS_ACTIVE, STATUS_CREATING, tx); err != nil {
			return err
		}
	}

	return nil
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
			if err := d.claimMedia(tx, photoSlot.PhotoId, common.ClaimerTypePhotoIntakeSection, photoSectionId); err != nil {
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

func insertAnswer(tx *sql.Tx, info IntakeInfo, answerToStore *common.AnswerIntake, status string) (int64, error) {
	cols := []string{info.Role().Column, info.Context().Column, "question_id", "answer_text", "layout_version_id", "status", "potential_answer_id"}
	vals := []interface{}{info.Role().Value, info.Context().Value, answerToStore.QuestionId.Int64(), answerToStore.AnswerText, answerToStore.LayoutVersionId.Int64(), status}

	if answerToStore.PotentialAnswerId.Int64() > 0 {
		vals = append(vals, answerToStore.PotentialAnswerId.Int64())
	} else {
		vals = append(vals, nil)
	}

	res, err := tx.Exec(`
			INSERT INTO `+info.TableName()+` (`+strings.Join(cols, ",")+`)
			VALUES (`+nReplacements(len(vals))+`)`, vals...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func insertAnswersForSubQuestions(tx *sql.Tx, info IntakeInfo, answersToStore []*common.AnswerIntake, parentInfoIntakeId, parentQuestionId int64, status string) error {

	cols := []string{info.Role().Column, info.Context().Column, "parent_info_intake_id", "parent_question_id", "question_id", "answer_text", "layout_version_id", "status", "potential_answer_id"}
	rows := make([]string, len(answersToStore))
	valParams := `(` + nReplacements(len(cols)) + `)`
	var vals []interface{}
	for i, answerToStore := range answersToStore {
		vals = append(vals,
			info.Role().Value,
			info.Context().Value,
			parentInfoIntakeId,
			parentQuestionId,
			answerToStore.QuestionId.Int64(),
			answerToStore.AnswerText,
			answerToStore.LayoutVersionId.Int64(),
			status)
		if answerToStore.PotentialAnswerId.Int64() > 0 {
			vals = append(vals, answerToStore.PotentialAnswerId.Int64())
		} else {
			vals = append(vals, nil)
		}
		rows[i] = valParams
	}

	_, err := tx.Exec(`
		INSERT INTO `+info.TableName()+`
		(`+strings.Join(cols, ",")+`)
		VALUES `+strings.Join(rows, ","), vals...)
	return err
}

// This private helper method is to make it possible to update the status of sub answers
// only in combination with the top-level answer to the question. This method makes it possible
// to change the status of the entire set in an atomic fashion.
func (d *DataService) updateSubAnswersWithStatus(questionIds []int64, info IntakeInfo, status string, previousStatus string, tx *sql.Tx) (err error) {

	if len(questionIds) == 0 {
		return
	}

	vals := []interface{}{info.Role().Value}
	vals = appendInt64sToInterfaceSlice(vals, questionIds)
	vals = append(vals, info.Context().Value, previousStatus)

	rows, err := tx.Query(`
		SELECT id 
		FROM `+info.TableName()+`
		WHERE `+info.Role().Column+` = ?
		AND question_id IN (`+nReplacements(len(questionIds))+`) 
		AND `+info.Context().Column+` = ?
		AND status = ?`, vals...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var parentInfoIntakeIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		parentInfoIntakeIDs = append(parentInfoIntakeIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(parentInfoIntakeIDs) == 0 {
		return nil
	}

	vals = []interface{}{status}
	vals = appendInt64sToInterfaceSlice(vals, parentInfoIntakeIDs)
	_, err = tx.Exec(`
		UPDATE `+info.TableName()+` 
		SET status = ?
		WHERE parent_info_intake_id in (`+nReplacements(len(parentInfoIntakeIDs))+`)`, vals...)
	return err
}

func (d *DataService) updateAnswersWithStatus(questionIds []int64, info IntakeInfo, status string, previousStatus string, tx *sql.Tx) (err error) {

	if len(questionIds) == 0 {
		return nil
	}

	vals := []interface{}{status, info.Role().Value}
	vals = appendInt64sToInterfaceSlice(vals, questionIds)
	vals = append(vals, info.Context().Value, previousStatus)

	_, err = tx.Exec(`
		UPDATE `+info.TableName()+`
		SET status = ?
		WHERE `+info.Role().Column+` = ? 
		AND question_id in (`+nReplacements(len(questionIds))+`)
		AND `+info.Context().Column+` = ?
		AND status = ?
		`, vals...)
	return err
}

func (d *DataService) getAnswersForQuestionsBasedOnQuery(query string, args ...interface{}) (map[int64][]common.Answer, error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patientAnswers := make(map[int64][]common.Answer)
	queriedAnswers := make([]common.Answer, 0)
	for rows.Next() {
		var patientAnswerToQuestion common.AnswerIntake
		var answerText, answerSummaryText, potentialAnswer sql.NullString
		if err := rows.Scan(
			&patientAnswerToQuestion.AnswerIntakeId,
			&patientAnswerToQuestion.QuestionId,
			&patientAnswerToQuestion.PotentialAnswerId,
			&potentialAnswer,
			&answerSummaryText,
			&answerText,
			&patientAnswerToQuestion.LayoutVersionId,
			&patientAnswerToQuestion.ParentQuestionId,
			&patientAnswerToQuestion.ParentAnswerId); err != nil {
			return nil, err
		}

		patientAnswerToQuestion.PotentialAnswer = potentialAnswer.String
		patientAnswerToQuestion.AnswerText = answerText.String
		patientAnswerToQuestion.AnswerSummary = answerSummaryText.String
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
