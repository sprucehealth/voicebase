package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sprucehealth/backend/info_intake"
)

func (d *DataService) GetQuestionType(questionId int64) (string, error) {
	var questionType string
	err := d.db.QueryRow(`select qtype from question
						inner join question_type on question_type.id = qtype_id
						where question.id = ?`, questionId).Scan(&questionType)
	return questionType, err
}

func (d *DataService) GetActiveLayoutForHealthCondition(healthConditionTag, role, purpose string) ([]byte, error) {
	var layoutBlob []byte
	err := d.db.QueryRow(`select layout from layout_version 
								inner join layout_blob_storage on layout_blob_storage_id = layout_blob_storage.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status=? and role = ? and layout_purpose = ? 
									and health_condition.health_condition_tag = ?`, STATUS_ACTIVE, role, purpose, healthConditionTag).Scan(&layoutBlob)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	}
	return layoutBlob, err
}
func (d *DataService) GetCurrentActivePatientLayout(languageId, healthConditionId int64) ([]byte, int64, error) {
	var layoutBlob []byte
	var layoutVersionId int64
	row := d.db.QueryRow(`select layout, layout_version_id from patient_layout_version 
							inner join layout_blob_storage on layout_blob_storage_id=layout_blob_storage.id 
								where patient_layout_version.status=? and health_condition_id = ? and language_id = ?`, STATUS_ACTIVE, healthConditionId, languageId)
	err := row.Scan(&layoutBlob, &layoutVersionId)
	return layoutBlob, layoutVersionId, err
}

func (d *DataService) GetCurrentActiveDoctorLayout(healthConditionId int64) ([]byte, int64, error) {
	return d.getActiveDoctorLayoutForPurpose(healthConditionId, REVIEW_PURPOSE)
}
func (d *DataService) GetActiveDoctorDiagnosisLayout(healthConditionId int64) ([]byte, int64, error) {
	return d.getActiveDoctorLayoutForPurpose(healthConditionId, DIAGNOSE_PURPOSE)
}

func (d *DataService) GetLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (int64, error) {
	var layoutVersionId int64
	err := d.db.QueryRow(`select layout_version_id from dr_layout_version 
					inner join layout_version on layout_version_id=layout_version.id 
						where dr_layout_version.status = ? and layout_purpose = ? and role = ? and dr_layout_version.health_condition_id = ?`, STATUS_ACTIVE, DIAGNOSE_PURPOSE, DOCTOR_ROLE, healthConditionId).Scan(&layoutVersionId)
	return layoutVersionId, err

}

func (d *DataService) getActiveDoctorLayoutForPurpose(healthConditionId int64, purpose string) ([]byte, int64, error) {
	var layoutBlob []byte
	var layoutVersionId int64
	row := d.db.QueryRow(`select layout, layout_version_id from dr_layout_version
							inner join layout_version on layout_version_id=layout_version.id 
							inner join layout_blob_storage on dr_layout_version.layout_blob_storage_id=layout_blob_storage.id 
								where dr_layout_version.status=? and 
								layout_purpose=? and role = ? and dr_layout_version.health_condition_id = ?`, STATUS_ACTIVE, purpose, DOCTOR_ROLE, healthConditionId)
	err := row.Scan(&layoutBlob, &layoutVersionId)
	return layoutBlob, layoutVersionId, err
}

func (d *DataService) GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error) {
	err = d.db.QueryRow("select layout_version_id from patient_visit where id = ?", patientVisitId).Scan(&layoutVersionId)
	return
}

func (d *DataService) GetPatientLayout(layoutVersionId, languageId int64) ([]byte, error) {
	var layoutBlob []byte
	err := d.db.QueryRow(`select layout from patient_layout_version 
							inner join layout_blob_storage on layout_blob_storage_id=layout_blob_storage.id 
								where layout_version_id = ? and language_id = ?`, layoutVersionId, languageId).Scan(&layoutBlob)
	return layoutBlob, err
}

func (d *DataService) CreateLayoutVersion(layout []byte, syntaxVersion, healthConditionId int64, role, purpose, comment string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	insertId, err := tx.Exec(`insert into layout_blob_storage (layout) values (?)`, layout)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	layoutBlobStorageId, err := insertId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	res, err := tx.Exec(`insert into layout_version (layout_blob_storage_id, syntax_version, health_condition_id,role, layout_purpose, comment, status) 
							values (?, ?, ?, ?, ?, ?, ?)`, layoutBlobStorageId, syntaxVersion, healthConditionId, role, purpose, comment, STATUS_CREATING)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	layoutVersionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return 0, err
	}

	return layoutVersionId, nil
}

func (d *DataService) CreatePatientLayout(layout []byte, languageId, layoutVersionId, healthConditionId int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	insertId, err := tx.Exec(`insert into layout_blob_storage (layout) values (?)`, layout)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	layoutBlobStorageId, err := insertId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	res, err := tx.Exec(`insert into patient_layout_version (layout_blob_storage_id, language_id, layout_version_id, health_condition_id, status) 
								values (?, ?, ?, ?, ?)`, layoutBlobStorageId, languageId, layoutVersionId, healthConditionId, STATUS_CREATING)

	if err != nil {
		tx.Rollback()
		return 0, err
	}

	patientLayoutVersionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return 0, err
	}

	return patientLayoutVersionId, nil
}

func (d *DataService) CreateDoctorLayout(layout []byte, layoutVersionId, healthConditionId int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, nil
	}

	lastInsertId, err := tx.Exec(`insert into layout_blob_storage (layout) values (?)`, layout)
	if err != nil {
		tx.Rollback()
		return 0, nil
	}

	layoutBlobStorageId, err := lastInsertId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, nil
	}

	res, err := tx.Exec(`insert into dr_layout_version (layout_blob_storage_id, layout_version_id, health_condition_id, status) 
							values (?, ?, ?, 'CREATING')`, layoutBlobStorageId, layoutVersionId, healthConditionId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	drLayoutVersionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, nil
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return 0, nil
	}

	return drLayoutVersionId, nil
}

func (d *DataService) UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error {
	tx, _ := d.db.Begin()
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
	_, err = tx.Exec(`update layout_version set status=? where id = ?`, STATUS_ACTIVE, layoutId)
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

	return tx.Commit()
}

func (d *DataService) UpdateDoctorActiveLayouts(layoutId int64, doctorLayoutId int64, healthConditionId int64, purpose string) error {
	tx, _ := d.db.Begin()

	// update the current client active layouts to DEPRECATED
	_, err := tx.Exec(`update dr_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ? and layout_version_id in (select id from layout_version where role = 'DOCTOR' and layout_purpose = ?)`, healthConditionId, purpose)
	if err == nil {
		// update the current active layouts to DEPRECATED
		_, err = tx.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and role = 'DOCTOR' and layout_purpose = ? and health_condition_id = ?`, purpose, healthConditionId)
	}
	if err == nil {
		// update the new layout as ACTIVE
		_, err = tx.Exec(`update layout_version set status=? where id = ?`, STATUS_ACTIVE, layoutId)
	}
	if err == nil {
		_, err = tx.Exec(`update dr_layout_version set status=? where id = ?`, STATUS_ACTIVE, doctorLayoutId)
	}
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetGlobalSectionIds() ([]int64, error) {
	rows, err := d.db.Query(`select id from section where health_condition_id is null`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	globalSectionIds := make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		if err := rows.Scan(&sectionId); err != nil {
			return nil, err
		}
		globalSectionIds = append(globalSectionIds, sectionId)
	}
	return globalSectionIds, rows.Err()
}

func (d *DataService) GetSectionIdsForHealthCondition(healthConditionId int64) ([]int64, error) {
	rows, err := d.db.Query(`select id from section where health_condition_id = ?`, healthConditionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionIds := make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		if err := rows.Scan(&sectionId); err != nil {
			return nil, err
		}
		sectionIds = append(sectionIds, sectionId)
	}
	return sectionIds, rows.Err()
}

func (d *DataService) GetHealthConditionInfo(healthConditionTag string) (int64, error) {
	var id int64
	err := d.db.QueryRow("select id from health_condition where comment = ? ", healthConditionTag).Scan(&id)
	return id, err
}

func (d *DataService) GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error) {
	err = d.db.QueryRow(`select section.id, ltext from section 
					inner join app_text on section_title_app_text_id = app_text.id 
					inner join localized_text on app_text_id = app_text.id 
						where language_id = ? and section_tag = ?`, languageId, sectionTag).Scan(&id, &title)
	if err == sql.ErrNoRows {
		err = NoRowsError
	}
	return
}

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (*info_intake.Question, error) {
	questionInfos, err := d.GetQuestionInfoForTags([]string{questionTag}, languageId)
	if err != nil {
		return nil, err
	}
	if len(questionInfos) > 0 {
		return questionInfos[0], nil
	}
	return nil, NoRowsError
}

func (d *DataService) GetQuestionInfoForTags(questionTags []string, languageId int64) ([]*info_intake.Question, error) {

	params := make([]interface{}, 0)
	params = appendStringsToInterfaceSlice(params, questionTags)
	params = append(params, languageId)
	params = append(params, languageId)
	params = append(params, languageId)

	rows, err := d.db.Query(fmt.Sprintf(
		`select question.question_tag, question.id, l1.ltext, qtext_has_tokens, qtype, parent_question_id, l2.ltext, l3.ltext, formatted_field_tags, required, to_alert, l4.ltext from question 
			left outer join localized_text as l1 on l1.app_text_id=qtext_app_text_id
			left outer join question_type on qtype_id=question_type.id
			left outer join localized_text as l2 on qtext_short_text_id = l2.app_text_id
			left outer join localized_text as l3 on subtext_app_text_id = l3.app_text_id
			left outer join localized_text as l4 on alert_app_text_id = l4.app_text_id
				where question_tag in (%s) and (l1.ltext is NULL or l1.language_id = ?) and (l3.ltext is NULL or l3.language_id=?)
				and (l4.ltext is NULL or l4.language_id=?)`, nReplacements(len(questionTags))), params...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	questionInfos, err := d.getQuestionInfoFromRows(rows, languageId)

	return questionInfos, err
}

func (d *DataService) getQuestionInfoFromRows(rows *sql.Rows, languageId int64) ([]*info_intake.Question, error) {

	questionInfos := make([]*info_intake.Question, 0)
	for rows.Next() {
		var id int64
		var questionTag string
		var questionTitle, questionType, questionSummary, questionSubText, formattedFieldTagsNull, alertText sql.NullString
		var nullParentQuestionId sql.NullInt64
		var requiredBit, toAlertBit, titleHasTokens sql.NullBool

		err := rows.Scan(
			&questionTag,
			&id,
			&questionTitle,
			&titleHasTokens,
			&questionType,
			&nullParentQuestionId,
			&questionSummary,
			&questionSubText,
			&formattedFieldTagsNull,
			&requiredBit,
			&toAlertBit,
			&alertText,
		)

		if err != nil {
			return nil, err
		}

		questionInfo := &info_intake.Question{
			QuestionId:             id,
			ParentQuestionId:       nullParentQuestionId.Int64,
			QuestionTag:            questionTag,
			QuestionTitle:          questionTitle.String,
			QuestionTitleHasTokens: titleHasTokens.Bool,
			QuestionType:           questionType.String,
			QuestionSummary:        questionSummary.String,
			QuestionSubText:        questionSubText.String,
			Required:               requiredBit.Bool,
			ToAlert:                toAlertBit.Bool,
			AlertFormattedText:     alertText.String,
		}
		if formattedFieldTagsNull.Valid && formattedFieldTagsNull.String != "" {
			questionInfo.FormattedFieldTags = []string{formattedFieldTagsNull.String}
		}

		// get any additional fields pertaining to the question from the database
		rows, err := d.db.Query(`select question_field, ltext from question_fields
								inner join localized_text on question_fields.app_text_id = localized_text.app_text_id
								where question_id = ? and language_id = ?`, questionInfo.QuestionId, languageId)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var questionField, fieldText string
			err = rows.Scan(&questionField, &fieldText)
			if err != nil {
				return nil, err
			}
			if questionInfo.AdditionalFields == nil {
				questionInfo.AdditionalFields = make(map[string]interface{})
			}
			questionInfo.AdditionalFields[questionField] = fieldText
		}
		if rows.Err() != nil {
			return nil, rows.Err()
		}

		// get any extra fields defined as json, after ensuring that json is valid (by unmarshaling)
		var jsonBytes []byte

		err = d.db.QueryRow(`select json from extra_question_fields where question_id = ?`, questionInfo.QuestionId).Scan(&jsonBytes)
		if err != sql.ErrNoRows {
			if err != nil {
				return nil, err
			}

			var extraJSON map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &extraJSON); err != nil {
				return nil, err
			}

			if questionInfo.AdditionalFields == nil {
				questionInfo.AdditionalFields = make(map[string]interface{})
			}
			// combine the extra fields with the other question fields
			for key, value := range extraJSON {
				questionInfo.AdditionalFields[key] = value
			}
		}

		questionInfos = append(questionInfos, questionInfo)
	}

	return questionInfos, rows.Err()
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) ([]*info_intake.PotentialAnswer, error) {
	rows, err := d.db.Query(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering, to_alert from potential_answer 
								left outer join localized_text as l1 on answer_localized_text_id=l1.app_text_id 
								left outer join answer_type on atype_id=answer_type.id 
								left outer join localized_text as l2 on answer_summary_text_id=l2.app_text_id
									where question_id = ? and (l1.language_id = ? or l1.ltext is null) and (l2.language_id = ? or l2.ltext is null) and status='ACTIVE'`, questionId, languageId, languageId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return createAnswerInfosFromRows(rows)
}

func createAnswerInfosFromRows(rows *sql.Rows) ([]*info_intake.PotentialAnswer, error) {
	answerInfos := make([]*info_intake.PotentialAnswer, 0)
	for rows.Next() {
		var id, ordering int64
		var answerType, answerTag string
		var answer, answerSummary sql.NullString
		var toAlert sql.NullBool
		err := rows.Scan(&id, &answer, &answerSummary, &answerType, &answerTag, &ordering, &toAlert)
		potentialAnswerInfo := &info_intake.PotentialAnswer{
			Answer:        answer.String,
			AnswerSummary: answerSummary.String,
			AnswerId:      id,
			AnswerTag:     answerTag,
			Ordering:      ordering,
			AnswerType:    answerType,
			ToAlert:       toAlert.Bool,
		}
		answerInfos = append(answerInfos, potentialAnswerInfo)
		if err != nil {
			return answerInfos, err
		}
	}
	return answerInfos, rows.Err()
}

func (d *DataService) GetAnswerInfoForTags(answerTags []string, languageId int64) ([]*info_intake.PotentialAnswer, error) {

	params := make([]interface{}, 0)
	params = appendStringsToInterfaceSlice(params, answerTags)
	params = append(params, languageId)
	params = append(params, languageId)
	rows, err := d.db.Query(fmt.Sprintf(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering, to_alert from potential_answer 
								left outer join localized_text as l1 on answer_localized_text_id=l1.app_text_id 
								left outer join answer_type on atype_id=answer_type.id 
								left outer join localized_text as l2 on answer_summary_text_id=l2.app_text_id
									where potential_answer_tag in (%s) and (l1.language_id = ? or l1.ltext is null) and (l2.language_id = ? or l2.ltext is null) and status='ACTIVE'`, nReplacements(len(answerTags))), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerInfos, err := createAnswerInfosFromRows(rows)
	if err != nil {
		return nil, err
	}

	// create a mapping so that we can send back the items in the same order as the tags
	answerInfoMapping := make(map[string]*info_intake.PotentialAnswer)
	for _, answerInfoItem := range answerInfos {
		answerInfoMapping[answerInfoItem.AnswerTag] = answerInfoItem
	}

	// order the items based on the ordering of the tags
	answerInfoInOrder := make([]*info_intake.PotentialAnswer, len(answerInfos))
	for i, answerTag := range answerTags {
		answerInfoInOrder[i] = answerInfoMapping[answerTag]
	}

	return answerInfoInOrder, nil
}

func (d *DataService) GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error) {
	err = d.db.QueryRow(`select tips_section.id, ltext1.ltext, ltext2.ltext from tips_section 
								inner join localized_text as ltext1 on tips_title_text_id=ltext1.app_text_id 
								inner join localized_text as ltext2 on tips_subtext_text_id=ltext2.app_text_id 
									where ltext1.language_id = ? and tips_section_tag = ?`, languageId, tipSectionTag).Scan(&id, &tipSectionTitle, &tipSectionSubtext)
	return
}

func (d *DataService) GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error) {
	err = d.db.QueryRow(`select tips.id, ltext from tips
								inner join localized_text on app_text_id=tips_text_id 
									where tips_tag = ? and language_id = ?`, tipTag, languageId).Scan(&id, &tip)
	return
}

func (d *DataService) GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error) {
	rows, err := d.db.Query(`select id,language from languages_supported`)
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
	return languagesSupported, languagesSupportedIds, rows.Err()
}

func (d *DataService) GetPhotoSlots(questionId, languageId int64) ([]*info_intake.PhotoSlot, error) {
	rows, err := d.db.Query(`select photo_slot.id, ltext, slot_type, required from photo_slot
		inner join localized_text on app_text_id = slot_name_app_text_id
		inner join photo_slot_type on photo_slot_type.id = slot_type_id
		where question_id=? and language_id = ? order by ordering`, questionId, languageId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photoSlotInfoList := make([]*info_intake.PhotoSlot, 0)
	for rows.Next() {
		var pSlotInfo info_intake.PhotoSlot
		if err := rows.Scan(&pSlotInfo.Id, &pSlotInfo.Name, &pSlotInfo.Type, &pSlotInfo.Required); err != nil {
			return nil, err
		}
		photoSlotInfoList = append(photoSlotInfoList, &pSlotInfo)
	}
	return photoSlotInfoList, rows.Err()
}
