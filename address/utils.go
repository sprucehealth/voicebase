package address

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/common"
)

type addressLookup interface {
	State(state string) (fullName string, abbreviation string, err error)
}

func ValidateAddress(lookup addressLookup, address *common.Address, validator Validator) error {
	fullStateName, _, err := lookup.State(address.State)
	if err != nil {
		return err
	}

	if fullStateName == "" {
		return errors.New("Enter a valid state")
	}

	address.State = fullStateName

	return validateZipcode(address.ZipCode, validator)
}

func validateZipcode(zipcode string, validator Validator) error {

	// first validate format of zipcode
	if err := validateZipcodeLocally(zipcode); err != nil {
		return err
	}

	// then check for existence of zipcode
	_, err := validator.ZipcodeLookup(zipcode)
	if err != nil {
		return fmt.Errorf("Invalid or non-existent zip code")
	}

	return nil
}

func validateZipcodeLocally(zipcode string) error {
	if len(zipcode) < 5 {
		return fmt.Errorf("Invalid zip code: has to be at least 5 digits in length")
	}

	_, err := strconv.ParseInt(zipcode[0:5], 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid zip code: Only digits allowed in zipcode")
	}

	if len(zipcode) > 5 {

		if zipcode[5] != '-' {

			if len(zipcode) != 9 {
				return fmt.Errorf("Invalid zip code format: zip+4 can only be 9 digits in length")
			}

			_, err := strconv.ParseInt(zipcode[5:], 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid zipcode: zip+4 can only have digits after hyphen")
			}

		} else {

			if len(zipcode) != 10 {
				return fmt.Errorf("Invalid zip code format: zip+4 has to be 9 digits in length")
			}

			_, err := strconv.ParseInt(zipcode[5:], 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid zipcode format: Only digits allowed in zip+4")
			}

		}
	}
	return nil
}
