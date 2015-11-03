package dal

import (
	"database/sql"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

var (
	ErrNoStateFound = errors.New("state not found")
)

type StateDAL interface {
	IsStateShortListed(id string) (bool, error)
	State(key string) (*models.State, error)
	StateShortList() ([]*models.State, error)
	SpruceDoctorIDsForState(stateKey string) ([]string, error)
	BannerImageIDsForState(state string) ([]string, error)
}

type stateDAL struct {
	db *sql.DB
}

func NewStateDAL(db *sql.DB) StateDAL {
	return &stateDAL{
		db: db,
	}
}

func (s *stateDAL) IsStateShortListed(id string) (bool, error) {
	var sID string
	if err := s.db.QueryRow(`
		SELECT state
		FROM city_shortlist
		INNER JOIN state ON state.abbreviation = city_shortlist.state
		WHERE state.key = $1`, id).Scan(&sID); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}

	return sID != "", nil
}

func (s *stateDAL) State(key string) (*models.State, error) {
	var st models.State
	if err := s.db.QueryRow(`
		SELECT key, abbreviation, full_name
		FROM state 
		WHERE key = $1`, key).Scan(
		&st.Key,
		&st.Abbreviation,
		&st.FullName); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoStateFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &st, nil
}

func (s *stateDAL) StateShortList() ([]*models.State, error) {
	rows, err := s.db.Query(`
		SELECT state.abbreviation, state.full_name, state.key
		FROM state
		WHERE abbreviation in (SELECT DISTINCT state from city_shortlist)`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var states []*models.State
	for rows.Next() {
		var state models.State
		if err := rows.Scan(&state.Abbreviation, &state.FullName, &state.Key); err != nil {
			return nil, errors.Trace(err)
		}

		states = append(states, &state)
	}

	return states, errors.Trace(rows.Err())
}

func (s *stateDAL) BannerImageIDsForState(state string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT image_id
		FROM banner_image
		INNER JOIN state ON state.abbreviation = banner_image.state
		WHERE state.abbreviation = $1
		ORDER BY image_id`, state)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var imageIDs []string
	for rows.Next() {
		var imageID string
		if rows.Scan(&imageID); err != nil {
			return nil, errors.Trace(err)
		}
		imageIDs = append(imageIDs, imageID)
	}

	return imageIDs, errors.Trace(rows.Err())
}

func (s *stateDAL) SpruceDoctorIDsForState(stateAbbreviation string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT dsl.doctor_id
		FROM doctor_short_list dsl
		INNER JOIN spruce_doctor_state_coverage sdsc ON sdsc.doctor_id = dsl.doctor_id
		WHERE state_abbreviation = $1`, stateAbbreviation)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var doctorIDs []string
	for rows.Next() {
		var doctorID string
		if err := rows.Scan(&doctorID); err != nil {
			return nil, errors.Trace(err)
		}
		doctorIDs = append(doctorIDs, doctorID)
	}

	return doctorIDs, errors.Trace(rows.Err())
}
