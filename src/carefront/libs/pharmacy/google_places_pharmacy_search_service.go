package pharmacy

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type googlePlacesResultItem struct {
	Geometry *googlePlacesLocation `json:"geometry"`
	Id       string                `json:"id"`
	Name     string                `json:"name"`
	Vicinity string                `json:"vicinity"`
}

type googlePlacesLocation struct {
	Location *point `json:"location"`
}

type googlePlacesResult struct {
	Results []*googlePlacesResultItem `json:"results"`
}

const (
	googlePlacesKey = "AIzaSyBI8mQSwAI7053UFC2TaTrc0axkpdiJ0Mk"
)

type GooglePlacesPharmacySearchService int

func (p GooglePlacesPharmacySearchService) GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error) {
	v := url.Values{}
	v.Set("key", googlePlacesKey)
	v.Set("sensor", "false")
	v.Set("types", "pharmacy")
	v.Set("radius", "3200")
	latString := strconv.FormatFloat(searchLocationLat, 'f', -1, 64)
	lngString := strconv.FormatFloat(searchLocationLng, 'f', -1, 64)
	v.Set("location", latString+","+lngString)

	resp, err := http.Get("https://maps.googleapis.com/maps/api/place/nearbysearch/json?" + v.Encode())
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	placesResult := &googlePlacesResult{}
	err = json.Unmarshal(body, placesResult)
	if err != nil {
		return
	}

	// go through results and populate pharmacy data
	pharmacies = make([]*PharmacyData, 0)
	for _, placesResultItem := range placesResult.Results {
		pharmacy := &PharmacyData{}
		pharmacy.Name = placesResultItem.Name
		pharmacy.Address = placesResultItem.Vicinity
		pharmacy.Latitude = strconv.FormatFloat(placesResultItem.Geometry.Location.Latitude, 'f', -1, 64)
		pharmacy.Longitude = strconv.FormatFloat(placesResultItem.Geometry.Location.Longitude, 'f', -1, 64)
		latFloat, _ := strconv.ParseFloat(pharmacy.Latitude, 64)
		lngFloat, _ := strconv.ParseFloat(pharmacy.Longitude, 64)

		pharmacy.DistanceInMiles = GreatCircleDistanceBetweenTwoPoints(&point{Latitude: searchLocationLat, Longitude: searchLocationLng}, &point{Latitude: latFloat, Longitude: lngFloat})

		pharmacies = append(pharmacies, pharmacy)
	}

	return

}

func (p GooglePlacesPharmacySearchService) GetPharmacyBasedOnId(pharmacyId string) (pharmacy *PharmacyData, err error) {
	return nil, nil
}
