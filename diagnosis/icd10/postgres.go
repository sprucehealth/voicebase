package icd10

import (
	"database/sql"
	"fmt"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

func (p *PostgresDB) Connect(host, username, name, password string, port int) error {
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", username, password, host, port, name))
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	p.db = db

	return nil
}

func (p *PostgresDB) Close() error {
	if p.db == nil {
		return nil
	}

	return p.db.Close()
}

func (p *PostgresDB) SetDiagnoses(diagnoses map[string]*Diagnosis) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}

	// create prepare statements
	insertStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_code 
		(id, code, name, billable) 
		VALUES ($1, $2, $3, $4)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer insertStatement.Close()

	includeNotesStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_includes_notes 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer includeNotesStatement.Close()

	inclusionTermsStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_inclusion_term 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer inclusionTermsStatement.Close()

	excludes1Statement, err := tx.Prepare(`
		INSERT INTO diagnosis_excludes1_note 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer excludes1Statement.Close()

	excludes2Statement, err := tx.Prepare(`
		INSERT INTO diagnosis_excludes2_note 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer excludes2Statement.Close()

	useAdditionalCodeStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_use_additional_code_note 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer useAdditionalCodeStatement.Close()

	codeFirstStatement, err := tx.Prepare(`
		INSERT INTO diagnosis_code_first_note 
		(diagnosis_code_id, note)
		VALUES ($1,$2)`)
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
			codeFirstStatement,
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
