package maps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type GeocodingResult struct {
	Results []*AddressLookup `json:"results"`
	Status  string           `json:"status"`
}

type AddressLookup struct {
	AddressComponents []*AddressComponent `json:"address_components"`
	FormattedAddress  string              `json:"formatted_address"`
	AddressGeometry   *AddressGeometry    `json:"geometry,omitempty"`
	Types             []string            `json:"types"`
}

type AddressGeometry struct {
	Location *LocationInfo `json:"location"`
}

type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

type GoogleMapsService int

func (g GoogleMapsService) ConvertZipcodeToCityState(zipcode string) (cityStateInfo CityStateInfo, err error) {
	queryStr := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&sensor=false", zipcode)
	resp, err := http.Get(queryStr)
	if err != nil {
		return
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	geocodingResult := &GeocodingResult{}
	err = json.Unmarshal(respData, geocodingResult)
	if err != nil {
		return
	}

	switch geocodingResult.Status {
	case "ZERO_RESULTS":
		err = ZeroResultsErr
		return
	case "OVER_QUERY_LIMIT":
		err = QuotaExceededErr
		return
	case "REQUEST_DENIED":
		err = RequestDeniedErr
		return
	case "INVALID_REQUEST":
		err = InvalidRequestErr
		return
	case "UNKNOWN_ERROR":
		err = UnknownError
		return
	}

	// look through the address components to find the ones that relate to the city level and the state level
	cityStateInfo = CityStateInfo{}
	for _, result := range geocodingResult.Results {
		for _, addressComponent := range result.AddressComponents {
			for _, addressComponentType := range addressComponent.Types {
				switch addressComponentType {
				case "administrative_area_level_1":
					cityStateInfo.LongStateName = addressComponent.LongName
					cityStateInfo.ShortStateName = addressComponent.ShortName
				case "locality":
					cityStateInfo.LongCityName = addressComponent.LongName
					cityStateInfo.ShortCityName = addressComponent.ShortName
				}
			}
		}
	}
	return
}

func (g GoogleMapsService) GetLatLongFromSearchLocation(searchLocation string) (locationInfo LocationInfo, err error) {
	v := url.Values{}
	v.Set("address", searchLocation)
	v.Set("sensor", "false")
	queryStr := fmt.Sprintf(`https://maps.googleapis.com/maps/api/geocode/json?%s`, v.Encode())
	resp, err := http.Get(queryStr)
	if err != nil {
		return
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	geocodingResult := &GeocodingResult{}
	err = json.Unmarshal(respData, geocodingResult)
	if err != nil {
		return
	}

	switch geocodingResult.Status {
	case "ZERO_RESULTS":
		err = ZeroResultsErr
		return
	case "OVER_QUERY_LIMIT":
		err = QuotaExceededErr
		return
	case "REQUEST_DENIED":
		err = RequestDeniedErr
		return
	case "INVALID_REQUEST":
		err = InvalidRequestErr
		return
	case "UNKNOWN_ERROR":
		err = UnknownError
		return
	}

	locationInfo = LocationInfo{}
	locationInfo.Latitude = geocodingResult.Results[0].AddressGeometry.Location.Latitude
	locationInfo.Longitude = geocodingResult.Results[0].AddressGeometry.Location.Longitude

	return

}
