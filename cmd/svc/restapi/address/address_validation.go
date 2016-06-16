package address

import "errors"

var (
	ErrInvalidZipcode = errors.New("Invalid Zipcode")
)

type CityState struct {
	City              string
	State             string
	StateAbbreviation string
}

type Validator interface {
	ZipcodeLookup(zipcode string) (*CityState, error)
}
