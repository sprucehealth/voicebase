package pharmacy

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
)

const (
	distanceBetweenLongitudesInMiles = 69.0
)

var (
	NoPharmacyExists = errors.New("no pharmacy with that id in the database")
)

type PharmacySearchService struct {
	PharmacyDB *sql.DB
}

func (p *PharmacySearchService) GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error) {

	// prepare the coordinates to define the minimum bounding rectangle based on the search radius
	minLng := searchLocationLng - (searchRadius / math.Abs(math.Cos(degreesToRadians(searchLocationLat))*distanceBetweenLongitudesInMiles))
	maxLng := searchLocationLng + (searchRadius / math.Abs(math.Cos(degreesToRadians(searchLocationLat))*distanceBetweenLongitudesInMiles))
	minLat := searchLocationLat - (searchRadius / distanceBetweenLongitudesInMiles)
	maxLat := searchLocationLat + (searchRadius / distanceBetweenLongitudesInMiles)

	rows, err := p.PharmacyDB.Query(`select id, biz_name, e_address, e_city, e_state, e_postal,loc_LAT_centroid, loc_LONG_centroid, biz_phone, biz_fax, web_url from dump_pharmacies 
										where st_within(loc_pt, envelope(linestring(point(?, ?), point(?, ?)))) 
											order by st_distance(point(?,?), loc_pt) limit ?`, minLng, minLat, maxLng, maxLat, searchLocationLng, searchLocationLat, numResults)
	if err != nil {
		return
	}
	defer rows.Close()

	pharmacies = make([]*PharmacyData, 0)
	for rows.Next() {
		pharmacy, shadowedErr := scanPharmacyDataFromRow(rows)
		if shadowedErr != nil {
			shadowedErr = err
			return
		}

		pharmacy.DistanceInMiles = GreatCircleDistanceBetweenTwoPoints(&point{Latitude: pharmacy.Latitude, Longitude: pharmacy.Longitude}, &point{Latitude: searchLocationLat, Longitude: searchLocationLng})

		pharmacy.Source = PHARMACY_SOURCE_ODDITY
		pharmacies = append(pharmacies, pharmacy)
	}

	return
}

func (p *PharmacySearchService) GetPharmacyBasedOnId(pharmacyId string) (pharmacy *PharmacyData, err error) {
	id, err := strconv.Atoi(pharmacyId)
	if err != nil {
		return
	}
	rows, err := p.PharmacyDB.Query(`select id, biz_name, e_address, e_city, e_state, e_postal,loc_LAT_centroid, loc_LONG_centroid, biz_phone, biz_fax, web_url from dump_pharmacies 
										where id = ?`, id)
	if err != nil {
		return
	}
	if rows.Next() {
		pharmacy, err = scanPharmacyDataFromRow(rows)
	}
	return
}

func (p *PharmacySearchService) GetPharmaciesBasedOnTextSearch(textSearch, lat, lng, searchResultInMiles string) (pharmacies []*PharmacyData, err error) {
	return nil, nil
}

func scanPharmacyDataFromRow(rows *sql.Rows) (pharmacy *PharmacyData, err error) {
	var id int64
	var name, address, city, state, postal, lat, lng, phone, fax, url sql.NullString
	err = rows.Scan(&id, &name, &address, &city, &state, &postal, &lat, &lng, &phone, &fax, &url)
	if err != nil {
		return
	}

	pharmacy = &PharmacyData{
		SourceId:     strconv.FormatInt(id, 10),
		Name:         name.String,
		AddressLine1: address.String,
		City:         city.String,
		State:        state.String,
		Postal:       postal.String,
		Phone:        phone.String,
		Fax:          fax.String,
		Url:          url.String,
	}

	if lat.Valid {
		latFloat, _ := strconv.ParseFloat(lat.String, 64)
		pharmacy.Latitude = latFloat
	}

	if lng.Valid {
		lngFloat, _ := strconv.ParseFloat(lng.String, 64)
		pharmacy.Longitude = lngFloat
	}

	return
}

func degreesToRadians(degrees float64) float64 {
	return (math.Pi * degrees / 180.0)
}
