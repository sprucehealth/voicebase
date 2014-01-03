package pharmacy

import (
	"database/sql"
	"math"
	"strconv"
)

const (
	distanceBetweenLongitudesInMiles = 69.0
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
		var id int64
		var name, address, city, state, postal, lat, lng, phone, fax, url sql.NullString
		err = rows.Scan(&id, &name, &address, &city, &state, &postal, &lat, &lng, &phone, &fax, &url)
		if err != nil {
			return
		}

		pharmacy := &PharmacyData{}
		pharmacy.Id = id

		if name.Valid {
			pharmacy.Name = name.String
		}

		if address.Valid {
			pharmacy.Address = address.String
		}

		if city.Valid {
			pharmacy.City = city.String
		}

		if state.Valid {
			pharmacy.State = state.String
		}

		if postal.Valid {
			pharmacy.Postal = postal.String
		}

		if lat.Valid {
			pharmacy.Latitude = lat.String
		}

		if lng.Valid {
			pharmacy.Longitude = lng.String
		}

		if phone.Valid {
			pharmacy.Phone = phone.String
		}

		if fax.Valid {
			pharmacy.Fax = fax.String
		}

		if url.Valid {
			pharmacy.Url = url.String
		}

		latFloat, _ := strconv.ParseFloat(pharmacy.Latitude, 64)
		lngFloat, _ := strconv.ParseFloat(pharmacy.Longitude, 64)

		pharmacy.DistanceInMiles = GreatCircleDistanceBetweenTwoPoints(&point{Latitude: latFloat, Longitude: lngFloat}, &point{Latitude: searchLocationLat, Longitude: searchLocationLng})
		pharmacies = append(pharmacies, pharmacy)
	}

	return
}

func degreesToRadians(degrees float64) float64 {
	return (math.Pi * degrees / 180.0)
}
