package maps

import (
	"errors"
)

type CityStateInfo struct {
	LongStateName  string
	ShortStateName string
	LongCityName   string
	ShortCityName  string
}

type LocationInfo struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

var ZeroResultsErr = errors.New("maps_service: No results returned")
var QuotaExceededErr = errors.New("maps_service: Query Quota exceed")
var RequestDeniedErr = errors.New("maps_service: Request denied")
var InvalidRequestErr = errors.New("maps_service: Invalid request possibly due to missing parameters")
var UnknownError = errors.New("maps_service: Unknown error")

type MapsService interface {
	ConvertZipcodeToCityState(zipcode string) (cityStateInfo CityStateInfo, err error)
	GetLatLongFromSearchLocation(searchLocation string) (locationInfo LocationInfo, err error)
}
