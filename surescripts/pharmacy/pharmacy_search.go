package pharmacy

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/pharmacy"
	_ "github.com/sprucehealth/backend/third_party/github.com/lib/pq"
)

type surescriptsPharmacySearch struct {
	db *sql.DB
}

type Config struct {
	User     string `long:"db_user" description:"Username for accessing database"`
	Password string `long:"db_password" description:"Password for accessing database"`
	Host     string `long:"db_host" description:"Database host"`
	Port     int    `long:"db_port" description:"Database port"`
	Name     string `long:"db_name" description:"Database name"`
}

const (
	metersInMile = float64(1609)
)

func NewSurescriptsPharmacySearch(config *Config) (*surescriptsPharmacySearch, error) {
	// validate config
	if config.User == "" {
		return nil, errors.New("Username required for database setup")
	} else if config.Host == "" {
		return nil, errors.New("Host required for database setup")
	} else if config.Port == 0 {
		return nil, errors.New("Port required for database setup")
	} else if config.Name == "" {
		return nil, errors.New("Name required for database setup")
	}

	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require", config.User, config.Password, config.Host, config.Port, config.Name))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &surescriptsPharmacySearch{
		db: db,
	}, nil
}

func (s *surescriptsPharmacySearch) GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) ([]*pharmacy.PharmacyData, error) {
	var rows *sql.Rows
	var err error

	// only include pharmacies that are considered retail pharmacies, accept newRx on the surescripts platform, and are currently active
	rows, err = s.db.Query(`SELECT pharmacy.id, pharmacy.ncpdpid, store_name, address_line_1, 
			address_line_2, city, state, zip, phone_primary, fax, pharmacy_location.longitude, pharmacy_location.latitude, specialty, active_end_time FROM pharmacy, pharmacy_location
			WHERE  pharmacy.id = pharmacy_location.id
			AND st_distance(pharmacy_location.geom, st_setsrid(st_makepoint($1,$2),4326)) < $3
			AND service_level & 1 = 1
			ORDER BY pharmacy_location.geom <-> st_setsrid(st_makepoint($1,$2),4326)
			LIMIT $4`, searchLocationLng, searchLocationLat, (searchRadius * metersInMile), numResults)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*pharmacy.PharmacyData
	now := time.Now().UTC()
	for rows.Next() {
		var item pharmacy.PharmacyData
		var specialty int
		var activeEndTime time.Time
		if err := rows.Scan(
			&item.SourceId,
			&item.NCPDPID,
			&item.Name,
			&item.AddressLine1,
			&item.AddressLine2,
			&item.City,
			&item.State,
			&item.Postal,
			&item.Phone,
			&item.Fax,
			&item.Longitude,
			&item.Latitude,
			&specialty,
			&activeEndTime); err != nil {
			return nil, err
		}

		// shortcircuit all pharmcies to map to a single test pharmacy
		// as the database that dosespot is using for their test pharmacies is different
		// than the production pharmacy database, but we want to use the production pharmacy database
		// in non-prod environments to be able to test the pharmacies that the pharmacy db has
		if !environment.IsProd() {
			item.SourceId = 47731
		}

		item.Source = pharmacy.PHARMACY_SOURCE_SURESCRIPTS

		// dont include the pharmacy in the search result if the pharmacy
		// is not a retail pharmacy or is not active
		if specialty&8 != 8 || activeEndTime.Before(now) {
			continue
		}

		results = append(results, sanitizePharmacyData(&item))
	}

	return dedupeOnNCPDPID(results), rows.Err()
}

func (s *surescriptsPharmacySearch) GetPharmacyFromId(pharmacyId int64) (*pharmacy.PharmacyData, error) {
	var item pharmacy.PharmacyData

	if err := s.db.QueryRow(`SELECT pharmacy.id, store_name, address_line_1, address_line_2, city, state, zip, phone_primary, fax, pharmacy_location.longitude, pharmacy_location.latitude 
		FROM pharmacy, pharmacy_location
		WHERE pharmacy.ncpdpid = pharmacy_location.ncpdpid AND id = $1`, pharmacyId).Scan(
		&item.SourceId,
		&item.Name,
		&item.AddressLine1,
		&item.AddressLine2,
		&item.City,
		&item.State,
		&item.Postal,
		&item.Phone,
		&item.Fax,
		&item.Longitude,
		&item.Latitude); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	item.Source = pharmacy.PHARMACY_SOURCE_SURESCRIPTS
	return sanitizePharmacyData(&item), nil
}

// sanitizePharmacyData cleans up the pharmacy data to remove whitespaces
// and correctly capitalize the address
// TODO: Rather than cleaning up data on read, we should clean up data when
// populating the database with data
func sanitizePharmacyData(pharmacy *pharmacy.PharmacyData) *pharmacy.PharmacyData {
	pharmacy.AddressLine1 = trimAndToTitle(pharmacy.AddressLine1)
	pharmacy.AddressLine2 = trimAndToTitle(pharmacy.AddressLine2)
	pharmacy.City = trimAndToTitle(pharmacy.City)
	pharmacy.Name = trimAndToTitle(pharmacy.Name)
	pharmacy.Phone = strings.TrimSpace(pharmacy.Phone)

	// break up the postal code into the zip-plus4 format
	if len(pharmacy.Postal) > 5 {
		pharmacy.Postal = pharmacy.Postal[:5]
	}

	// also remove any storenumbers in the pharmacy name
	pharmacy.Name = removeStoreNumbersFromName(pharmacy.Name)

	return pharmacy
}

func removeStoreNumbersFromName(storeName string) string {
	index := strings.LastIndex(storeName, " ")
	if index == -1 {
		return storeName
	} else if index == len(storeName)-1 {
		return storeName
	} else if index == 0 {
		return storeName
	}

	var lastWord string

	if len(storeName[index+1:]) > 1 && storeName[index+1] == '#' {
		lastWord = storeName[index+2:]
	} else {
		lastWord = storeName[index+1:]
	}

	_, err := strconv.Atoi(lastWord)
	if err == nil {
		return storeName[0:index]
	}

	return storeName
}

func trimAndToTitle(str string) string {
	return strings.Title(strings.ToLower(strings.TrimSpace(str)))
}

// dedupeOnNCPDPID returns results with unique ncpdpid, which uniquely identifies the pharmacy.
// The reason to filter on the result set instead of including the filter as part of the query
// is because the query is a lot slower if we were to get rows with distinct ncpdpid values as
// a sorting on the ncpdpid has to occur which renders the index on the spatial data useless
func dedupeOnNCPDPID(results []*pharmacy.PharmacyData) []*pharmacy.PharmacyData {
	dedupedResults := make([]*pharmacy.PharmacyData, 0, len(results))
	uniqueNCPDPID := make(map[string]bool)
	for _, result := range results {
		if !uniqueNCPDPID[result.NCPDPID] {
			uniqueNCPDPID[result.NCPDPID] = true
			dedupedResults = append(dedupedResults, result)
		}
	}
	return dedupedResults
}
