package address

import "errors"

var (
	InvalidZipcodeError = errors.New("Invalid Zipcode")
)

type CityState struct {
	City              string
	State             string
	StateAbbreviation string
}

type AddressValidationAPI interface {
	ZipcodeLookup(zipcode string) (*CityState, error)
}
