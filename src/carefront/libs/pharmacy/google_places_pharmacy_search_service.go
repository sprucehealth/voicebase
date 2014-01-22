package pharmacy

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type googlePlacesResultItem struct {
	Geometry             *googlePlacesLocation           `json:"geometry"`
	AddressComponents    []*googlePlacesAddressComponent `json:"address_components"`
	FormattedAddress     string                          `json:"formatted_address"`
	Id                   string                          `json:"id"`
	Name                 string                          `json:"name"`
	Reference            string                          `json:"reference"`
	FormattedPhoneNumber string                          `json:"formatted_phone_number"`
	Vicinity             string                          `json:"vicinity"`
}

const (
	city         = "locality"
	state        = "administrative_area_level_1"
	country      = "country"
	zipCode      = "postal_code"
	streetName   = "route"
	streetNumber = "street_number"
)

type googlePlacesAddressComponent struct {
	LongName string   `json:"long_name"`
	Types    []string `json:"types"`
}

type googlePlacesLocation struct {
	Location *point `json:"location"`
}

type googlePlacesResult struct {
	Results []*googlePlacesResultItem `json:"results"`
	Result  *googlePlacesResultItem   `json:"result"`
	Status  string                    `json:"status"`
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
		pharmacy := getPharmacyFromResultItem(placesResultItem)
		latFloat, _ := strconv.ParseFloat(pharmacy.Latitude, 64)
		lngFloat, _ := strconv.ParseFloat(pharmacy.Longitude, 64)

		pharmacy.DistanceInMiles = GreatCircleDistanceBetweenTwoPoints(&point{Latitude: searchLocationLat, Longitude: searchLocationLng}, &point{Latitude: latFloat, Longitude: lngFloat})

		pharmacies = append(pharmacies, pharmacy)
	}

	return

}

func (p GooglePlacesPharmacySearchService) GetPharmaciesBasedOnTextSearch(textSearch, lat, lng, searchRadius string) (pharmacies []*PharmacyData, err error) {
	v := url.Values{}
	v.Set("key", googlePlacesKey)
	v.Set("sensor", "false")
	v.Set("types", "pharmacy")
	v.Set("query", textSearch)

	if lat != "" && lng != "" {
		v.Set("location", lat+","+lng)

		if searchRadius != "" {
			v.Set("radius", searchRadius)
		}
	}

	resp, err := http.Get("https://maps.googleapis.com/maps/api/place/textsearch/json?" + v.Encode())
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
		pharmacy := getPharmacyFromResultItem(placesResultItem)
		if lat != "" && lng != "" {
			latFloat, _ := strconv.ParseFloat(lat, 64)
			lngFloat, _ := strconv.ParseFloat(lng, 64)
			pharmacyLatFloat, _ := strconv.ParseFloat(pharmacy.Latitude, 64)
			pharmacyLngFloat, _ := strconv.ParseFloat(pharmacy.Longitude, 64)
			pharmacy.DistanceInMiles = GreatCircleDistanceBetweenTwoPoints(&point{Latitude: latFloat, Longitude: lngFloat}, &point{Latitude: pharmacyLatFloat, Longitude: pharmacyLngFloat})
		}

		pharmacies = append(pharmacies, pharmacy)
	}

	return
}

func getPharmacyFromResultItem(resultItem *googlePlacesResultItem) *PharmacyData {
	pharmacyDetails := &PharmacyData{}
	var streetNameComponent, streetNumberComponent string
	if resultItem.AddressComponents != nil {
		for _, addressComponent := range resultItem.AddressComponents {
			for _, addressType := range addressComponent.Types {
				switch addressType {
				case city:
					pharmacyDetails.City = addressComponent.LongName
				case state:
					pharmacyDetails.State = addressComponent.LongName
				case country:
					pharmacyDetails.Country = addressComponent.LongName
				case zipCode:
					pharmacyDetails.Postal = addressComponent.LongName
				case streetName:
					streetNameComponent = addressComponent.LongName
				case streetNumber:
					streetNumberComponent = addressComponent.LongName
				}
			}
		}
		pharmacyDetails.Address = streetNumberComponent + " " + streetNameComponent
	} else if resultItem.Vicinity != "" {
		pharmacyDetails.Address = resultItem.Vicinity
	} else if resultItem.FormattedAddress != "" {
		pharmacyDetails.Address = resultItem.FormattedAddress
	}

	pharmacyDetails.Phone = resultItem.FormattedPhoneNumber
	pharmacyDetails.Name = resultItem.Name
	pharmacyDetails.Id = resultItem.Reference
	pharmacyDetails.Latitude = strconv.FormatFloat(resultItem.Geometry.Location.Latitude, 'f', -1, 64)
	pharmacyDetails.Longitude = strconv.FormatFloat(resultItem.Geometry.Location.Longitude, 'f', -1, 64)
	return pharmacyDetails

}

func (p GooglePlacesPharmacySearchService) GetPharmacyBasedOnId(pharmacyId string) (pharmacyDetails *PharmacyData, err error) {
	v := url.Values{}
	v.Set("key", googlePlacesKey)
	v.Set("sensor", "false")
	v.Set("reference", pharmacyId)
	// latString := strconv.FormatFloat(searchLocationLat, 'f', -1, 64)
	// lngString := strconv.FormatFloat(searchLocationLng, 'f', -1, 64)
	// v.Set("location", latString+","+lngString)

	resp, err := http.Get("https://maps.googleapis.com/maps/api/place/details/json?" + v.Encode())
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

	if placesResult.Result != nil {
		pharmacyDetails = getPharmacyFromResultItem(placesResult.Result)
	}

	return pharmacyDetails, nil
}
