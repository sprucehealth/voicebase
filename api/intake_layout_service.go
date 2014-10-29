package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/sku"
)

func (d *DataService) GetQuestionType(questionId int64) (string, error) {
	var questionType string
	err := d.db.QueryRow(`select qtype from question
						inner join question_type on question_type.id = qtype_id
						where question.id = ?`, questionId).Scan(&questionType)
	return questionType, err
}

func (d *DataService) IntakeLayoutForReviewLayoutVersion(reviewMajor, reviewMinor int, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error) {
	var layout []byte
	var layoutVersionID int64
	if reviewMajor == 0 && reviewMinor == 0 {
		// return the latest active intake layout version in the case
		// that no doctor version is specified
		err := d.db.QueryRow(`
			SELECT layout_version_id, layout
			FROM patient_layout_version
			INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
			WHERE status = ? AND health_condition_id = ? AND language_id = ? AND sku_id = ?
			ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, STATUS_ACTIVE, healthConditionID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
			Scan(&layoutVersionID, &layout)
		if err == sql.ErrNoRows {
			return nil, 0, NoRowsError
		} else if err != nil {
			return nil, 0, err
		}
		return layout, layoutVersionID, nil
	}

	// first look up the intake MAJOR,MINOR pairing
	var intakeMajor, intakeMinor int
	err := d.db.QueryRow(`
		SELECT patient_major, patient_minor
		FROM patient_doctor_layout_mapping
		WHERE dr_major = ? AND dr_minor = ? AND health_condition_id = ? AND sku_id = ?`,
		reviewMajor, reviewMinor, healthConditionID, d.skuMapping[skuType.String()]).
		Scan(&intakeMajor, &intakeMinor)
	if err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	// now find the latest patient layout version with this MAJOR,MINOR pairing
	err = d.db.QueryRow(`
		SELECT layout_version_id, layout
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
		WHERE major = ? AND minor = ? AND health_condition_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`,
		intakeMajor, intakeMinor, healthConditionID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) ReviewLayoutForIntakeLayoutVersionID(layoutVersionID int64, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error) {
	// identify the MAJOR, MINOR id of the given layoutVersionID
	var intakeMajor, intakeMinor int
	if err := d.db.QueryRow(`
		SELECT major, minor 
		FROM layout_version
		WHERE id = ?`, layoutVersionID).Scan(&intakeMajor, &intakeMinor); err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	return d.ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor, healthConditionID, skuType)
}

func (d *DataService) ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor int, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error) {
	var layout []byte
	var layoutVersionID int64
	if intakeMajor == 0 && intakeMinor == 0 {
		// return the latest active review layout version in the case
		// that no patient version is specified
		err := d.db.QueryRow(`
			SELECT layout_version_id, layout
			FROM dr_layout_version
			INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
			WHERE status = ? AND health_condition_id = ? AND sku_id = ?
			ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, STATUS_ACTIVE, healthConditionID, d.skuMapping[skuType.String()]).
			Scan(&layoutVersionID, &layout)
		if err == sql.ErrNoRows {
			return nil, 0, NoRowsError
		} else if err != nil {
			return nil, 0, err
		}
		return layout, layoutVersionID, nil
	}

	// first look up the review MAJOR,MINOR pairing
	var reviewMajor, reviewMinor int
	err := d.db.QueryRow(`
		SELECT dr_major, dr_minor
		FROM patient_doctor_layout_mapping
		WHERE patient_major = ? AND patient_minor = ? AND health_condition_id = ? AND sku_id = ?`,
		intakeMajor, intakeMinor, healthConditionID, d.skuMapping[skuType.String()]).
		Scan(&reviewMajor, &reviewMinor)
	if err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	// now find the latest review layout version with this MAJOR,MINOR pairing
	err = d.db.QueryRow(`
		SELECT layout_version_id, layout
		FROM dr_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = dr_layout_version.layout_blob_storage_id
		WHERE major = ? AND minor = ? AND health_condition_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, reviewMajor, reviewMinor,
		healthConditionID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) IntakeLayoutForAppVersion(appVersion *common.Version, platform common.Platform, healthConditionID, languageID int64, skuType sku.SKU) ([]byte, int64, error) {

	if appVersion == nil || appVersion.IsZero() {
		return nil, 0, errors.New("No app version specified")
	}

	// identify the major version of the intake layout supported by the provided app version
	intakeMajor, err := d.majorLayoutVersionSupportedByAppVersion(appVersion, platform, healthConditionID, PATIENT_ROLE, ConditionIntakePurpose, skuType)
	if err != nil {
		return nil, 0, err
	}

	// now identify the latest version for the MAJOR version
	var layout []byte
	var layoutVersionID int64
	err = d.db.QueryRow(`
		SELECT layout_version_id, layout
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
		WHERE major = ? AND status = ? AND health_condition_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major desc, minor DESC, patch DESC LIMIT 1
		`, intakeMajor, STATUS_ACTIVE, healthConditionID, languageID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, NoRowsError
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) majorLayoutVersionSupportedByAppVersion(appVersion *common.Version, platform common.Platform, healthConditionID int64, role, purpose string, skuType sku.SKU) (int, error) {
	var intakeMajor int
	err := d.db.QueryRow(`
		SELECT layout_major 
		FROM app_version_layout_mapping
		WHERE health_condition_id = ? 
			AND (
					app_major < ?
					OR (app_major = ? AND app_minor < ?)
					OR (app_major = ? AND app_minor = ? AND app_patch <= ?)
				) 
			AND platform = ?
			AND role = ? AND purpose = ?
			AND sku_id = ?
		ORDER BY app_major DESC, app_minor DESC, app_patch DESC LIMIT 1`,
		healthConditionID,
		appVersion.Major,
		appVersion.Major, appVersion.Minor,
		appVersion.Major, appVersion.Minor, appVersion.Patch,
		platform.String(), role,
		purpose, d.skuMapping[skuType.String()]).Scan(&intakeMajor)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return intakeMajor, nil
}

func (d *DataService) IntakeLayoutVersionIDForAppVersion(appVersion *common.Version, platform common.Platform, healthConditionID, languageID int64, skuType sku.SKU) (int64, error) {
	if appVersion == nil || appVersion.IsZero() {
		return 0, errors.New("No app version specified")
	}

	// identify the major version of the intake layout supported by the provided app version
	intakeMajor, err := d.majorLayoutVersionSupportedByAppVersion(appVersion, platform, healthConditionID, PATIENT_ROLE, ConditionIntakePurpose, skuType)
	if err != nil {
		return 0, err
	}

	var layoutVersionID int64
	err = d.db.QueryRow(`
		SELECT layout_version_id
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
		WHERE major = ? AND status = ? AND health_condition_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major desc, minor DESC, patch DESC LIMIT 1
		`, intakeMajor, STATUS_ACTIVE, healthConditionID, languageID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return layoutVersionID, nil
}

func (d *DataService) GetActiveDoctorDiagnosisLayout(healthConditionId int64) (*LayoutVersion, error) {
	var layoutVersion LayoutVersion
	err := d.db.QueryRow(`
		SELECT diagnosis_layout_version.id, layout, layout_version_id, major, minor, patch 
		FROM diagnosis_layout_version
		INNER JOIN layout_blob_storage on diagnosis_layout_version.layout_blob_storage_id=layout_blob_storage.id 
		WHERE status=? AND health_condition_id = ?`,
		STATUS_ACTIVE, healthConditionId).
		Scan(&layoutVersion.ID, &layoutVersion.Layout, &layoutVersion.LayoutTemplateVersionID, &layoutVersion.Version.Major,
		&layoutVersion.Version.Minor, &layoutVersion.Version.Patch)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &layoutVersion, nil
}

func (d *DataService) CreateLayoutMapping(intakeMajor, intakeMinor, reviewMajor, reviewMinor int, healthConditionID int64, skuType sku.SKU) error {
	_, err := d.db.Exec(`
		INSERT INTO patient_doctor_layout_mapping 
		(dr_major, dr_minor, patient_major, patient_minor, health_condition_id, sku_id)
		VALUES (?,?,?,?,?,?)`,
		reviewMajor, reviewMinor, intakeMajor, intakeMinor, healthConditionID, d.skuMapping[skuType.String()])
	return err
}

func (d *DataService) CreateAppVersionMapping(appVersion *common.Version, platform common.Platform,
	layoutMajor int, role, purpose string, healthConditionID int64, skuType sku.SKU) error {

	if appVersion == nil || appVersion.IsZero() {
		return errors.New("no app version specified")
	}

	_, err := d.db.Exec(`
		INSERT INTO app_version_layout_mapping 
		(app_major, app_minor, app_patch, layout_major, health_condition_id, platform, role, purpose, sku_id)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		appVersion.Major, appVersion.Minor, appVersion.Patch, layoutMajor,
		healthConditionID, platform.String(), role, purpose, d.skuMapping[skuType.String()])
	return err
}

func (d *DataService) GetLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (int64, error) {
	var layoutVersionId int64
	err := d.db.QueryRow(`select layout_version_id from diagnosis_layout_version 
					inner join layout_version on layout_version_id=layout_version.id 
						where diagnosis_layout_version.status = ? and layout_purpose = ? and role = ? 
						and diagnosis_layout_version.health_condition_id = ?`,
		STATUS_ACTIVE, DiagnosePurpose, DOCTOR_ROLE, healthConditionId).Scan(&layoutVersionId)
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

func (d *DataService) GetPatientLayout(layoutVersionId, languageId int64) (*LayoutVersion, error) {
	var layoutVersion LayoutVersion
	err := d.db.QueryRow(`
		SELECT patient_layout_version.id, layout, layout_version_id, major, minor, patch 
		FROM patient_layout_version 
		INNER JOIN layout_blob_storage ON layout_blob_storage_id=layout_blob_storage.id 
		WHERE layout_version_id = ? and language_id = ?`, layoutVersionId, languageId).
		Scan(&layoutVersion.ID, &layoutVersion.Layout, &layoutVersion.LayoutTemplateVersionID, &layoutVersion.Version.Major,
		&layoutVersion.Version.Minor, &layoutVersion.Version.Patch)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &layoutVersion, nil
}

func (d *DataService) LayoutTemplateVersionBeyondVersion(versionInfo *VersionInfo, role, purpose string, healthConditionID int64, skuID *int64) (*LayoutTemplateVersion, error) {
	cols := make([]string, 0, 8)
	vals := make([]interface{}, 0, 9)
	cols = append(cols, "layout_purpose = ?", "role = ?", "status in (?, ?)", "health_condition_id = ?")
	vals = append(vals, purpose, role, STATUS_ACTIVE, STATUS_DEPRECATED, healthConditionID)

	if skuID != nil {
		cols = append(cols, "sku_id = ?")
		vals = append(vals, skuID)
	}

	if versionInfo != nil {
		if versionInfo.Major != nil {
			cols = append(cols, "major = ?")
			vals = append(vals, *versionInfo.Major)
		}
		if versionInfo.Minor != nil {
			cols = append(cols, "minor = ?")
			vals = append(vals, *versionInfo.Minor)
		}
		if versionInfo.Patch != nil {
			cols = append(cols, "patch = ?")
			vals = append(vals, *versionInfo.Patch)
		}
	}

	var layoutVersion LayoutTemplateVersion
	if err := d.db.QueryRow(`
		SELECT id, major, minor, patch, layout_purpose, role, health_condition_id, sku_id, status 
		FROM layout_version 
		WHERE `+strings.Join(cols, " AND ")+` ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, vals...).Scan(
		&layoutVersion.ID,
		&layoutVersion.Version.Major,
		&layoutVersion.Version.Minor,
		&layoutVersion.Version.Patch,
		&layoutVersion.Purpose,
		&layoutVersion.Role,
		&layoutVersion.HealthConditionID,
		&layoutVersion.SKUID,
		&layoutVersion.Status); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	return &layoutVersion, nil
}

func (d *DataService) CreateLayoutTemplateVersion(layout *LayoutTemplateVersion) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	insertId, err := tx.Exec(`insert into layout_blob_storage (layout) values (?)`, layout.Layout)
	if err != nil {
		tx.Rollback()
		return err
	}

	layoutBlobStorageId, err := insertId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	res, err := tx.Exec(`
		INSERT INTO layout_version (layout_blob_storage_id, major, minor, patch, health_condition_id, sku_id, role, layout_purpose, status) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, layoutBlobStorageId, layout.Version.Major, layout.Version.Minor, layout.Version.Patch,
		layout.HealthConditionID, layout.SKUID, layout.Role, layout.Purpose, layout.Status)
	if err != nil {
		tx.Rollback()
		return err
	}

	layout.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CreateLayoutVersion(layout *LayoutVersion) error {
	var tableName string
	cols := []string{"major", "minor", "patch", "layout_version_id", "health_condition_id", "language_id", "status"}
	vals := []interface{}{layout.Version.Major, layout.Version.Minor, layout.Version.Patch, layout.LayoutTemplateVersionID,
		layout.HealthConditionID, layout.LanguageID, layout.Status}

	switch layout.Purpose {
	case ConditionIntakePurpose:
		tableName = "patient_layout_version"
		cols = append(cols, "sku_id")
		vals = append(vals, layout.SKUID)
	case ReviewPurpose:
		tableName = "dr_layout_version"
		cols = append(cols, "sku_id")
		vals = append(vals, layout.SKUID)
	case DiagnosePurpose:
		tableName = "diagnosis_layout_version"
	}

	tx, err := d.db.Begin()
	if err != nil {
		return nil
	}

	lastInsertId, err := tx.Exec(`INSERT INTO layout_blob_storage (layout) VALUES (?)`, layout.Layout)
	if err != nil {
		tx.Rollback()
		return err
	}

	layoutBlobStorageId, err := lastInsertId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	cols = append(cols, "layout_blob_storage_id")
	vals = append(vals, layoutBlobStorageId)

	res, err := tx.Exec(`
		INSERT INTO `+tableName+` (`+strings.Join(cols, ",")+` )
		VALUES (`+nReplacements(len(vals))+`)`, vals...)
	if err != nil {
		tx.Rollback()
		return err
	}

	layout.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) UpdateActiveLayouts(purpose string, version *common.Version, layoutTemplateID int64, clientLayoutIDs []int64,
	healthConditionID int64, skuID *int64) error {
	var tableName string

	whereClause := "status = ? AND health_condition_id = ? AND major = ?"
	vals := []interface{}{STATUS_ACTIVE, healthConditionID, version.Major}

	switch purpose {
	case ConditionIntakePurpose:
		tableName = "patient_layout_version"
		whereClause += " AND sku_id = ?"
		vals = append(vals, skuID)
	case ReviewPurpose:
		tableName = "dr_layout_version"
		whereClause += " AND sku_id = ?"
		vals = append(vals, skuID)
	case DiagnosePurpose:
		tableName = "diagnosis_layout_version"
	default:
		return errors.New("Unknown purpose")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// identify the layoutVersionIDs to mark as inactive
	rows, err := tx.Query(`
		SELECT layout_version_id
		FROM `+tableName+`
		WHERE `+whereClause, vals...)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()

	var layoutVersionIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return err
		}
		layoutVersionIDs = append(layoutVersionIDs, id)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}

	if len(layoutVersionIDs) > 0 {

		v := []interface{}{STATUS_DEPRECATED}
		v = appendInt64sToInterfaceSlice(v, layoutVersionIDs)

		// mark all other layout for this MAJOR version as deprecated
		_, err = tx.Exec(`
		UPDATE layout_version 
		SET status=? 
		WHERE id in (`+nReplacements(len(layoutVersionIDs))+`)`, v...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// mark all other layouts for this MAJOR version as deprecated
	v := []interface{}{STATUS_DEPRECATED}
	v = append(v, vals...)
	_, err = tx.Exec(`
		UPDATE `+tableName+
		` SET status = ?  
		WHERE `+whereClause, v...)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = tx.Exec(`
		UPDATE layout_version 
		SET status = ? where id = ?`, STATUS_ACTIVE, layoutTemplateID)
	if err != nil {
		tx.Rollback()
		return err
	}

	params := make([]interface{}, 0, 1+len(clientLayoutIDs))
	params = appendInt64sToInterfaceSlice(append(params, STATUS_ACTIVE), clientLayoutIDs)
	_, err = tx.Exec(`
		UPDATE `+tableName+
		` SET status = ? 
		WHERE id in (`+nReplacements(len(clientLayoutIDs))+`)`,
		params...)
	if err != nil {
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

	// order the items based on the ordering of the tags (note that its totally possible
	// that some tags requested didn't exist as answers in which case there would be more tags than answers)
	answerInfoInOrder := make([]*info_intake.PotentialAnswer, 0, len(answerInfos))
	for _, answerTag := range answerTags {
		answer := answerInfoMapping[answerTag]
		if answer != nil {
			answerInfoInOrder = append(answerInfoInOrder, answer)
		}
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

func (d *DataService) LatestAppVersionSupported(healthConditionId int64, skuID *int64, platform common.Platform, role, purpose string) (*common.Version, error) {
	var version common.Version
	vals := []interface{}{healthConditionId, platform.String(), role, purpose}
	whereClause := "health_condition_id = ? AND platform = ? AND role = ? AND purpose = ?"
	if skuID != nil {
		whereClause += " AND sku_id = ?"
		vals = append(vals, skuID)
	}
	err := d.db.QueryRow(`
		SELECT app_major, app_minor, app_patch
		FROM app_version_layout_mapping 
		WHERE `+whereClause+`
		ORDER BY app_major DESC, app_minor DESC, app_patch DESC`, vals...).
		Scan(&version.Major, &version.Minor, &version.Patch)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	}

	return &version, nil
}
