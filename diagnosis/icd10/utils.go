package icd10

import "database/sql"

func addDiagnosisToDB(
	diagnosis *Diagnosis,
	insertStmt,
	includesNotesStmt,
	inclusionTermStmt,
	excludes1Stmt,
	excludes2Stmt,
	useAdditionalCodeStmt,
	codeFirstStmt *sql.Stmt) error {

	_, err := insertStmt.Exec(
		diagnosis.Code.Key(),
		diagnosis.Code.String(),
		diagnosis.Description,
		diagnosis.Billable)
	if err != nil {
		return err
	}

	if err := addNotes(
		includesNotesStmt,
		diagnosis.Code.Key(),
		diagnosis.Includes); err != nil {
		return err
	}

	if err := addNotes(
		inclusionTermStmt,
		diagnosis.Code.Key(),
		diagnosis.InclusionTerms); err != nil {
		return err
	}

	if err := addNotes(
		excludes1Stmt,
		diagnosis.Code.Key(),
		diagnosis.Excludes1); err != nil {
		return err
	}

	if err := addNotes(
		excludes2Stmt,
		diagnosis.Code.Key(),
		diagnosis.Excludes2); err != nil {
		return err
	}

	if err := addNotes(
		useAdditionalCodeStmt,
		diagnosis.Code.Key(),
		diagnosis.UseAdditionalCode); err != nil {
		return err
	}

	if err := addNotes(
		codeFirstStmt,
		diagnosis.Code.Key(),
		diagnosis.CodeFirst); err != nil {
		return err
	}

	return nil
}

func addNotes(stmt *sql.Stmt, diagnosisID string, notes []string) error {
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
