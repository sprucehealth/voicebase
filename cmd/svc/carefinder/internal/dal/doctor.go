package dal

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

type DoctorDAL interface {
	IsDoctorShortListed(id string) (bool, error)
	Doctor(id string) (*models.Doctor, error)
	Doctors(ids []string) ([]*models.Doctor, error)
	SpruceReviews(doctorID string) ([]*models.Review, error)
	StateCoverageForSpruceDoctor(doctorID string) ([]*models.State, error)
	ShortListedStatesForSpruceDoctor(doctorID string) ([]*models.State, error)
	ShortListedDoctorIDs() ([]string, error)
	ShortListedCityClosestToPracticeLocation(doctorID string) (*models.City, error)
}

var (
	ErrNoDoctorFound = errors.New("doctor not found")
)

type doctorDAL struct {
	db *sql.DB
}

func NewDoctorDAL(db *sql.DB) DoctorDAL {
	return &doctorDAL{
		db: db,
	}
}

func (d *doctorDAL) IsDoctorShortListed(id string) (bool, error) {
	var doctorID string

	if err := d.db.QueryRow(`
		SELECT doctor_id 
		FROM doctor_short_list
		WHERE doctor_id = $1`, id).Scan(&doctorID); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}

	return doctorID != "", nil
}

func (d *doctorDAL) Doctor(id string) (*models.Doctor, error) {
	row := d.db.QueryRow(`
		SELECT id, npi, is_spruce_doctor, first_name, last_name, gender,
			average_rating, review_count,
			COALESCE(graduation_year,''),
			COALESCE(medical_school,''), 
			COALESCE(residency,''), 
			COALESCE(specialties_csv,''), 
			COALESCE(profile_image_id,''), 
			COALESCE(description,''), 
			COALESCE(seo_description,''), 
			COALESCE(yelp_url,''),
			COALESCE(yelp_business_id,''), 
			COALESCE(insurance_accepted_csv,''), 
			COALESCE(practice_location_address_line_1,''),
			COALESCE(practice_location_address_line_2,''), 
			COALESCE(practice_location_city,''), 
			COALESCE(practice_location_state,''),
			COALESCE(practice_location_zipcode,''), 
			COALESCE(practice_phone,''), 
			COALESCE(practice_location_lat,0.0), 
			COALESCE(practice_location_lng,0.0),
			COALESCE(referral_code,''),
			COALESCE(referral_link,''),
			COALESCE(spruce_provider_id,0)
		FROM carefinder_doctor_info
		WHERE id = $1`, id)

	doctor, err := d.scanDoctor(row)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoDoctorFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return doctor, nil
}

func (d *doctorDAL) Doctors(ids []string) ([]*models.Doctor, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, npi, is_spruce_doctor, first_name, last_name, gender,
			average_rating, review_count,
			COALESCE(graduation_year,''),
			COALESCE(medical_school,''), 
			COALESCE(residency,''), 
			COALESCE(specialties_csv,''), 
			COALESCE(profile_image_id,''), 
			COALESCE(description,''), 
			COALESCE(seo_description,''), 
			COALESCE(yelp_url,''),
			COALESCE(yelp_business_id,''), 
			COALESCE(insurance_accepted_csv,''), 
			COALESCE(practice_location_address_line_1,''),
			COALESCE(practice_location_address_line_2,''), 
			COALESCE(practice_location_city,''), 
			COALESCE(practice_location_state,''),
			COALESCE(practice_location_zipcode,''), 
			COALESCE(practice_phone,''), 
			COALESCE(practice_location_lat,0.0), 
			COALESCE(practice_location_lng,0.0),
			COALESCE(referral_code,''),
			COALESCE(referral_link,''),
			COALESCE(spruce_provider_id,0)
		FROM carefinder_doctor_info
		WHERE id in (`+dbutil.PostgresArgs(1, len(ids))+`)`, dbutil.AppendStringsToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	doctors := make([]*models.Doctor, len(ids))
	var i int
	for rows.Next() {
		doctor, err := d.scanDoctor(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}

		doctors[i] = doctor
		i++
	}

	return doctors, errors.Trace(rows.Err())
}

func (d *doctorDAL) scanDoctor(scanner dbutil.Scanner) (*models.Doctor, error) {
	var doctor models.Doctor
	var specialtiesCSV, insurancesAcceptedCSV string
	var addressLine1, addressLine2, city, state, zipcode, phone string
	var lat, lng float64

	if err := scanner.Scan(
		&doctor.ID,
		&doctor.NPI,
		&doctor.IsSpruceDoctor,
		&doctor.FirstName,
		&doctor.LastName,
		&doctor.Gender,
		&doctor.AverageRating,
		&doctor.ReviewCount,
		&doctor.GraduationYear,
		&doctor.MedicalSchool,
		&doctor.Residency,
		&specialtiesCSV,
		&doctor.ProfileImageID,
		&doctor.Description,
		&doctor.SEODescription,
		&doctor.YelpURL,
		&doctor.YelpBusinessID,
		&insurancesAcceptedCSV,
		&addressLine1,
		&addressLine2,
		&city,
		&state,
		&zipcode,
		&phone,
		&lat,
		&lng,
		&doctor.ReferralCode,
		&doctor.ReferralLink,
		&doctor.SpruceProviderID); err != nil {
		return nil, errors.Trace(err)
	}

	if addressLine1 != "" {
		doctor.Address = &models.Address{
			AddressLine1: addressLine1,
			AddressLine2: addressLine2,
			City:         city,
			State:        state,
			Zipcode:      zipcode,
			Latitude:     lat,
			Longitude:    lng,
			Phone:        phone,
		}
	}

	if insurancesAcceptedCSV != "" {
		doctor.InsurancesAccepted = strings.Split(insurancesAcceptedCSV, ",")
	}

	if specialtiesCSV != "" {
		doctor.Specialties = strings.Split(specialtiesCSV, "|")
	}

	return &doctor, nil
}

func (d *doctorDAL) SpruceReviews(doctorID string) ([]*models.Review, error) {
	rows, err := d.db.Query(`
		SELECT doctor_id, review, rating, created_date 
		FROM spruce_review
		WHERE doctor_id = $1`, doctorID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var spruceReviews []*models.Review
	for rows.Next() {
		var sr models.Review
		if err := rows.Scan(
			&sr.DoctorID,
			&sr.Text,
			&sr.Rating,
			&sr.CreatedDate); err != nil {
			return nil, errors.Trace(err)
		}

		spruceReviews = append(spruceReviews, &sr)
	}

	return spruceReviews, errors.Trace(rows.Err())
}

func (d *doctorDAL) StateCoverageForSpruceDoctor(doctorID string) ([]*models.State, error) {
	rows, err := d.db.Query(`
		SELECT state.abbreviation,state.full_name
		FROM spruce_doctor_state_coverage
		INNER JOIN carefinder_doctor_info ON carefinder_doctor_info.npi = spruce_doctor_state_coverage.npi
		INNER JOIN state ON state.abbreviation = state_abbreviation
		WHERE carefinder_doctor_info.id = $1`, doctorID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var states []*models.State
	for rows.Next() {
		var state models.State
		if err := rows.Scan(&state.Abbreviation, &state.FullName); err != nil {
			return nil, errors.Trace(err)
		}
		states = append(states, &state)
	}

	return states, errors.Trace(rows.Err())
}

func (d *doctorDAL) ShortListedStatesForSpruceDoctor(doctorID string) ([]*models.State, error) {
	rows, err := d.db.Query(`
		SELECT state.abbreviation,state.full_name, state.key
		FROM spruce_doctor_state_coverage
		INNER JOIN carefinder_doctor_info ON carefinder_doctor_info.npi = spruce_doctor_state_coverage.npi
		INNER JOIN state ON state.abbreviation = state_abbreviation
		WHERE carefinder_doctor_info.id = $1
		AND state.abbreviation IN (SELECT DISTINCT state FROM city_shortlist)`, doctorID)
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

func (d *doctorDAL) ShortListedDoctorIDs() ([]string, error) {
	rows, err := d.db.Query(`
		SELECT doctor_id 
		FROM doctor_short_list`)
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

	return doctorIDs, errors.Trace(err)
}

func (d *doctorDAL) ShortListedCityClosestToPracticeLocation(doctorID string) (*models.City, error) {
	var city models.City
	err := d.db.QueryRow(`
		SELECT c1.id, c1.name, c1.admin1_code, state.full_name, c1.latitude, c1.longitude
		FROM cities c1 
		INNER JOIN city_shortlist ON city_shortlist.city_id = c1.id
		INNER JOIN state ON state.abbreviation = c1.admin1_code
		INNER JOIN carefinder_doctor_info ON carefinder_doctor_info.id = $1
		INNER JOIN business_geocode ON business_geocode.npi = carefinder_doctor_info.npi
		ORDER BY ST_DISTANCE(c1.geom, business_geocode.geom)
		LIMIT 1`, doctorID).Scan(
		&city.ID,
		&city.Name,
		&city.StateAbbreviation,
		&city.State,
		&city.Latitude,
		&city.Longitude)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoCityFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &city, nil
}
