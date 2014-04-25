package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"log"
)

func (d *DataService) GetQuestionType(questionId int64) (string, error) {
	var questionType string
	err := d.DB.QueryRow(`select qtype from question
						inner join question_type on question_type.id = qtype_id
						where question.id = ?`, questionId).Scan(&questionType)
	return questionType, err
}

func (d *DataService) GetActiveLayoutInfoForHealthCondition(healthConditionTag, role, purpose string) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status='ACTIVE' and role = ? and layout_purpose = ? and health_condition.health_condition_tag = ?`, role, purpose, healthConditionTag).Scan(&bucket, &key, &region)
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
							inner join layout_version on layout_version_id=layout_version.id 
							inner join object_storage on dr_layout_version.object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where dr_layout_version.status='ACTIVE' and layout_purpose='REVIEW' and role='DOCTOR' and dr_layout_version.health_condition_id = ?`, healthConditionId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}
func (d *DataService) GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from dr_layout_version
							inner join layout_version on layout_version_id=layout_version.id 
							inner join object_storage on dr_layout_version.object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where dr_layout_version.status='ACTIVE' and 
								layout_purpose='DIAGNOSE' and role = 'DOCTOR' and dr_layout_version.health_condition_id = ?`, healthConditionId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error) {
	err = d.DB.QueryRow("select layout_version_id from patient_visit where id = ?", patientVisitId).Scan(&layoutVersionId)
	return
}

func (d *DataService) GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where layout_version_id = ? and language_id = ?`, layoutVersionId, languageId).Scan(&bucket, &key, &region)
	return
}

func (d *DataService) MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, purpose, comment string) (int64, error) {
	res, err := d.DB.Exec(`insert into layout_version (object_storage_id, syntax_version, health_condition_id,role, layout_purpose, comment, status) 
							values (?, ?, ?, ?, ?, ?, 'CREATING')`, objectId, syntaxVersion, healthConditionId, role, purpose, comment)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (d *DataService) MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_layout_version (object_storage_id, language_id, layout_version_id, health_condition_id, status) 
								values (?, ?, ?, ?, 'CREATING')`, objectId, languageId, layoutVersionId, healthConditionId)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
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

func (d *DataService) MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into dr_layout_version (object_storage_id, layout_version_id, health_condition_id, status) 
							values (?, ?, ?, 'CREATING')`, objectId, layoutVersionId, healthConditionId)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (d *DataService) UpdateDoctorActiveLayouts(layoutId int64, doctorLayoutId int64, healthConditionId int64, purpose string) error {
	tx, _ := d.DB.Begin()

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
	rows, err := d.DB.Query(`select id from section where health_condition_id is null`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	globalSectionIds := make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		globalSectionIds = append(globalSectionIds, sectionId)
	}
	return globalSectionIds, rows.Err()
}

func (d *DataService) GetSectionIdsForHealthCondition(healthConditionId int64) ([]int64, error) {
	rows, err := d.DB.Query(`select id from section where health_condition_id = ?`, healthConditionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionIds := make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		sectionIds = append(sectionIds, sectionId)
	}
	return sectionIds, rows.Err()
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

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (*common.QuestionInfo, error) {
	questionInfos, err := d.GetQuestionInfoForTags([]string{questionTag}, languageId)
	if len(questionInfos) > 0 {
		return questionInfos[0], err
	}
	return nil, err
}

func (d *DataService) GetQuestionInfoForTags(questionTags []string, languageId int64) ([]*common.QuestionInfo, error) {

	params := make([]interface{}, 0)
	params = appendStringsToInterfaceSlice(params, questionTags)
	params = append(params, languageId)
	params = append(params, languageId)
	params = append(params, languageId)

	rows, err := d.DB.Query(fmt.Sprintf(
		`select question.question_tag, question.id, l1.ltext, qtype, parent_question_id, l2.ltext, l3.ltext, formatted_field_tags, required, to_alert, l4.ltext from question 
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

func (d *DataService) getQuestionInfoFromRows(rows *sql.Rows, languageId int64) ([]*common.QuestionInfo, error) {

	questionInfos := make([]*common.QuestionInfo, 0)
	for rows.Next() {
		var id int64
		var questionTag string
		var questionTitle, questionType, questionSummary, questionSubText, formattedFieldTagsNull, alertText sql.NullString
		var nullParentQuestionId sql.NullInt64
		var requiredBit, toAlertBit sql.NullBool

		err := rows.Scan(
			&questionTag,
			&id,
			&questionTitle,
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

		questionInfo := &common.QuestionInfo{
			Id:                 id,
			ParentQuestionId:   nullParentQuestionId.Int64,
			QuestionTag:        questionTag,
			Title:              questionTitle.String,
			Type:               questionType.String,
			Summary:            questionSummary.String,
			SubText:            questionSubText.String,
			FormattedFieldTags: formattedFieldTagsNull.String,
			Required:           requiredBit.Valid && requiredBit.Bool,
			ToAlert:            toAlertBit.Valid && toAlertBit.Bool,
			AlertFormattedText: alertText.String,
		}

		// get any additional fields pertaining to the question from the database
		rows, err := d.DB.Query(`select question_field, ltext from question_fields
								inner join localized_text on question_fields.app_text_id = localized_text.app_text_id
								where question_id = ? and language_id = ?`, questionInfo.Id, languageId)
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
				questionInfo.AdditionalFields = make(map[string]string)
			}
			questionInfo.AdditionalFields[questionField] = fieldText
		}
		if rows.Err() != nil {
			return nil, rows.Err()
		}
		questionInfos = append(questionInfos, questionInfo)
	}

	return questionInfos, rows.Err()
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) ([]PotentialAnswerInfo, error) {
	rows, err := d.DB.Query(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering, to_alert from potential_answer 
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

func createAnswerInfosFromRows(rows *sql.Rows) ([]PotentialAnswerInfo, error) {
	answerInfos := make([]PotentialAnswerInfo, 0)
	for rows.Next() {
		var id, ordering int64
		var answerType, answerTag string
		var answer, answerSummary sql.NullString
		var toAlert sql.NullBool
		err := rows.Scan(&id, &answer, &answerSummary, &answerType, &answerTag, &ordering, &toAlert)
		potentialAnswerInfo := PotentialAnswerInfo{
			Answer:            answer.String,
			AnswerSummary:     answerSummary.String,
			PotentialAnswerId: id,
			AnswerTag:         answerTag,
			Ordering:          ordering,
			AnswerType:        answerType,
			ToAlert:           toAlert.Valid && toAlert.Bool,
		}
		answerInfos = append(answerInfos, potentialAnswerInfo)
		if err != nil {
			return answerInfos, err
		}
	}
	return answerInfos, rows.Err()
}

func (d *DataService) GetAnswerInfoForTags(answerTags []string, languageId int64) ([]PotentialAnswerInfo, error) {

	params := make([]interface{}, 0)
	params = appendStringsToInterfaceSlice(params, answerTags)
	params = append(params, languageId)
	params = append(params, languageId)
	rows, err := d.DB.Query(fmt.Sprintf(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering, to_alert from potential_answer 
								left outer join localized_text as l1 on answer_localized_text_id=l1.app_text_id 
								left outer join answer_type on atype_id=answer_type.id 
								left outer join localized_text as l2 on answer_summary_text_id=l2.app_text_id
									where potential_answer_tag in (%s) and (l1.language_id = ? or l1.ltext is null) and (l2.language_id = ? or l2.ltext is null) and status='ACTIVE'`, nReplacements(len(answerTags))), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return createAnswerInfosFromRows(rows)
}

func (d *DataService) GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error) {
	err = d.DB.QueryRow(`select tips_section.id, ltext1.ltext, ltext2.ltext from tips_section 
								inner join localized_text as ltext1 on tips_title_text_id=ltext1.app_text_id 
								inner join localized_text as ltext2 on tips_subtext_text_id=ltext2.app_text_id 
									where ltext1.language_id = ? and tips_section_tag = ?`, languageId, tipSectionTag).Scan(&id, &tipSectionTitle, &tipSectionSubtext)
	return
}

func (d *DataService) GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error) {
	err = d.DB.QueryRow(`select tips.id, ltext from tips
								inner join localized_text on app_text_id=tips_text_id 
									where tips_tag = ? and language_id = ?`, tipTag, languageId).Scan(&id, &tip)
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
	return languagesSupported, languagesSupportedIds, rows.Err()
}
