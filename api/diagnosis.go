package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) DiagnosesThatHaveDetails(codeIDs []string) (map[string]bool, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT diagnosis_code_id
		FROM diagnosis_details_layout
		WHERE diagnosis_code_id in (`+dbutil.MySQLArgs(len(codeIDs))+`)`,
		dbutil.AppendStringsToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codeIDsWithIntake := make(map[string]bool)
	for rows.Next() {
		var codeID string
		if err := rows.Scan(&codeID); err != nil {
			return nil, err
		}
		codeIDsWithIntake[codeID] = true
	}

	return codeIDsWithIntake, rows.Err()
}

func (d *DataService) LayoutVersionIDsForDiagnosisCodes(codes map[string]*common.Version) (map[string]int64, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	queryStatement, err := d.db.Prepare(`
		SELECT id
		FROM diagnosis_details_layout
		WHERE diagnosis_code_id = ?
		AND major = ? AND minor = ? AND patch = ?`)
	if err != nil {
		return nil, err
	}
	defer queryStatement.Close()

	layoutVersionIDs := make(map[string]int64, len(codes))
	for codeID, version := range codes {
		if version == nil {
			return nil, fmt.Errorf("no version specified for codeID %s", codeID)
		}

		var id int64
		if err := queryStatement.QueryRow(
			codeID,
			version.Major,
			version.Minor,
			version.Patch).Scan(&id); err == sql.ErrNoRows {
			return nil, ErrNotFound("diagnosis_details_layout")
		} else if err != nil {
			return nil, err
		}
		layoutVersionIDs[codeID] = id
	}

	return layoutVersionIDs, nil
}

func (d *DataService) ActiveDiagnosisDetailsIntakeVersion(codeID string) (*common.Version, error) {
	var version common.Version
	err := d.db.QueryRow(`
		SELECT major, minor, patch
		FROM diagnosis_details_layout_template
		WHERE diagnosis_code_id = ? and active=1`, codeID).Scan(
		&version.Major,
		&version.Minor,
		&version.Patch)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("diagnosis_details_layout_template")
	} else if err != nil {
		return nil, err
	}
	return &version, nil
}

func (d *DataService) ActiveDiagnosisDetailsIntake(codeID string, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error) {
	row := d.db.QueryRow(`
		SELECT dq.id, type, layout, diagnosis_code_id, major, minor, patch, active, created
		FROM diagnosis_details_layout as dq
		WHERE diagnosis_code_id = ? AND active = 1`, codeID)
	return scanDiagnosisDetailsIntake(row, types)
}

func (d *DataService) DetailsIntakeVersionForDiagnoses(codeIDs []string) (map[string]*common.Version, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT diagnosis_code_id, major, minor, patch
		FROM diagnosis_details_layout
		WHERE active = 1 
		AND diagnosis_code_id in (`+dbutil.MySQLArgs(len(codeIDs))+`)	
		`, dbutil.AppendStringsToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	layoutVersions := make(map[string]*common.Version)
	for rows.Next() {
		var version common.Version
		var codeID string
		if err := rows.Scan(
			&codeID,
			&version.Major,
			&version.Minor,
			&version.Patch); err != nil {
			return nil, err
		}

		layoutVersions[codeID] = &version
	}

	return layoutVersions, rows.Err()
}

func (d *DataService) DiagnosisDetailsIntake(ids []int64, types map[string]reflect.Type) (map[int64]*common.DiagnosisDetailsIntake, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, type, layout, diagnosis_code_id, major, minor, patch, active, created
		FROM diagnosis_details_layout as dq
		WHERE dq.id in (`+dbutil.MySQLArgs(len(ids))+`)`, dbutil.AppendInt64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	intakes := make(map[int64]*common.DiagnosisDetailsIntake)
	for rows.Next() {
		var intake common.DiagnosisDetailsIntake
		intake.Version = &common.Version{}
		var intakeType string
		var intakeLayout []byte

		if err := rows.Scan(
			&intake.ID,
			&intakeType,
			&intakeLayout,
			&intake.CodeID,
			&intake.Version.Major,
			&intake.Version.Minor,
			&intake.Version.Patch,
			&intake.Active,
			&intake.Created); err != nil {
			return nil, err
		}
		dataType, ok := types[intakeType]
		if !ok {
			return nil, fmt.Errorf("Unable to find the diagnosis question type for type %s", intakeType)
		}
		intake.Layout = reflect.New(dataType).Interface().(common.Typed)
		if err := json.Unmarshal(intakeLayout, intake.Layout); err != nil {
			return nil, err
		}
		intakes[intake.ID] = &intake
	}

	return intakes, rows.Err()
}

func scanDiagnosisDetailsIntake(row *sql.Row, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error) {
	var dqi common.DiagnosisDetailsIntake
	dqi.Version = &common.Version{}
	var dqiType string
	var dqiData []byte

	if err := row.Scan(
		&dqi.ID,
		&dqiType,
		&dqiData,
		&dqi.CodeID,
		&dqi.Version.Major,
		&dqi.Version.Minor,
		&dqi.Version.Patch,
		&dqi.Active,
		&dqi.Created); err == sql.ErrNoRows {
		return nil, ErrNotFound("diagnosis_details_layout")
	} else if err != nil {
		return nil, err
	}

	dataType, ok := types[dqiType]
	if !ok {
		return nil, fmt.Errorf("Unable to find the diagnosis question type for type %s", dqiType)
	}

	dqi.Layout = reflect.New(dataType).Interface().(common.Typed)
	if dqiData != nil {
		if err := json.Unmarshal(dqiData, dqi.Layout); err != nil {
			return nil, err
		}
	}

	return &dqi, nil
}

func (d *DataService) SetDiagnosisDetailsIntake(template, info *common.DiagnosisDetailsIntake) error {
	// validation
	if template.CodeID == "" {
		return errors.New("CodeID not specified")
	}
	if template.Version.IsZero() {
		return errors.New("Code version not specified")
	}
	if template.Layout == nil {
		return errors.New("template layout not specified")
	}
	if info.CodeID != template.CodeID {
		return errors.New("CodeIDs between template and info does not match")
	}
	if !info.Version.Equals(template.Version) {
		return errors.New("Version between template and info does not match")
	}
	if info.Layout == nil {
		return errors.New("info layout not specified")
	}

	templateData, err := json.Marshal(template.Layout)
	if err != nil {
		return err
	}

	infoData, err := json.Marshal(info.Layout)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// inactivate any preexisting template layouts for the diagnosis
	_, err = tx.Exec(`
		UPDATE diagnosis_details_layout_template 
		SET active = 0 
		WHERE diagnosis_code_id = ? AND active = 1`, template.CodeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	res, err := tx.Exec(`
		INSERT INTO diagnosis_details_layout_template 
		(type, layout, major, minor, patch, diagnosis_code_id, active)
		VALUES (?,?,?,?,?,?,?)`,
		template.Layout.TypeName(),
		templateData,
		template.Version.Major,
		template.Version.Minor,
		template.Version.Patch,
		template.CodeID,
		true)
	if err != nil {
		tx.Rollback()
		return err
	}

	template.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// inactive any preexisting info layout for the diagnosis
	_, err = tx.Exec(`
		UPDATE diagnosis_details_layout
		SET active = 0
		WHERE diagnosis_code_id = ? AND active = 1`, template.CodeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	res, err = tx.Exec(`
		INSERT INTO diagnosis_details_layout
		(type, layout, template_layout_id, major, minor, patch, diagnosis_code_id, active)
		VALUES (?,?,?,?,?,?,?,?)`,
		info.Layout.TypeName(),
		infoData,
		template.ID,
		info.Version.Major,
		info.Version.Minor,
		info.Version.Patch,
		template.CodeID, true)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CommonDiagnosisSet(pathwayTag string) (string, []string, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return "", nil, err
	}

	// get the title of the common diagnosis set
	var title string
	if err := d.db.QueryRow(`
		SELECT title FROM common_diagnosis_set
		WHERE pathway_id = ?`, pathwayID).
		Scan(&title); err == sql.ErrNoRows {
		return "", nil, ErrNotFound("common_diagnosis_set")
	} else if err != nil {
		return "", nil, err
	}

	rows, err := d.db.Query(`
		SELECT diagnosis_code_id 
		FROM common_diagnosis_set_item 
		WHERE pathway_id = ?
		AND active = 1`, pathwayID)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()

	var diagnosisCodeIDs []string
	for rows.Next() {
		var diagnosisCodeID string
		if err := rows.Scan(&diagnosisCodeID); err != nil {
			return "", nil, err
		}
		diagnosisCodeIDs = append(diagnosisCodeIDs, diagnosisCodeID)
	}

	return title, diagnosisCodeIDs, rows.Err()
}

func (d *DataService) PatchCommonDiagnosisSet(pathwayTag string, patch *DiagnosisSetPatch) error {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if patch.Title != nil && *patch.Title != "" {
		_, err = tx.Exec(`
			INSERT INTO common_diagnosis_set (title, pathway_id)
			VALUES (?,?)
			ON DUPLICATE KEY UPDATE title = ?`, *patch.Title, pathwayID, *patch.Title)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(patch.Delete) > 0 {
		vals := dbutil.AppendStringsToInterfaceSlice(nil, patch.Delete)
		vals = append(vals, pathwayID)
		_, err = tx.Exec(`
			DELETE FROM common_diagnosis_set_item
			WHERE diagnosis_code_id in (`+dbutil.MySQLArgs(len(patch.Delete))+`)
			AND pathway_id = ?`, vals...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(patch.Create) > 0 {
		inserts := dbutil.MySQLMultiInsert(len(patch.Create))
		for _, createItem := range patch.Create {
			inserts.Append(createItem, true, pathwayID)
		}
		_, err = tx.Exec(`INSERT INTO common_diagnosis_set_item (diagnosis_code_id, active, pathway_id) VALUES `+inserts.Query(),
			inserts.Values()...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
