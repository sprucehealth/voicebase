package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/libs/golog"
)

type geocodingWorker struct {
	clientID      string
	clientSecret  string
	db            *sql.DB
	geoClient     *arcgisClient
	consulService *consul.Service
}

type pharmacy struct {
	id           int64
	ncpdpid      string
	addressLine1 string
	addressLine2 string
	city         string
	state        string
	zip          string
}

func (g *geocodingWorker) start() {

	g.geoClient = &arcgisClient{
		clientID:     g.clientID,
		clientSecret: g.clientSecret,
	}

	lock := g.consulService.NewLock("service/pharmacydb/geocoding", nil)

	go func() {
		defer lock.Release()
		for {
			if !lock.Wait() {
				return
			}

			if err := g.geoClient.getAccessToken(); err != nil {
				golog.Errorf(err.Error())
			}

			if err := g.geocodeAddressesForActivePharmacies(); err != nil {
				golog.Errorf(err.Error())
			}

			time.Sleep(24 * time.Hour)
		}
	}()
}

func (g *geocodingWorker) geocodeAddressesForActivePharmacies() error {
	for {
		if numGeocoded, err := g.batchGeocodeAddresses(); err != nil {
			golog.Errorf(err.Error())
		} else if numGeocoded == 0 {
			return nil
		}
		time.Sleep(10 * time.Second)
	}
}

func (g *geocodingWorker) batchGeocodeAddresses() (int, error) {

	// identify active pharmacies that accept new rx's that have not been geocoded
	rows, err := g.db.Query(
		`SELECT id 
		 FROM (SELECT id FROM pharmacy WHERE MOD(service_level, 2) = 1) AS s1 
		 EXCEPT (SELECT id FROM pharmacy_location) 
		 LIMIT 30;`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var pharmacyIds []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		pharmacyIds = append(pharmacyIds, id)
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	params := make([]string, len(pharmacyIds))
	vals := make([]interface{}, len(pharmacyIds))
	for i, pharmacyId := range pharmacyIds {
		params[i] = "$" + strconv.FormatInt(int64(i+1), 10)
		vals[i] = pharmacyId
	}

	// get pharmacies
	rows, err = g.db.Query(`
		SELECT id, ncpdpid, address_line_1, address_line_2, city, state, zip
		FROM pharmacy where id in (`+strings.Join(params, ",")+`)`, vals...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	pharmacies := make([]*pharmacy, len(pharmacyIds))
	pharmacyMap := make(map[int64]*pharmacy, len(pharmacyIds))
	var i int64
	for rows.Next() {
		var pItem pharmacy
		if err := rows.Scan(
			&pItem.id,
			&pItem.ncpdpid,
			&pItem.addressLine1,
			&pItem.addressLine2,
			&pItem.city,
			&pItem.state,
			&pItem.zip); err != nil {
			return 0, err
		}
		pharmacies[i] = &pItem
		pharmacyMap[pItem.id] = &pItem
		i++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	addresses := make([]*address, len(pharmacies))
	for i, pharmacyItem := range pharmacies {
		addresses[i] = &address{
			ObjectID: pharmacyItem.id,
			Address:  pharmacyItem.addressLine1 + " " + pharmacyItem.addressLine2,
			City:     pharmacyItem.city,
			Region:   pharmacyItem.state,
			Postal:   pharmacyItem.zip,
		}
	}

	result, err := g.geoClient.geocodeAddresses(addresses)
	if err != nil {
		return 0, err
	}

	tx, err := g.db.Begin()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO pharmacy_location (id, ncpdpid, latitude, longitude, source, precision)
		VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	for _, resulItem := range result.Locations {
		pItem := pharmacyMap[resulItem.Attributes.ResultID]
		if _, err := stmt.Exec(resulItem.Attributes.ResultID,
			pItem.ncpdpid,
			resulItem.Location.Y,
			resulItem.Location.X,
			"ersi",
			resulItem.Score); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// update the geometric data
	_, err = tx.Exec(`
		UPDATE pharmacy_location 
		SET geom = ST_GeomFromText('POINT(' || longitude || ' ' || latitude || ')',4326) 
		WHERE geom is NULL;`)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return len(pharmacies), tx.Commit()
}
