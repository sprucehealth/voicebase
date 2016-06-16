package icd10

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDB struct {
	db *sql.DB
}

func (m *MySQLDB) Connect(host, username, name, password string, port int) error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		username, password, host, port, name))
	if err != nil {
		return err
	}

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	m.db = db

	return nil
}

func (m *MySQLDB) Close() error {
	if m.db == nil {
		return nil
	}

	return m.db.Close()
}

func (m *MySQLDB) SetDiagnoses(diagnoses map[string]*Diagnosis) error {

	tx, err := m.db.Begin()
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
		if err := addDiagnosisToDB(
			diagnosis,
			insertStatement,
			includeNotesStatement,
			inclusionTermsStatement,
			excludes1Statement,
			excludes2Statement,
			useAdditionalCodeStatement,
			codeFirstStatement); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
