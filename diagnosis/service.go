package diagnosis

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/dbutil"
)

type Service struct {
	db *sql.DB
}

func NewService(config *config.DB) (API, error) {
	s := &Service{}

	var err error
	s.db, err = config.ConnectPostgres()
	if err != nil {
		return nil, err
	}

	return s, err
}

func (s *Service) DoCodesExist(codeIDs []string) (bool, []string, error) {
	if len(codeIDs) == 0 {
		return false, nil, nil
	}

	rows, err := s.db.Query(`
		SELECT id from diagnosis_code
		WHERE id in (`+dbutil.PostgresArgs(len(codeIDs))+`)`,
		dbutil.AppendStringsToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	existingCodeIDs := make(map[string]bool)
	for rows.Next() {
		var codeID string
		if err := rows.Scan(&codeID); err != nil {
			return false, nil, err
		}
		existingCodeIDs[codeID] = true
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}

	// track codes that don't exist
	nonExistentCodeIDs := make([]string, 0, len(codeIDs))
	for _, codeID := range codeIDs {
		if !existingCodeIDs[codeID] {
			nonExistentCodeIDs = append(nonExistentCodeIDs, codeID)
		}
	}

	return len(nonExistentCodeIDs) == 0, nonExistentCodeIDs, nil
}

func (s *Service) DiagnosisForCodeIDs(codeIDs []string) (map[string]*Diagnosis, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT id, code, name 
		FROM diagnosis_code
		WHERE id in (`+dbutil.PostgresArgs(len(codeIDs))+`)`,
		dbutil.AppendStringsToInterfaceSlice(nil, codeIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	diagnoses := make(map[string]*Diagnosis, len(codeIDs))
	for rows.Next() {
		var diagnosis Diagnosis
		if err := rows.Scan(
			&diagnosis.ID,
			&diagnosis.Code,
			&diagnosis.Description); err != nil {
			return nil, err
		}
		diagnoses[diagnosis.ID] = &diagnosis
	}
	return diagnoses, rows.Err()
}

func (s *Service) SynonymsForDiagnoses(codeIDs []string) (map[string][]string, error) {
	if len(codeIDs) == 0 {
		return nil, nil
	}

	args := dbutil.PostgresArgs(len(codeIDs))
	vals := dbutil.AppendStringsToInterfaceSlice(nil, codeIDs)

	rows, err := s.db.Query(`
		SELECT diagnosis_code_id, note 
		FROM diagnosis_includes_notes
		WHERE diagnosis_code_id in (`+args+`)`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	synonyms := make(map[string][]string)
	if err := buildSynonymsFromRows(rows, synonyms); err != nil {
		return nil, err
	}

	rows, err = s.db.Query(`
		SELECT diagnosis_code_id, note 
		FROM diagnosis_inclusion_term
		WHERE diagnosis_code_id in (`+args+`)`, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := buildSynonymsFromRows(rows, synonyms); err != nil {
		return nil, err
	}

	return synonyms, nil
}

func buildSynonymsFromRows(rows *sql.Rows, synonymMap map[string][]string) error {
	for rows.Next() {
		var codeID, note string
		if err := rows.Scan(&codeID, &note); err != nil {
			return err
		}

		if _, ok := synonymMap[codeID]; !ok {
			synonymMap[codeID] = make([]string, 0)
		}

		synonymMap[codeID] = append(synonymMap[codeID], note)
	}
	return rows.Err()
}

func (s *Service) SearchDiagnosesByCode(query string, numResults int) ([]*Diagnosis, error) {
	if len(query) == 0 || numResults == 0 {
		return nil, nil
	}

	// change query to also look for any diagnoses
	// that start with the provided query
	query += ":*"

	rows, err := s.db.Query(`
		SELECT id, code, name 
		FROM diagnosis_code 
		WHERE billable = true 
		AND to_tsquery('english', $1) @@ to_tsvector('english', code) 
		ORDER BY code
		LIMIT $2`, query, numResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDiagnosisFromRows(rows)
}

func (s *Service) SearchDiagnoses(query string, numResults int) ([]*Diagnosis, error) {
	if len(query) == 0 || numResults == 0 {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT did, code, name 
		FROM diagnosis_search_index
		WHERE document @@ plainto_tsquery('english', $1)
		AND billable = true 
		ORDER BY ts_rank(document, plainto_tsquery('english', $1)) DESC
		LIMIT $2
		`, query, numResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDiagnosisFromRows(rows)
}

func (s *Service) FuzzyTextSearchDiagnoses(query string, numResults int) ([]*Diagnosis, error) {
	if len(query) == 0 || numResults == 0 {
		return nil, nil
	}

	// find unique lexemes that fuzzy match based on the query
	rows, err := s.db.Query(`
		SELECT word 
		FROM diagnosis_unique_lexeme
		WHERE word % $1
		ORDER BY similarity(word, $1) DESC`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(words) == 0 {
		return nil, nil
	}

	// use the words identified to search for any matching diagnoses
	rows, err = s.db.Query(`
		SELECT did, code, name 
		FROM diagnosis_search_index
		WHERE document @@ to_tsquery('english', $1) 
		AND billable = true
		ORDER BY ts_rank(document, to_tsquery('english', $1)) DESC
		LIMIT $2
		`, strings.Join(words, " | "), numResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDiagnosisFromRows(rows)
}

func scanDiagnosisFromRows(rows *sql.Rows) ([]*Diagnosis, error) {
	var diagnoses []*Diagnosis
	for rows.Next() {
		var diagnosis Diagnosis
		if err := rows.Scan(
			&diagnosis.ID,
			&diagnosis.Code,
			&diagnosis.Description); err != nil {
			return nil, err
		}
		diagnoses = append(diagnoses, &diagnosis)
	}

	return diagnoses, rows.Err()
}
