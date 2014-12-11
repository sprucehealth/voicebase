package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) DoDiagnosisCodesExist(codes []string) (bool, []string, error) {
	if len(codes) == 0 {
		return false, nil, nil
	}

	rows, err := d.db.Query(`
		SELECT code from diagnosis_code where code in (`+nReplacements(len(codes))+`)`,
		appendStringsToInterfaceSlice(nil, codes)...)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	existingDiagnosisSet := make(map[string]bool)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return false, nil, err
		}
		existingDiagnosisSet[code] = true
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}

	// track the codes that don't exist
	nonExistentCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		if !existingDiagnosisSet[code] {
			nonExistentCodes = append(nonExistentCodes, code)
		}
	}

	return len(nonExistentCodes) == 0, nonExistentCodes, nil
}

func (d *DataService) DiagnosisForCodeIDs(codeIDs []int64) (map[int64]*common.Diagnosis, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, code, name
		FROM diagnosis_code
		WHERE id in (`+nReplacements(len(codeIDs))+`)`, appendInt64sToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codes := make(map[int64]*common.Diagnosis)
	for rows.Next() {
		var diagnosis common.Diagnosis
		if err := rows.Scan(
			&diagnosis.ID,
			&diagnosis.Code,
			&diagnosis.Description); err != nil {
			return nil, err
		}
		codes[diagnosis.ID] = &diagnosis
	}

	return codes, rows.Err()
}

func (d *DataService) DiagnosisForCodes(codes []string) (map[string]*common.Diagnosis, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, code, name
		FROM diagnosis_code
		WHERE code in (`+nReplacements(len(codes))+`)`, appendStringsToInterfaceSlice(nil, codes)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	diagnosisMap := make(map[string]*common.Diagnosis)
	for rows.Next() {
		var diagnosis common.Diagnosis
		if err := rows.Scan(
			&diagnosis.ID,
			&diagnosis.Code,
			&diagnosis.Description); err != nil {
			return nil, err
		}
		diagnosisMap[diagnosis.Code] = &diagnosis
	}

	return diagnosisMap, rows.Err()
}

func (d *DataService) LayoutVersionIDsForDiagnosisCodes(codes map[int64]*common.Version) (map[int64]int64, error) {
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

	layoutVersionIDs := make(map[int64]int64, len(codes))
	for codeID, version := range codes {
		if version == nil {
			return nil, fmt.Errorf("no version specified for codeID %d", codeID)
		}

		var id int64
		if err := queryStatement.QueryRow(
			codeID,
			version.Major,
			version.Minor,
			version.Patch).Scan(&id); err == sql.ErrNoRows {
			return nil, NoRowsError
		} else if err != nil {
			return nil, err
		}
		layoutVersionIDs[codeID] = id
	}

	return layoutVersionIDs, nil
}

func (d *DataService) ActiveDiagnosisDetailsIntakeVersion(code string) (*common.Version, error) {
	var version common.Version
	err := d.db.QueryRow(`
		SELECT major, minor, patch
		FROM diagnosis_details_layout_template
		INNER JOIN diagnosis_code on diagnosis_code.id = diagnosis_code_id
		WHERE code = ? and active=1`, code).Scan(
		&version.Major,
		&version.Minor,
		&version.Patch)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &version, nil
}

func diagnosisCodeID(d db, code string) (int64, error) {
	var id int64
	if err := d.QueryRow(`
		SELECT id FROM diagnosis_code
		WHERE code = ?
		FOR UPDATE`, code).Scan(&id); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}

	return id, nil
}

func (d *DataService) ActiveDiagnosisDetailsIntake(codeID int64, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error) {
	row := d.db.QueryRow(`
		SELECT dq.id, type, layout, diagnosis_code_id, major, minor, patch, active, created
		FROM diagnosis_details_layout as dq
		WHERE diagnosis_code_id = ? AND active = 1`, codeID)
	return scanDiagnosisDetailsIntake(row, types)
}

func (d *DataService) DetailsIntakeVersionForDiagnoses(codeIDs []int64) (map[int64]*common.Version, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT diagnosis_code_id, major, minor, patch
		FROM diagnosis_details_layout
		WHERE active = 1 
		AND diagnosis_code_id in (`+nReplacements(len(codeIDs))+`)	
		`, appendInt64sToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	layoutVersions := make(map[int64]*common.Version)
	for rows.Next() {
		var version common.Version
		var codeID int64
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
		WHERE dq.id in (`+nReplacements(len(ids))+`)`, appendInt64sToInterfaceSlice(nil, ids)...)
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
		return nil, NoRowsError
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
	if template.Code == "" {
		return errors.New("Code not specified")
	}
	if template.Version.IsZero() {
		return errors.New("Code version not specified")
	}
	if template.Layout == nil {
		return errors.New("template layout not specified")
	}
	if info.Code != template.Code {
		return errors.New("Code between template and info does not match")
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

	codeID, err := diagnosisCodeID(tx, template.Code)
	if err == NoRowsError {
		tx.Rollback()
		return errors.New("Code does not exist")
	} else if err != nil {
		tx.Rollback()
		return err
	}

	// inactivate any preexisting template layouts for the diagnosis
	_, err = tx.Exec(`
		UPDATE diagnosis_details_layout_template 
		SET active = 0 
		WHERE diagnosis_code_id = ? AND active = 1`, codeID)
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
		codeID,
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
		WHERE diagnosis_code_id = ? AND active = 1`, codeID)
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
		codeID, true)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
