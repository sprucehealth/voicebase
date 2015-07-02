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

var (
	ErrZeroResults    = errors.New("maps_service: No results returned")
	ErrQuotaExceeded  = errors.New("maps_service: Query Quota exceed")
	ErrRequestDenied  = errors.New("maps_service: Request denied")
	ErrInvalidRequest = errors.New("maps_service: Invalid request possibly due to missing parameters")
	ErrUnknown        = errors.New("maps_service: Unknown error")
)

type MapsService interface {
	ConvertZipcodeToCityState(zipcode string) (*CityStateInfo, error)
	GetLatLongFromSearchLocation(searchLocation string) (*LocationInfo, error)
}
