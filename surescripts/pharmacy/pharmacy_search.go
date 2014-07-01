package pharmacy

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (s *surescriptsPharmacySearch) GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*pharmacy.PharmacyData, err error) {
	// only include pharmacies that have the lowest order bit set for the service level as that indicates pharmacies that have NewRX capabilities
	rows, err := s.db.Query(`SELECT id, ncpdpid, store_name, address_line_1, address_line_2, city, state, zip, phone_primary, fax, longitude, latitude FROM pharmacy
		WHERE st_distance(geom, st_setsrid(st_makepoint($1,$2),4326)) < $3
			AND mod(service_level, 2) = 1
			ORDER BY geom <-> st_setsrid(st_makepoint($1,$2),4326)
			LIMIT $4`, searchLocationLng, searchLocationLat, (searchRadius * metersInMile), numResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*pharmacy.PharmacyData
	for rows.Next() {
		var item pharmacy.PharmacyData
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
			&item.Latitude); err != nil {
			return nil, err
		}
		item.Source = pharmacy.PHARMACY_SOURCE_SURESCRIPTS
		results = append(results, sanitizePharmacyData(&item))
	}

	return dedupeOnNCPDPID(results), rows.Err()
}

func (s *surescriptsPharmacySearch) GetPharmacyFromId(pharmacyId int64) (*pharmacy.PharmacyData, error) {
	var item pharmacy.PharmacyData
	if err := s.db.QueryRow(`SELECT id, store_name, address_line_1, address_line_2, city, state, zip, phone_primary, fax, longitude, latitude FROM pharmacy
		WHERE id = $1`, pharmacyId).Scan(
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

	// break up the postal code into the zip-plus4 format
	if len(pharmacy.Postal) > 5 {
		var postalCode bytes.Buffer
		postalCode.WriteString(pharmacy.Postal[:5])
		postalCode.WriteString("-")
		postalCode.WriteString(pharmacy.Postal[5:])
		pharmacy.Postal = postalCode.String()
	}

	return pharmacy
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
