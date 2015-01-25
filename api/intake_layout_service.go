package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/sku"
)

func (d *DataService) GetQuestionType(questionID int64) (string, error) {
	var questionType string
	err := d.db.QueryRow(
		`SELECT question_type FROM question
			WHERE question.id = ?`, questionID).Scan(&questionType)
	return questionType, err
}

func (d *DataService) IntakeLayoutForReviewLayoutVersion(reviewMajor, reviewMinor int, pathwayID int64, skuType sku.SKU) ([]byte, int64, error) {
	var layout []byte
	var layoutVersionID int64
	if reviewMajor == 0 && reviewMinor == 0 {
		// return the latest active intake layout version in the case
		// that no doctor version is specified
		err := d.db.QueryRow(`
			SELECT layout_version_id, layout
			FROM patient_layout_version
			INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
			WHERE status = ? AND clinical_pathway_id = ? AND language_id = ? AND sku_id = ?
			ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, STATUS_ACTIVE, pathwayID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
			Scan(&layoutVersionID, &layout)
		if err == sql.ErrNoRows {
			return nil, 0, ErrNotFound("patient_layout_version")
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
		WHERE dr_major = ? AND dr_minor = ? AND clinical_pathway_id = ? AND sku_id = ?`,
		reviewMajor, reviewMinor, pathwayID, d.skuMapping[skuType.String()]).
		Scan(&intakeMajor, &intakeMinor)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("patient_doctor_layout_mapping")
	} else if err != nil {
		return nil, 0, err
	}

	// now find the latest patient layout version with this MAJOR,MINOR pairing
	err = d.db.QueryRow(`
		SELECT layout_version_id, layout
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
		WHERE major = ? AND minor = ? AND clinical_pathway_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`,
		intakeMajor, intakeMinor, pathwayID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("patient_layout_version")
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) ReviewLayoutForIntakeLayoutVersionID(layoutVersionID, pathwayID int64, skuType sku.SKU) ([]byte, int64, error) {
	// identify the MAJOR, MINOR id of the given layoutVersionID
	var intakeMajor, intakeMinor int
	if err := d.db.QueryRow(`
		SELECT major, minor
		FROM layout_version
		WHERE id = ?`, layoutVersionID).Scan(&intakeMajor, &intakeMinor); err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("layout_version")
	} else if err != nil {
		return nil, 0, err
	}

	return d.ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor, pathwayID, skuType)
}

func (d *DataService) ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor int, pathwayID int64, skuType sku.SKU) ([]byte, int64, error) {
	var layout []byte
	var layoutVersionID int64
	if intakeMajor == 0 && intakeMinor == 0 {
		// return the latest active review layout version in the case
		// that no patient version is specified
		err := d.db.QueryRow(`
			SELECT layout_version_id, layout
			FROM dr_layout_version
			INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
			WHERE status = ? AND clinical_pathway_id = ? AND sku_id = ?
			ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, STATUS_ACTIVE, pathwayID, d.skuMapping[skuType.String()]).
			Scan(&layoutVersionID, &layout)
		if err == sql.ErrNoRows {
			return nil, 0, ErrNotFound("dr_layout_version")
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
		WHERE patient_major = ? AND patient_minor = ? AND clinical_pathway_id = ? AND sku_id = ?`,
		intakeMajor, intakeMinor, pathwayID, d.skuMapping[skuType.String()]).
		Scan(&reviewMajor, &reviewMinor)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("patient_doctor_layout_mapping")
	} else if err != nil {
		return nil, 0, err
	}

	// now find the latest review layout version with this MAJOR,MINOR pairing
	err = d.db.QueryRow(`
		SELECT layout_version_id, layout
		FROM dr_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = dr_layout_version.layout_blob_storage_id
		WHERE major = ? AND minor = ? AND clinical_pathway_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, reviewMajor, reviewMinor,
		pathwayID, EN_LANGUAGE_ID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("dr_layout_version")
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) IntakeLayoutForAppVersion(appVersion *common.Version, platform common.Platform, pathwayID, languageID int64, skuType sku.SKU) ([]byte, int64, error) {

	if appVersion == nil || appVersion.IsZero() {
		return nil, 0, errors.New("No app version specified")
	}

	// identify the major version of the intake layout supported by the provided app version
	intakeMajor, err := d.majorLayoutVersionSupportedByAppVersion(appVersion, platform, pathwayID, PATIENT_ROLE, ConditionIntakePurpose, skuType)
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
		WHERE major = ? AND status = ? AND clinical_pathway_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major desc, minor DESC, patch DESC LIMIT 1
		`, intakeMajor, STATUS_ACTIVE, pathwayID, languageID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID, &layout)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound("patient_layout_version")
	} else if err != nil {
		return nil, 0, err
	}

	return layout, layoutVersionID, nil
}

func (d *DataService) majorLayoutVersionSupportedByAppVersion(appVersion *common.Version, platform common.Platform, pathwayID int64, role, purpose string, skuType sku.SKU) (int, error) {
	var intakeMajor int
	err := d.db.QueryRow(`
		SELECT layout_major
		FROM app_version_layout_mapping
		WHERE clinical_pathway_id = ?
			AND (
					app_major < ?
					OR (app_major = ? AND app_minor < ?)
					OR (app_major = ? AND app_minor = ? AND app_patch <= ?)
			)
			AND platform = ?
			AND role = ? AND purpose = ?
			AND sku_id = ?
		ORDER BY app_major DESC, app_minor DESC, app_patch DESC LIMIT 1`,
		pathwayID,
		appVersion.Major,
		appVersion.Major, appVersion.Minor,
		appVersion.Major, appVersion.Minor, appVersion.Patch,
		platform.String(), role,
		purpose, d.skuMapping[skuType.String()]).Scan(&intakeMajor)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("app_version_layout_mapping")
	} else if err != nil {
		return 0, err
	}

	return intakeMajor, nil
}

func (d *DataService) IntakeLayoutVersionIDForAppVersion(appVersion *common.Version, platform common.Platform, pathwayID, languageID int64, skuType sku.SKU) (int64, error) {
	if appVersion == nil || appVersion.IsZero() {
		return 0, errors.New("No app version specified")
	}

	// identify the major version of the intake layout supported by the provided app version
	intakeMajor, err := d.majorLayoutVersionSupportedByAppVersion(appVersion, platform, pathwayID, PATIENT_ROLE, ConditionIntakePurpose, skuType)
	if err != nil {
		return 0, err
	}

	var layoutVersionID int64
	err = d.db.QueryRow(`
		SELECT layout_version_id
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage.id = patient_layout_version.layout_blob_storage_id
		WHERE major = ? AND status = ? AND clinical_pathway_id = ? AND language_id = ? AND sku_id = ?
		ORDER BY major desc, minor DESC, patch DESC LIMIT 1
		`, intakeMajor, STATUS_ACTIVE, pathwayID, languageID, d.skuMapping[skuType.String()]).
		Scan(&layoutVersionID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("patient_layout_version")
	} else if err != nil {
		return 0, err
	}

	return layoutVersionID, nil
}

func (d *DataService) GetActiveDoctorDiagnosisLayout(pathwayID int64) (*LayoutVersion, error) {
	var layoutVersion LayoutVersion
	err := d.db.QueryRow(`
		SELECT diagnosis_layout_version.id, layout, layout_version_id, major, minor, patch
		FROM diagnosis_layout_version
		INNER JOIN layout_blob_storage ON diagnosis_layout_version.layout_blob_storage_id = layout_blob_storage.id
		WHERE status=? AND clinical_pathway_id = ?`,
		STATUS_ACTIVE, pathwayID).
		Scan(&layoutVersion.ID, &layoutVersion.Layout, &layoutVersion.LayoutTemplateVersionID, &layoutVersion.Version.Major,
		&layoutVersion.Version.Minor, &layoutVersion.Version.Patch)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("diagnosis_layout_version")
	} else if err != nil {
		return nil, err
	}
	return &layoutVersion, nil
}

func (d *DataService) CreateLayoutMapping(intakeMajor, intakeMinor, reviewMajor, reviewMinor int, pathwayID int64, skuType sku.SKU) error {
	_, err := d.db.Exec(`
		INSERT INTO patient_doctor_layout_mapping
			(dr_major, dr_minor, patient_major, patient_minor, clinical_pathway_id, sku_id)
		VALUES (?,?,?,?,?,?)`,
		reviewMajor, reviewMinor, intakeMajor, intakeMinor, pathwayID, d.skuMapping[skuType.String()])
	return err
}

func (d *DataService) CreateAppVersionMapping(appVersion *common.Version, platform common.Platform,
	layoutMajor int, role, purpose string, pathwayID int64, skuType sku.SKU) error {

	if appVersion == nil || appVersion.IsZero() {
		return errors.New("no app version specified")
	}

	_, err := d.db.Exec(`
		INSERT INTO app_version_layout_mapping
		(app_major, app_minor, app_patch, layout_major, clinical_pathway_id, platform, role, purpose, sku_id)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		appVersion.Major, appVersion.Minor, appVersion.Patch, layoutMajor,
		pathwayID, platform.String(), role, purpose, d.skuMapping[skuType.String()])
	return err
}

func (d *DataService) GetLayoutVersionIDOfActiveDiagnosisLayout(pathwayID int64) (int64, error) {
	var layoutVersionID int64
	err := d.db.QueryRow(`
		SELECT layout_version_id
		FROM diagnosis_layout_version
		INNER JOIN layout_version ON layout_version_id = layout_version.id
		WHERE diagnosis_layout_version.status = ?
			AND layout_purpose = ?
			AND role = ?
			AND diagnosis_layout_version.clinical_pathway_id = ?`,
		STATUS_ACTIVE, DiagnosePurpose, DOCTOR_ROLE, pathwayID).Scan(&layoutVersionID)
	return layoutVersionID, err

}

func (d *DataService) getActiveDoctorLayoutForPurpose(pathwayID int64, purpose string) ([]byte, int64, error) {
	var layoutBlob []byte
	var layoutVersionID int64
	row := d.db.QueryRow(`
		SELECT layout, layout_version_id
		FROM dr_layout_version
		INNER JOIN layout_version ON layout_version_id = layout_version.id
		INNER JOIN layout_blob_storage ON dr_layout_version.layout_blob_storage_id = layout_blob_storage.id
		WHERE dr_layout_version.status = ?
			AND layout_purpose = ?
			AND role = ?
			AND dr_layout_version.clinical_pathway_id = ?`,
		STATUS_ACTIVE, purpose, DOCTOR_ROLE, pathwayID)
	err := row.Scan(&layoutBlob, &layoutVersionID)
	return layoutBlob, layoutVersionID, err
}

func (d *DataService) GetPatientLayout(layoutVersionID, languageID int64) (*LayoutVersion, error) {
	var layoutVersion LayoutVersion
	err := d.db.QueryRow(`
		SELECT patient_layout_version.id, layout, layout_version_id, major, minor, patch
		FROM patient_layout_version
		INNER JOIN layout_blob_storage ON layout_blob_storage_id = layout_blob_storage.id
		WHERE layout_version_id = ? AND language_id = ?`, layoutVersionID, languageID).
		Scan(&layoutVersion.ID, &layoutVersion.Layout, &layoutVersion.LayoutTemplateVersionID, &layoutVersion.Version.Major,
		&layoutVersion.Version.Minor, &layoutVersion.Version.Patch)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_layout_version")
	} else if err != nil {
		return nil, err
	}
	return &layoutVersion, nil
}

func (d *DataService) LayoutTemplateVersionBeyondVersion(versionInfo *VersionInfo, role, purpose string, pathwayID int64, skuID *int64) (*LayoutTemplateVersion, error) {
	cols := make([]string, 0, 8)
	vals := make([]interface{}, 0, 9)
	cols = append(cols, "layout_purpose = ?", "role = ?", "status in (?, ?)", "clinical_pathway_id = ?")
	vals = append(vals, purpose, role, STATUS_ACTIVE, STATUS_DEPRECATED, pathwayID)

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
		SELECT id, major, minor, patch, layout_purpose, role, clinical_pathway_id, sku_id, status
		FROM layout_version
		WHERE `+strings.Join(cols, " AND ")+`
		ORDER BY major DESC, minor DESC, patch DESC LIMIT 1`, vals...).Scan(
		&layoutVersion.ID,
		&layoutVersion.Version.Major,
		&layoutVersion.Version.Minor,
		&layoutVersion.Version.Patch,
		&layoutVersion.Purpose,
		&layoutVersion.Role,
		&layoutVersion.PathwayID,
		&layoutVersion.SKUID,
		&layoutVersion.Status); err == sql.ErrNoRows {
		return nil, ErrNotFound("layout_version")
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
		INSERT INTO layout_version (layout_blob_storage_id, major, minor, patch, clinical_pathway_id, sku_id, role, layout_purpose, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, layoutBlobStorageId, layout.Version.Major, layout.Version.Minor, layout.Version.Patch,
		layout.PathwayID, layout.SKUID, layout.Role, layout.Purpose, layout.Status)
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
	cols := []string{"major", "minor", "patch", "layout_version_id", "clinical_pathway_id", "language_id", "status"}
	vals := []interface{}{layout.Version.Major, layout.Version.Minor, layout.Version.Patch, layout.LayoutTemplateVersionID,
		layout.PathwayID, layout.LanguageID, layout.Status}

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
		VALUES (`+dbutil.MySQLArgs(len(vals))+`)`, vals...)
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
	pathwayID int64, skuID *int64) error {
	var tableName string

	whereClause := "status = ? AND clinical_pathway_id = ? AND major = ?"
	vals := []interface{}{STATUS_ACTIVE, pathwayID, version.Major}

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
		v = dbutil.AppendInt64sToInterfaceSlice(v, layoutVersionIDs)

		// mark all other layout for this MAJOR version as deprecated
		_, err = tx.Exec(`
		UPDATE layout_version 
		SET status=? 
		WHERE id in (`+dbutil.MySQLArgs(len(layoutVersionIDs))+`)`, v...)
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
	params = dbutil.AppendInt64sToInterfaceSlice(append(params, STATUS_ACTIVE), clientLayoutIDs)
	_, err = tx.Exec(`
		UPDATE `+tableName+
		` SET status = ?
		WHERE id in (`+dbutil.MySQLArgs(len(clientLayoutIDs))+`)`,
		params...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetSectionIDsForPathway(pathwayID int64) ([]int64, error) {
	rows, err := d.db.Query(`SELECT id FROM section WHERE clinical_pathway_id = ?`, pathwayID)
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

func (d *DataService) GetSectionInfo(sectionTag string, languageID int64) (id int64, title string, err error) {
	err = d.db.QueryRow(`
		SELECT section.id, ltext
		FROM section
		INNER JOIN app_text on section_title_app_text_id = app_text.id
		INNER JOIN localized_text on app_text_id = app_text.id
		WHERE language_id = ? AND section_tag = ?`,
		languageID, sectionTag).Scan(&id, &title)
	if err == sql.ErrNoRows {
		err = ErrNotFound("section")
	}
	return
}

// QuestionQueryParams is an object used to describe the paramters needed to correctly query a versioned question
type QuestionQueryParams struct {
	QuestionTag string
	LanguageID  int64
	Version     int64
}

// AnswerQueryParams is an object used to describe the paramters needed to correctly query a versioned question
type AnswerQueryParams struct {
	AnswerTag  string
	QuestionID int64
	LanguageID int64
}

// VersionedQuestionFromID retrieves a single record from the question table relating to a specific versioned answer
func (d *DataService) VersionedQuestionFromID(id int64) (*common.VersionedQuestion, error) {
	vq := &common.VersionedQuestion{}
	var parentID sql.NullInt64
	var err error
	if err = d.db.QueryRow(
		`SELECT id, question_tag, parent_question_id, COALESCE(required,0), COALESCE(formatted_field_tags,''),
      COALESCE(to_alert,0), COALESCE(qtext_has_tokens,0), language_id, version, COALESCE(question_text,''), COALESCE(subtext_text,''), COALESCE(summary_text,''), COALESCE(alert_text,''), question_type
      FROM question
      WHERE id = ?`, id).Scan(&vq.ID, &vq.QuestionTag, &parentID, &vq.Required, &vq.FormattedFieldTags,
		&vq.ToAlert, &vq.TextHasTokens, &vq.LanguageID, &vq.Version, &vq.QuestionText, &vq.SubtextText,
		&vq.SummaryText, &vq.AlertText, &vq.QuestionType); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound("question")
		}
		return nil, err
	}
	if parentID.Valid {
		vq.ParentQuestionID = &parentID.Int64
	}
	return vq, nil
}

// VersionedQuestion retrieves a set of records from the question table relating to a specific set of versioned questions based on versioning info
func (d *DataService) VersionedQuestions(questionQueryParams []*QuestionQueryParams) ([]*common.VersionedQuestion, error) {
	if len(questionQueryParams) == 0 {
		return nil, nil
	}

	versionedQuestionStmt, err :=
		d.db.Prepare(
			`SELECT id, question_tag, parent_question_id, COALESCE(required,0), COALESCE(formatted_field_tags,''),
      	COALESCE(to_alert,0), COALESCE(qtext_has_tokens,0), language_id, version, COALESCE(question_text,''), COALESCE(subtext_text,''), COALESCE(summary_text,''), COALESCE(alert_text,''), question_type
      	FROM question WHERE 
      	question_tag = ? AND 
      	language_id = ? AND
      	version = ?`)
	if err != nil {
		return nil, err
	}
	defer versionedQuestionStmt.Close()

	versionedQuestions := make([]*common.VersionedQuestion, len(questionQueryParams))
	for i, v := range questionQueryParams {
		parentID := sql.NullInt64{}
		vq := &common.VersionedQuestion{}
		if err := versionedQuestionStmt.QueryRow(v.QuestionTag, v.LanguageID, v.Version).Scan(
			&vq.ID, &vq.QuestionTag, &parentID, &vq.Required, &vq.FormattedFieldTags,
			&vq.ToAlert, &vq.TextHasTokens, &vq.LanguageID, &vq.Version, &vq.QuestionText, &vq.SubtextText,
			&vq.SummaryText, &vq.AlertText, &vq.QuestionType); err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrNotFound("question")
			}
			return nil, err
		}
		if parentID.Valid {
			vq.ParentQuestionID = &parentID.Int64
		}
		versionedQuestions[i] = vq
	}

	return versionedQuestions, nil
}

func (d *DataService) InsertVersionedQuestion(versionedQuestion *common.VersionedQuestion, versionedAnswers []*common.VersionedAnswer, versionedAdditionalQuestionField *common.VersionedAdditionalQuestionField) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	id, err := d.insertVersionedQuestionWithVersionedParents(tx, versionedQuestion, versionedAnswers, versionedAdditionalQuestionField)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, nil
}

// InsertVersionedQuestion inserts a new versioned question
func (d *DataService) insertVersionedQuestionWithVersionedParents(db db, versionedQuestion *common.VersionedQuestion, versionedAnswers []*common.VersionedAnswer, versionedAdditionalQuestionField *common.VersionedAdditionalQuestionField) (int64, error) {
	if versionedQuestion.ParentQuestionID != nil {
		pvq, err := d.VersionedQuestionFromID(*versionedQuestion.ParentQuestionID)
		if err != nil {
			return 0, err
		}

		pvas, err := d.VersionedAnswers([]*AnswerQueryParams{&AnswerQueryParams{LanguageID: pvq.LanguageID, QuestionID: pvq.ID}})
		if err != nil {
			return 0, err
		}

		pvaqfs, err := d.VersionedAdditionalQuestionFields(pvq.ID, pvq.LanguageID)
		if err != nil {
			return 0, err
		}

		pvaqf, err := d.flattenVersionedAdditionalQuestionFields(pvaqfs)
		if err != nil {
			return 0, err
		}

		qid, err := d.insertVersionedQuestionWithVersionedParents(db, pvq, pvas, pvaqf)
		if err != nil {
			return 0, err
		}
		versionedQuestion.ParentQuestionID = &qid
	}

	newID, err := d.insertVersionedQuestion(db, versionedQuestion)
	if err != nil {
		return 0, err
	}

	for _, va := range versionedAnswers {
		va.QuestionID = newID
		d.insertVersionedAnswer(db, va)
		if err != nil {
			return 0, err
		}
	}

	if versionedAdditionalQuestionField != nil {
		versionedAdditionalQuestionField.QuestionID = newID
		d.insertVersionedAdditionalQuestionField(db, versionedAdditionalQuestionField)
		if err != nil {
			return 0, err
		}
	}
	versionedQuestion.ID = newID

	return newID, nil
}

// QuestionIDFromTag returns the id of described question
func (d *DataService) QuestionIDFromTag(questionTag string, languageID, version int64) (int64, error) {
	var id int64
	err := d.db.QueryRow(`SELECT id FROM question WHERE question_tag = ? AND language_id = ? AND version = ?`, questionTag, languageID, version).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrNotFound("question")
		}
		return 0, err
	}

	return id, nil
}

// MaxQuestionVersion returns the latest version of the described question
func (d *DataService) MaxQuestionVersion(questionTag string, languageID int64) (int64, error) {
	var maxVersion sql.NullInt64
	err := d.db.QueryRow(`SELECT MAX(version) max FROM question WHERE question_tag = ? AND language_id = ?`, questionTag, languageID).Scan(&maxVersion)
	if err != nil {
		return 0, err
	}
	return maxVersion.Int64, nil
}

// insertVersionedQuestionInTransaction inserts and auto versions the related question set if one is related
// NOTE: Any values in the ID or VERSION fields will be ignored
func (d *DataService) insertVersionedQuestion(db db, versionedQuestion *common.VersionedQuestion) (int64, error) {
	// Note: This initial version does not take into account concurrent modifiers or perform retries.
	// 	It relies on the constraints and transactional safety to reject the loser during concurrent modification
	currentVersion, err := d.MaxQuestionVersion(versionedQuestion.QuestionTag, versionedQuestion.LanguageID)
	switch {
	case err == sql.ErrNoRows:
		versionedQuestion.Version = 1
	case err != nil:
		return 0, err
	default:
		versionedQuestion.Version = currentVersion + 1
	}

	cols := []string{`version`, `question_tag`, `language_id`, `question_type`, `parent_question_id`,
		`required`, `formatted_field_tags`, `to_alert`, `qtext_has_tokens`, `question_text`, `subtext_text`,
		`summary_text`, `alert_text`}
	vals := []interface{}{versionedQuestion.Version, versionedQuestion.QuestionTag, versionedQuestion.LanguageID, versionedQuestion.QuestionType, versionedQuestion.ParentQuestionID,
		versionedQuestion.Required, versionedQuestion.FormattedFieldTags, versionedQuestion.ToAlert, versionedQuestion.TextHasTokens, versionedQuestion.QuestionText, versionedQuestion.SubtextText,
		versionedQuestion.SummaryText, versionedQuestion.AlertText}

	res, err := db.Exec(`INSERT INTO question (`+strings.Join(cols, `, `)+`) VALUES (`+dbutil.MySQLArgs(len(vals))+`)`, vals...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// VersionedAnswerFromID retrieves a single record from the potential_answer table relating to a specific versioned answer
func (d *DataService) VersionedAnswerFromID(id int64) (*common.VersionedAnswer, error) {
	va := &common.VersionedAnswer{}
	if err := d.db.QueryRow(
		`SELECT id, potential_answer_tag, COALESCE(to_alert,0), ordering, question_id, language_id, 
			COALESCE(answer_text,''), COALESCE(answer_summary_text,''), answer_type
      FROM potential_answer WHERE
      id = ?`, id).Scan(&va.ID, &va.AnswerTag, &va.ToAlert, &va.Ordering, &va.QuestionID, &va.LanguageID,
		&va.AnswerText, &va.AnswerSummaryText, &va.AnswerType); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound("potential_answer")
		}
		return nil, err
	}
	return va, nil
}

// VersionedAnswers looks up a given set of versioned answer
func (d *DataService) VersionedAnswers(answerQueryParams []*AnswerQueryParams) ([]*common.VersionedAnswer, error) {
	return d.versionedAnswers(d.db, answerQueryParams)
}

// versionedAnswersInTransaction looks up a given set of versioned answers in the context of a transaction
func (d *DataService) versionedAnswers(db db, answerQueryParams []*AnswerQueryParams) ([]*common.VersionedAnswer, error) {
	if len(answerQueryParams) == 0 {
		return nil, nil
	}

	var versionedAnswers []*common.VersionedAnswer
	for _, queryParams := range answerQueryParams {
		vals := []interface{}{queryParams.QuestionID, queryParams.LanguageID}
		query :=
			`SELECT id, potential_answer_tag, COALESCE(to_alert,0), ordering, question_id, language_id, 
				COALESCE(answer_text,''), COALESCE(answer_summary_text,''), answer_type, status
    		FROM potential_answer WHERE 
    		question_id = ? AND
    		language_id = ?`
		if queryParams.AnswerTag != "" {
			query += ` AND potential_answer_tag = ?`
			vals = append(vals, queryParams.AnswerTag)
		}
		rows, err := db.Query(query, vals...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			va := &common.VersionedAnswer{}
			if err := rows.Scan(&va.ID, &va.AnswerTag, &va.ToAlert, &va.Ordering, &va.QuestionID, &va.LanguageID,
				&va.AnswerText, &va.AnswerSummaryText, &va.AnswerType, &va.Status); err != nil {
				if err == sql.ErrNoRows {
					return nil, ErrNotFound("potential_answer")
				}
				return nil, err
			}
			if err = rows.Err(); err != nil {
				return nil, err
			}
			versionedAnswers = append(versionedAnswers, va)
		}
		// We don't want to keep the rows open. Even though we're stacking defered close calls, close here on success.
		rows.Close()
	}

	return versionedAnswers, nil
}

// VersionedAnswerTagsForQuestion returns a unique set of answer tags associated with the given question id
func (d *DataService) VersionedAnswerTagsForQuestion(questionID int64) ([]string, error) {
	rows, err := d.db.Query(`SELECT DISTINCT(potential_answer_tag) FROM potential_answer WHERE question_id = ?`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// insertVersionedAnswer inserts or a new versioned answer record
func (d *DataService) insertVersionedAnswer(db db, versionedAnswer *common.VersionedAnswer) (int64, error) {
	cols := []string{`potential_answer_tag`, `language_id`, `answer_type`, `ordering`, `question_id`, `status`,
		`to_alert`, `answer_text`, `answer_summary_text`}
	vals := []interface{}{versionedAnswer.AnswerTag, versionedAnswer.LanguageID, versionedAnswer.AnswerType, versionedAnswer.Ordering, versionedAnswer.QuestionID, versionedAnswer.Status,
		versionedAnswer.ToAlert, versionedAnswer.AnswerText, versionedAnswer.AnswerSummaryText}

	res, err := db.Exec(`INSERT INTO potential_answer (`+strings.Join(cols, `, `)+`) VALUES (`+dbutil.MySQLArgs(len(vals))+`)`, vals...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// VersionedAdditionalQuestionFields returns a set of additional fields for the question
func (d *DataService) VersionedAdditionalQuestionFields(questionID, languageID int64) ([]*common.VersionedAdditionalQuestionField, error) {
	rows, err := d.db.Query(`SELECT id, question_id, json, language_id FROM additional_question_fields WHERE question_id = ? AND language_id = ?`, questionID, languageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vaqfs := make([]*common.VersionedAdditionalQuestionField, 0)
	for rows.Next() {
		var jsonBytes []byte
		var id, questionID, languageID int64
		err := rows.Scan(&id, &questionID, &jsonBytes, &languageID)
		if err != nil {
			return nil, err
		}

		vaqfs = append(vaqfs, &common.VersionedAdditionalQuestionField{
			ID:         id,
			QuestionID: questionID,
			JSON:       jsonBytes,
			LanguageID: languageID,
		})
	}
	return vaqfs, rows.Err()
}

func (d *DataService) flattenVersionedAdditionalQuestionFields(vaqfs []*common.VersionedAdditionalQuestionField) (*common.VersionedAdditionalQuestionField, error) {
	if len(vaqfs) == 0 {
		return nil, nil
	}

	jsonMap := make(map[string]interface{})
	for _, field := range vaqfs {
		var innerMap map[string]interface{}
		if err := json.Unmarshal(field.JSON, &innerMap); err != nil {
			return nil, err
		}

		for k, v := range innerMap {
			jsonMap[k] = v
		}
	}

	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}
	vaqf := &common.VersionedAdditionalQuestionField{
		LanguageID: vaqfs[0].LanguageID,
		JSON:       jsonBytes,
	}

	return vaqf, nil
}

// insertVersionedAdditionalQuestionField inserts a json blob additional question field for the question record
func (d *DataService) insertVersionedAdditionalQuestionField(db db, vaqf *common.VersionedAdditionalQuestionField) (int64, error) {
	res, err := db.Exec(`INSERT INTO additional_question_fields (question_id, json, language_id) VALUES (?, ?, ?)`, vaqf.QuestionID, vaqf.JSON, vaqf.LanguageID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) GetQuestionInfo(questionTag string, languageID, version int64) (*info_intake.Question, error) {
	questionInfos, err := d.GetQuestionInfoForTags([]string{questionTag}, languageID)
	if err != nil {
		return nil, err
	} else if len(questionInfos) > 0 {
		return questionInfos[0], nil
	}
	return nil, ErrNotFound("question_info")
}

func (d *DataService) GetQuestionInfoForTags(questionTags []string, languageID int64) ([]*info_intake.Question, error) {
	queries := make([]*QuestionQueryParams, len(questionTags))
	for i, tag := range questionTags {
		version, err := d.MaxQuestionVersion(tag, languageID)
		if err != nil {
			return nil, err
		} else if version == 0 {
			return nil, fmt.Errorf("Could not locate question with tag %v and language id %v", tag, languageID)
		}
		queries[i] = &QuestionQueryParams{
			QuestionTag: tag,
			LanguageID:  languageID,
			Version:     version,
		}
	}
	versionedQuestions, err := d.VersionedQuestions(queries)
	if err != nil {
		return nil, err
	}
	return d.getQuestionInfoForQuestionSet(versionedQuestions, languageID)
}

func (d *DataService) getQuestionInfoForQuestionSet(versionedQuestions []*common.VersionedQuestion, languageID int64) ([]*info_intake.Question, error) {

	var questionInfos []*info_intake.Question
	for _, vq := range versionedQuestions {
		questionInfo := &info_intake.Question{
			QuestionID:             vq.ID,
			QuestionTag:            vq.QuestionTag,
			QuestionTitle:          vq.QuestionText,
			QuestionTitleHasTokens: vq.TextHasTokens,
			QuestionType:           vq.QuestionType,
			QuestionSummary:        vq.SummaryText,
			QuestionSubText:        vq.SubtextText,
			Required:               vq.Required,
			ToAlert:                vq.ToAlert,
			AlertFormattedText:     vq.AlertText,
		}
		if vq.ParentQuestionID != nil {
			questionInfo.ParentQuestionId = *vq.ParentQuestionID
		}
		if vq.FormattedFieldTags != "" {
			questionInfo.FormattedFieldTags = []string{vq.FormattedFieldTags}
		}

		var jsonBytes []byte
		rows, err := d.db.Query(`SELECT json FROM additional_question_fields WHERE question_id = ? AND language_id = ?`, questionInfo.QuestionID, vq.LanguageID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		questionInfo.AdditionalFields = make(map[string]interface{})
		for rows.Next() {
			err = rows.Scan(&jsonBytes)
			if err != nil {
				return nil, err
			}

			var jsonMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
				return nil, err
			}

			for key, value := range jsonMap {
				questionInfo.AdditionalFields[key] = value
			}
		}

		questionInfos = append(questionInfos, questionInfo)
	}

	return questionInfos, nil
}

func (d *DataService) GetAnswerInfo(questionID, languageID int64) ([]*info_intake.PotentialAnswer, error) {
	versionedAnswers, err := d.VersionedAnswers([]*AnswerQueryParams{&AnswerQueryParams{LanguageID: languageID, QuestionID: questionID}})
	if err != nil {
		return nil, err
	}

	return getAnswerInfosFromAnswerSet(versionedAnswers)
}

func (d *DataService) GetAnswerInfoForTags(answerTags []string, languageID int64) ([]*info_intake.PotentialAnswer, error) {

	params := make([]interface{}, 0)
	params = dbutil.AppendStringsToInterfaceSlice(params, answerTags)
	params = append(params, languageID)
	params = append(params, languageID)
	rows, err := d.db.Query(fmt.Sprintf(
		`select id, answer_text, answer_summary_text, answer_type, potential_answer_tag, ordering, to_alert from potential_answer 
									where potential_answer_tag in (%s) and (language_id = ? or answer_text is null) and (language_id = ? or answer_summary_text is null) and status='ACTIVE'`, dbutil.MySQLArgs(len(answerTags))), params...)
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

func createAnswerInfosFromRows(rows *sql.Rows) ([]*info_intake.PotentialAnswer, error) {
	answerInfos := make([]*info_intake.PotentialAnswer, 0)
	for rows.Next() {
		var id, ordering int64
		var answerType, answerTag string
		var answer, answerSummary sql.NullString
		var toAlert sql.NullBool
		err := rows.Scan(&id, &answer, &answerSummary, &answerType, &answerTag, &ordering, &toAlert)
		if err != nil {
			return answerInfos, err
		}
		potentialAnswerInfo := &info_intake.PotentialAnswer{
			Answer:        answer.String,
			AnswerSummary: answerSummary.String,
			AnswerID:      id,
			AnswerTag:     answerTag,
			Ordering:      ordering,
			AnswerType:    answerType,
			ToAlert:       toAlert.Bool,
		}
		answerInfos = append(answerInfos, potentialAnswerInfo)
	}
	return answerInfos, rows.Err()
}

func getAnswerInfosFromAnswerSet(answerSet []*common.VersionedAnswer) ([]*info_intake.PotentialAnswer, error) {
	answerInfos := make([]*info_intake.PotentialAnswer, len(answerSet))
	for i, va := range answerSet {
		answerInfo := &info_intake.PotentialAnswer{
			Answer:        va.AnswerText,
			AnswerSummary: va.AnswerSummaryText,
			AnswerID:      va.ID,
			AnswerTag:     va.AnswerTag,
			Ordering:      va.Ordering,
			AnswerType:    va.AnswerType,
			ToAlert:       va.ToAlert,
		}
		answerInfos[i] = answerInfo
	}
	return answerInfos, nil
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
		var languageID int64
		var language string
		err := rows.Scan(&languageID, &language)
		if err != nil {
			return nil, nil, err
		}
		languagesSupported = append(languagesSupported, language)
		languagesSupportedIds = append(languagesSupportedIds, languageID)
	}
	return languagesSupported, languagesSupportedIds, rows.Err()
}

func (d *DataService) GetPhotoSlots(questionID, languageID int64) ([]*info_intake.PhotoSlot, error) {
	rows, err := d.db.Query(`select photo_slot.id, ltext, slot_type, required from photo_slot
		inner join localized_text on app_text_id = slot_name_app_text_id
		inner join photo_slot_type on photo_slot_type.id = slot_type_id
		where question_id=? and language_id = ? order by ordering`, questionID, languageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photoSlotInfoList := make([]*info_intake.PhotoSlot, 0)
	for rows.Next() {
		var pSlotInfo info_intake.PhotoSlot
		if err := rows.Scan(&pSlotInfo.ID, &pSlotInfo.Name, &pSlotInfo.Type, &pSlotInfo.Required); err != nil {
			return nil, err
		}
		photoSlotInfoList = append(photoSlotInfoList, &pSlotInfo)
	}
	return photoSlotInfoList, rows.Err()
}

func (d *DataService) LatestAppVersionSupported(pathwayID int64, skuID *int64, platform common.Platform, role, purpose string) (*common.Version, error) {
	var version common.Version
	vals := []interface{}{pathwayID, platform.String(), role, purpose}
	whereClause := "clinical_pathway_id = ? AND platform = ? AND role = ? AND purpose = ?"
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
		return nil, ErrNotFound("app_version_layout_mapping")
	}

	return &version, nil
}

type PathwayPurposeVersionMapping map[string]map[string][]*common.Version

func (d *DataService) LayoutVersionMapping() (PathwayPurposeVersionMapping, error) {
	rows, err := d.db.Query(
		`SELECT tag, layout_purpose, major, minor, patch FROM layout_version
			JOIN clinical_pathway ON clinical_pathway.id = clinical_pathway_id 
			WHERE layout_version.status = 'ACTIVE' ORDER BY major, minor, patch ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pathwayMap := make(map[string]map[string][]*common.Version)
	var tag, purpose string
	var major, minor, patch int
	for rows.Next() {
		err := rows.Scan(&tag, &purpose, &major, &minor, &patch)
		if err != nil {
			return nil, err
		}
		purposeMap, ok := pathwayMap[tag]
		if !ok {
			purposeMap = make(map[string][]*common.Version)
			pathwayMap[tag] = purposeMap
		}
		purposeMap[purpose] = append(purposeMap[purpose], &common.Version{Major: major, Minor: minor, Patch: patch})
	}
	return pathwayMap, rows.Err()
}

func (d *DataService) LayoutTemplate(pathway, purpose string, version *common.Version) ([]byte, error) {
	var jsonBytes []byte
	if err := d.db.QueryRow(
		`SELECT layout FROM layout_version
			JOIN layout_blob_storage ON layout_blob_storage.id = layout_version.layout_blob_storage_id
			JOIN clinical_pathway ON layout_version.clinical_pathway_id = clinical_pathway.id WHERE 
			tag = ? AND layout_purpose = ? AND major = ? AND minor = ? AND patch = ?`, pathway, purpose, version.Major, version.Minor, version.Patch).Scan(&jsonBytes); err != nil {
		return nil, err
	}
	return jsonBytes, nil
}
