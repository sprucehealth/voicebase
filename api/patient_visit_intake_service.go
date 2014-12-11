package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) AnswersForQuestions(questionIDs []int64, info IntakeInfo) (answerIntakes map[int64][]common.Answer, err error) {

	if len(questionIDs) == 0 {
		return nil, nil
	}

	replacements := nReplacements(len(questionIDs))
	vals := appendInt64sToInterfaceSlice(nil, questionIDs)
	vals = appendInt64sToInterfaceSlice(vals, questionIDs)
	vals = append(vals, info.Role().Value, info.Context().Value)

	return d.getAnswersForQuestionsBasedOnQuery(`
		SELECT i.id, i.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text,
			layout_version_id, parent_question_id, parent_info_intake_id 
		FROM `+info.TableName()+` as i  
		LEFT OUTER JOIN potential_answer ON potential_answer_id = potential_answer.id
		LEFT OUTER JOIN localized_text as l1 ON potential_answer.answer_localized_text_id = l1.app_text_id
		LEFT OUTER JOIN localized_text as l2 ON potential_answer.answer_summary_text_id = l2.app_text_id
		WHERE (i.question_id in (`+replacements+`) OR parent_question_id in (`+replacements+`)) 
		AND `+info.Role().Column+` = ? and `+info.Context().Column+` = ?`, vals...)
}

func (d *DataService) PreviousPatientAnswersForQuestions(
	questionIDs []int64,
	patientID int64,
	beforeTime time.Time) (map[int64][]common.Answer, error) {

	if len(questionIDs) == 0 {
		return nil, nil
	}

	replacements := nReplacements(len(questionIDs))
	vals := appendInt64sToInterfaceSlice(nil, questionIDs)
	vals = appendInt64sToInterfaceSlice(vals, questionIDs)
	vals = append(vals, patientID, beforeTime, beforeTime)

	return d.getAnswersForQuestionsBasedOnQuery(`
		SELECT i.id, i.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text,
			layout_version_id, parent_question_id, parent_info_intake_id 
		FROM info_intake as i  
		LEFT OUTER JOIN potential_answer ON potential_answer_id = potential_answer.id
		LEFT OUTER JOIN localized_text as l1 ON potential_answer.answer_localized_text_id = l1.app_text_id
		LEFT OUTER JOIN localized_text as l2 ON potential_answer.answer_summary_text_id = l2.app_text_id
		WHERE (i.question_id in (`+replacements+`) OR parent_question_id in (`+replacements+`)) 
		AND patient_id = ? 
		AND answered_date < ?
		AND patient_visit_id = 
			(SELECT max(patient_visit_id) 
			 FROM info_intake i2
			 WHERE i2.answered_date < ? 
			 AND i2.patient_id = i.patient_id 
			 AND i2.question_id = i.question_id)`, vals...)
}

func (d *DataService) StoreAnswersForIntakes(intakes []IntakeInfo) error {
	if len(intakes) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.storeAnswers(tx, intakes); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) StorePhotoSectionsForQuestion(
	questionID,
	patientID,
	patientVisitID int64,
	sessionID string,
	sessionCounter uint,
	photoSections []*common.PhotoIntakeSection) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	incomingClock := &clientClock{
		sessionID:      sessionID,
		sessionCounter: sessionCounter,
	}

	clientClockStatement, err := tx.Prepare(`SELECT client_clock
		FROM photo_intake_section
		WHERE question_id = ?
		AND patient_visit_id = ?
		AND patient_id = ?
		LIMIT 1
		FOR UPDATE`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer clientClockStatement.Close()

	accept, err := acceptIncomingWrite(
		clientClockStatement,
		incomingClock,
		questionID, patientVisitID, patientID)
	if err != nil {
		tx.Rollback()
		return err
	} else if !accept {
		tx.Rollback()
		return nil
	}

	// delete any pre-existing photo intake sections
	_, err = tx.Exec(`
		DELETE FROM photo_intake_section 
		WHERE question_id = ? 
		AND patient_id = ? 
		AND patient_visit_id = ?`,
		questionID, patientID, patientVisitID)
	if err != nil {
		tx.Rollback()
		return err
	}

	photoIntakeSectionStatement, err := tx.Prepare(`
			INSERT INTO photo_intake_section
			(section_name, question_id, patient_id, patient_visit_id, client_clock)
			VALUES (?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer photoIntakeSectionStatement.Close()

	photoIntakeSlotStatement, err := tx.Prepare(`
		INSERT INTO photo_intake_slot
		(photo_slot_id, photo_id, photo_slot_name, photo_intake_section_id)
		VALUES (?,?,?,?)
		`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer photoIntakeSlotStatement.Close()

	// iterate through the photo sections to create new ones
	for _, photoSection := range photoSections {
		res, err := photoIntakeSectionStatement.Exec(
			photoSection.Name, questionID, patientID, patientVisitID, incomingClock.String())
		if err != nil {
			tx.Rollback()
			return err
		}

		photoSectionID, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, photoSlot := range photoSection.Photos {
			// claim the photo that was uploaded via the generic photo uploader
			if err := d.claimMedia(tx, photoSlot.PhotoID,
				common.ClaimerTypePhotoIntakeSection, photoSectionID); err != nil {
				tx.Rollback()
				return err
			}

			_, err = photoIntakeSlotStatement.Exec(
				photoSlot.SlotID, photoSlot.PhotoID, photoSlot.Name, photoSectionID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) PatientPhotoSectionsForQuestionIDs(
	questionIDs []int64,
	patientID,
	patientVisitID int64) (map[int64][]common.Answer, error) {
	if len(questionIDs) == 0 {
		return nil, nil
	}
	photoSectionsByQuestion := make(map[int64][]common.Answer)
	photoIntakeSections := make(map[int64]*common.PhotoIntakeSection)
	var photoIntakeSectionIDs []interface{}
	params := []interface{}{patientID}
	params = appendInt64sToInterfaceSlice(params, questionIDs)
	params = append(params, patientVisitID)

	rows, err := d.db.Query(`
		SELECT id, question_id, section_name, creation_date 
		FROM photo_intake_section 
		WHERE patient_id = ? 
		AND question_id in (`+nReplacements(len(questionIDs))+`) 
		AND patient_visit_id = ?`, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photoIntakeSection common.PhotoIntakeSection
		if err := rows.Scan(
			&photoIntakeSection.ID,
			&photoIntakeSection.QuestionID,
			&photoIntakeSection.Name,
			&photoIntakeSection.CreationDate); err != nil {
			return nil, err
		}
		photoSections := photoSectionsByQuestion[photoIntakeSection.QuestionID]
		photoSections = append(photoSections, &photoIntakeSection)
		photoSectionsByQuestion[photoIntakeSection.QuestionID] = photoSections

		photoIntakeSectionIDs = append(photoIntakeSectionIDs, photoIntakeSection.ID)
		photoIntakeSections[photoIntakeSection.ID] = &photoIntakeSection
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(photoIntakeSectionIDs) == 0 {
		return photoSectionsByQuestion, nil
	}

	// populate the photos associated with each of the photo sections
	rows, err = d.db.Query(`
		SELECT id, photo_slot_id, photo_intake_section_id, photo_id, photo_slot_name, creation_date 
		FROM photo_intake_slot 
		WHERE photo_intake_section_id IN (`+nReplacements(len(photoIntakeSectionIDs))+`)`, photoIntakeSectionIDs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photoIntakeSlot common.PhotoIntakeSlot
		var photoIntakeSectionID int64
		if err := rows.Scan(
			&photoIntakeSlot.ID,
			&photoIntakeSlot.SlotID,
			&photoIntakeSectionID,
			&photoIntakeSlot.PhotoID,
			&photoIntakeSlot.Name,
			&photoIntakeSlot.CreationDate); err != nil {
			return nil, err
		}
		photoIntakeSection := photoIntakeSections[photoIntakeSectionID]
		photoIntakeSection.Photos = append(photoIntakeSection.Photos, &photoIntakeSlot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return photoSectionsByQuestion, rows.Err()
}

func (d *DataService) storeAnswers(tx *sql.Tx, infos []IntakeInfo) error {

	// optimization: keep track of the different variations
	// of the prepared statements so as to ensure to create
	// each variant just once
	deleteStatements := make(map[string]*sql.Stmt)
	clientClockStatements := make(map[string]*sql.Stmt)
	insertStatements := make(map[string]*sql.Stmt)
	defer func() {
		// loop through each map and close any prepared statements
		for _, stmt := range deleteStatements {
			stmt.Close()
		}
		for _, stmt := range clientClockStatements {
			stmt.Close()
		}
		for _, stmt := range insertStatements {
			stmt.Close()
		}
	}()

	for _, info := range infos {
		key := info.TableName() + info.Context().Column + info.Role().Column
		var err error

		incomingClock := &clientClock{
			sessionID:      info.SessionID(),
			sessionCounter: info.SessionCounter(),
		}
		clockValue := incomingClock.String()

		deleteStatement, ok := deleteStatements[key]
		if !ok {
			deleteStatement, err = tx.Prepare(`
			DELETE FROM ` + info.TableName() + `
			WHERE ` + info.Context().Column + ` = ?
			AND ` + info.Role().Column + ` = ?` + `
			AND question_id = ?`)
			if err != nil {
				return err
			}
			deleteStatements[key] = deleteStatement
		}

		clientClockStatement, ok := clientClockStatements[key]
		if !ok {
			clientClockStatement, err = tx.Prepare(`SELECT client_clock
			FROM ` + info.TableName() + `
			WHERE question_id = ?
			AND ` + info.Context().Column + ` = ?
			AND ` + info.Role().Column + ` = ?
			LIMIT 1
			FOR UPDATE`)
			if err != nil {
				return err
			}
			clientClockStatements[key] = clientClockStatement
		}

		insertAnswerStatement, ok := insertStatements[key]
		if !ok {
			cols := []string{
				info.Role().Column,
				info.Context().Column,
				"question_id",
				"answer_text",
				"layout_version_id",
				"client_clock",
				"potential_answer_id"}

			insertAnswerStatement, err = tx.Prepare(`
			INSERT INTO ` + info.TableName() + ` (` + strings.Join(cols, ",") + `)
			VALUES (` + nReplacements(len(cols)) + `)`)
			if err != nil {
				return err
			}
			insertStatements[key] = insertAnswerStatement
		}

		for questionID, answersToStore := range info.Answers() {

			accept, err := acceptIncomingWrite(
				clientClockStatement,
				incomingClock,
				questionID,
				info.Context().Value,
				info.Role().Value)
			if err != nil {
				return err
			} else if !accept {
				continue
			}

			// delete existing answers for the question
			_, err = deleteStatement.Exec(
				info.Context().Value,
				info.Role().Value,
				questionID)
			if err != nil {
				return err
			}

			infoIntakeIDs := make(map[int64]*common.AnswerIntake)
			for _, answerToStore := range answersToStore {
				infoIntakeID, err := insertAnswer(
					insertAnswerStatement,
					info,
					answerToStore,
					clockValue)
				if err != nil {
					return err
				}

				if answerToStore.SubAnswers != nil {
					infoIntakeIDs[infoIntakeID] = answerToStore
				}
			}

			// create a query to batch insert all subanswers
			for infoIntakeID, answerToStore := range infoIntakeIDs {
				if err := insertAnswersForSubQuestions(
					tx,
					info,
					answerToStore.SubAnswers,
					infoIntakeID,
					answerToStore.QuestionID.Int64()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// acceptIncomingWrite determines whether or not to accept
// the incoming write based on the existing clock value for the answer
// if one doesn't exist then the write is accepted, else
// existing clock value is compared to the incoming clock value
func acceptIncomingWrite(
	stmt *sql.Stmt,
	incomingClockValue *clientClock,
	params ...interface{}) (bool, error) {

	var existingClockValue clientClock
	err := stmt.QueryRow(params...).Scan(&existingClockValue)
	if err != sql.ErrNoRows && err != nil {
		return false, err
	}

	return existingClockValue.lessThan(incomingClockValue), nil
}

func insertAnswer(stmt *sql.Stmt, info IntakeInfo, answerToStore *common.AnswerIntake, clientClock string) (int64, error) {

	vals := []interface{}{
		info.Role().Value,
		info.Context().Value,
		answerToStore.QuestionID.Int64(),
		answerToStore.AnswerText,
		answerToStore.LayoutVersionID.Int64(),
		clientClock}

	if answerToStore.PotentialAnswerID.Int64() > 0 {
		vals = append(vals, answerToStore.PotentialAnswerID.Int64())
	} else {
		vals = append(vals, nil)
	}

	res, err := stmt.Exec(vals...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func insertAnswersForSubQuestions(
	tx *sql.Tx,
	info IntakeInfo,
	answersToStore []*common.AnswerIntake,
	parentInfoIntakeId, parentQuestionId int64) error {

	if len(answersToStore) == 0 {
		return nil
	}

	cols := []string{
		info.Role().Column, info.Context().Column,
		"parent_info_intake_id", "parent_question_id", "question_id",
		"answer_text", "layout_version_id", "potential_answer_id"}
	rows := make([]string, len(answersToStore))
	valParams := `(` + nReplacements(len(cols)) + `)`
	var vals []interface{}
	for i, answerToStore := range answersToStore {
		vals = append(vals,
			info.Role().Value,
			info.Context().Value,
			parentInfoIntakeId,
			parentQuestionId,
			answerToStore.QuestionID.Int64(),
			answerToStore.AnswerText,
			answerToStore.LayoutVersionID.Int64())
		if answerToStore.PotentialAnswerID.Int64() > 0 {
			vals = append(vals, answerToStore.PotentialAnswerID.Int64())
		} else {
			vals = append(vals, nil)
		}
		rows[i] = valParams
	}

	// unfortunately cannot create a prepared statement out of this query
	// due to the varied number of inserts, however, in the case of multiple
	// new inserts considered it faster to batch insert versus have a prepared
	// statement that loops through inserts
	_, err := tx.Exec(`
		INSERT INTO `+info.TableName()+`
		(`+strings.Join(cols, ",")+`)
		VALUES `+strings.Join(rows, ","), vals...)
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
			&patientAnswerToQuestion.AnswerIntakeID,
			&patientAnswerToQuestion.QuestionID,
			&patientAnswerToQuestion.PotentialAnswerID,
			&potentialAnswer,
			&answerSummaryText,
			&answerText,
			&patientAnswerToQuestion.LayoutVersionID,
			&patientAnswerToQuestion.ParentQuestionId,
			&patientAnswerToQuestion.ParentAnswerId); err != nil {
			return nil, err
		}

		patientAnswerToQuestion.PotentialAnswer = potentialAnswer.String
		patientAnswerToQuestion.AnswerText = answerText.String
		patientAnswerToQuestion.AnswerSummary = answerSummaryText.String
		queriedAnswers = append(queriedAnswers, &patientAnswerToQuestion)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// populate all top-level answers into the map
	patientAnswers = make(map[int64][]common.Answer)
	for _, queriedAnswer := range queriedAnswers {
		answer := queriedAnswer.(*common.AnswerIntake)
		if answer.ParentQuestionId.Int64() == 0 {
			questionID := answer.QuestionID.Int64()
			patientAnswers[questionID] = append(patientAnswers[questionID], queriedAnswer)
		}
	}

	// add all subanswers to the top-level answers by iterating through the queried answers
	// to identify any sub answers
	for _, queriedAnswer := range queriedAnswers {
		answer := queriedAnswer.(*common.AnswerIntake)
		if answer.ParentQuestionId.Int64() != 0 {
			questionID := answer.ParentQuestionId.Int64()
			// go through the list of answers to identify the particular answer we care about
			for _, patientAnswer := range patientAnswers[questionID] {
				pAnswer := patientAnswer.(*common.AnswerIntake)
				if pAnswer.AnswerIntakeID.Int64() == answer.ParentAnswerId.Int64() {
					pAnswer.SubAnswers = append(pAnswer.SubAnswers, answer)
				}
			}
		}
	}
	return patientAnswers, nil
}
