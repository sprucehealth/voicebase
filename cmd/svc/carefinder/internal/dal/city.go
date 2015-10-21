package dal

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

type CityDAL interface {
	IsCityShortListed(id string) (bool, error)
	City(id string) (*models.City, error)
	BannerImageIDsForCity(id string) ([]string, error)
	BannerImageIDsForState(state string) ([]string, error)
	LocalDoctorIDsForCity(cityID string) ([]string, error)
	SpruceDoctorIDsForCity(cityID string) ([]string, error)
	CareRatingForCity(cityID string) (*models.CareRating, error)
	TopSkinConditionsForCity(cityID string, n int) ([]string, error)
	NearbyCitiesForCity(cityID string, n int) ([]*models.City, error)
	StateShortList() ([]*models.State, error)
}

var (
	ErrNoCityFound        = errors.New("city not found")
	ErrNoBannerImageFound = errors.New("banner image not found")
	ErrNoCareRatingFound  = errors.New("care rating for city not found")
)

type cityDAL struct {
	db *sql.DB
}

func NewCityDAL(db *sql.DB) CityDAL {
	return &cityDAL{
		db: db,
	}
}

func (c *cityDAL) IsCityShortListed(id string) (bool, error) {
	var cityID string
	if err := c.db.QueryRow(`
		SELECT city_id 
		FROM city_shortlist
		WHERE city_id = $1`, id).Scan(&cityID); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}
	return cityID != "", nil
}

func (c *cityDAL) City(id string) (*models.City, error) {
	var city models.City
	if err := c.db.QueryRow(`
		SELECT cities.id, name, admin1_code, state.full_name, latitude, longitude
		FROM cities
		INNER JOIN state ON state.abbreviation = admin1_code
		WHERE cities.id = $1`, id).Scan(
		&city.ID,
		&city.Name,
		&city.StateAbbreviation,
		&city.State,
		&city.Latitude,
		&city.Longitude,
	); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoCityFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &city, nil
}

func (c *cityDAL) BannerImageIDsForCity(id string) ([]string, error) {
	// first lets try to get banner images for the city
	rows, err := c.db.Query(`
		SELECT image_id 
		FROM banner_image
		WHERE city_id = $1`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var imageIDs []string
	for rows.Next() {
		var imageID string
		if err := rows.Scan(&imageID); err != nil {
			return nil, errors.Trace(err)
		}
		imageIDs = append(imageIDs, imageID)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	if len(imageIDs) > 0 {
		return imageIDs, nil
	}

	// if no image was found for the city, then fall back
	// to banner images for the state
	rows2, err := c.db.Query(`
		SELECT image_id
		FROM banner_image
		INNER JOIN cities ON cities.id = $1
		WHERE admin1_code = banner_image.state
		AND cities.id = $1`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var imageID string
		if err := rows2.Scan(&imageID); err != nil {
			return nil, errors.Trace(err)
		}
		imageIDs = append(imageIDs, imageID)
	}

	return imageIDs, errors.Trace(rows2.Err())
}

func (c *cityDAL) BannerImageIDsForState(state string) ([]string, error) {
	rows, err := c.db.Query(`
		SELECT image_id
		FROM banner_image
		INNER JOIN state ON state.abbreviation = banner_image.state
		WHERE state.abbreviation = $1`, state)
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

func (c *cityDAL) LocalDoctorIDsForCity(cityID string) ([]string, error) {
	rows, err := c.db.Query(`
		SELECT dcsl.doctor_id
		FROM doctor_city_short_list dcsl
		INNER JOIN doctor_short_list dsl ON dsl.doctor_id = dcsl.doctor_id
		WHERE city_id = $1
		AND dsl.is_spruce_doctor = false`, cityID)
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

func (c *cityDAL) SpruceDoctorIDsForCity(cityID string) ([]string, error) {
	rows, err := c.db.Query(`
		SELECT sdsc.doctor_id
		FROM spruce_doctor_state_coverage sdsc
		INNER JOIN cities ON cities.admin1_code = state_abbreviation
		INNER JOIN doctor_short_list dsl ON dsl.doctor_id = sdsc.doctor_id
		WHERE cities.id = $1
		AND dsl.is_spruce_doctor = true`, cityID)
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

func (c *cityDAL) CareRatingForCity(cityID string) (*models.CareRating, error) {
	var careRating models.CareRating
	var bulletsPipeSeparated string
	if err := c.db.QueryRow(`
		SELECT grade, title, bullets 
		FROM care_rating 
		WHERE city_id = $1`, cityID).Scan(
		&careRating.Rating,
		&careRating.Title,
		&bulletsPipeSeparated); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoCareRatingFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	careRating.Bullets = strings.Split(bulletsPipeSeparated, "|")

	return &careRating, nil
}

func (c *cityDAL) TopSkinConditionsForCity(cityID string, n int) ([]string, error) {
	rows, err := c.db.Query(`
		SELECT bucket
		FROM top_skin_conditions_by_state
		INNER JOIN cities ON cities.admin1_code = state
		WHERE cities.id = $1
		ORDER BY count DESC
		LIMIT $2`, cityID, n)
	if err != nil {
		return nil, errors.Trace(err)
	}

	defer rows.Close()

	var skinConditions []string
	for rows.Next() {
		var condition string
		if err := rows.Scan(&condition); err != nil {
			return nil, errors.Trace(err)
		}
		skinConditions = append(skinConditions, condition)
	}

	return skinConditions, errors.Trace(rows.Err())
}

func (c *cityDAL) NearbyCitiesForCity(cityID string, n int) ([]*models.City, error) {
	rows, err := c.db.Query(`
		SELECT c1.id, c1.name, c1.admin1_code, state.full_name, c1.latitude, c1.longitude
		FROM cities c1
		INNER JOIN cities c2 ON c2.id = $1 
		INNER JOIN city_shortlist ON city_shortlist.city_id = c1.id
		INNER JOIN state ON state.abbreviation = c1.admin1_code
		WHERE c1.country_code = 'US' and c1.feature_code like '%PPL%'
		AND c1.population > 20000
		AND ST_DWITHIN(c1.geog, c2.geog, 48270)
		AND c1.id != c2.id 
		ORDER BY ST_DISTANCE(c1.geom, c2.geom)
		LIMIT $2`, cityID, n)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var cities []*models.City
	for rows.Next() {
		var city models.City
		if err := rows.Scan(
			&city.ID,
			&city.Name,
			&city.StateAbbreviation,
			&city.State,
			&city.Latitude,
			&city.Longitude); err != nil {
			return nil, errors.Trace(err)
		}

		cities = append(cities, &city)
	}

	return cities, errors.Trace(rows.Err())
}

func (c *cityDAL) StateShortList() ([]*models.State, error) {
	rows, err := c.db.Query(`
		SELECT state.abbreviation, state.full_name
		FROM city_shortlist
		INNER JOIN state ON state.abbreviation = city_shortlist.state`)
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
