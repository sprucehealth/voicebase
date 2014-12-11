package icd10

import "database/sql"

func SetDiagnoses(db *sql.DB, diagnoses map[string]*Diagnosis) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// create prepare statements
	insertStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_code (code, name, billable) 
		VALUES (?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer insertStatement.Close()

	includeNotesStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_includes_note (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer includeNotesStatement.Close()

	inclusionTermsStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_inclusion_term (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer inclusionTermsStatement.Close()

	excludes1Statement, err := tx.Prepare(`
		INSERT INTO diagnosis_excludes1_note (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer excludes1Statement.Close()

	excludes2Statement, err := tx.Prepare(`
		INSERT INTO diagnosis_excludes2_note (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer excludes2Statement.Close()

	useAdditionalCodeStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_use_additional_code_note (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer useAdditionalCodeStatement.Close()

	codeFirstStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_code_first_note (diagnosis_code_id, note) 
			VALUES (?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer codeFirstStatement.Close()

	for _, diagnosis := range diagnoses {
		res, err := insertStatement.Exec(diagnosis.Code, diagnosis.Description, diagnosis.Billable)
		if err != nil {
			tx.Rollback()
			return err
		}

		diagnosisID, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			includeNotesStatement,
			diagnosisID,
			diagnosis.Includes); err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			inclusionTermsStatement,
			diagnosisID,
			diagnosis.InclusionTerms); err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			excludes1Statement,
			diagnosisID,
			diagnosis.Excludes1); err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			excludes2Statement,
			diagnosisID,
			diagnosis.Excludes2); err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			useAdditionalCodeStatement,
			diagnosisID,
			diagnosis.UseAdditionalCode); err != nil {
			tx.Rollback()
			return err
		}

		if err := addNotes(
			codeFirstStatement,
			diagnosisID,
			diagnosis.CodeFirst); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func addNotes(stmt *sql.Stmt, diagnosisID int64, notes []string) error {
	// add the rest of the information if it exists
	if len(notes) == 0 {
		return nil
	}

	for _, note := range notes {
		_, err := stmt.Exec(diagnosisID, note)
		if err != nil {
			return err
		}
	}

	return nil
}
