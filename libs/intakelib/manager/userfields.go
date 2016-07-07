package manager

import (
	"fmt"
	"strconv"
)

// keyType defines the only acceptable keys that can be present inside userFields
type keyType string

const (
	// keyTypePatientGender is used by the client to indicate the patient's gender
	keyTypePatientGender keyType = "gender"

	// keyTypeIsPatientPharmacySet is used by the client to indicate whether
	// or not the patient's preferred pharmacy has been set.
	keyTypeIsPatientPharmacySet keyType = "is_pharmacy_set"

	// keyTypePatientAgeInYears is used by the client to indicate the patient's age in years
	keyTypePatientAgeInYears keyType = "age_in_years"
)

func (k keyType) String() string {
	return string(k)
}

type userFields struct {
	fields map[string]interface{}
}

// set adds the value to its internal map for the specified key if the
// key is supported and the value is an expected one for the given key.
func (u *userFields) set(key string, value string) error {
	if u.fields == nil {
		u.fields = make(map[string]interface{})
	}

	switch key {
	case keyTypePatientGender.String():
		switch value {
		case "male", "female", "other":
		default:
			return fmt.Errorf("Unrecognized value for key type `gender`. Only accepted values are male, female and other.")
		}
		u.fields[key] = value
	case keyTypeIsPatientPharmacySet.String():
		switch value {
		case "true", "false":
		default:
			return fmt.Errorf("Unrecognized value for key type %s. Only accepted values are true and false.", key)
		}
		var err error
		u.fields[key], err = strconv.ParseBool(value)
		if err != nil {
			return err
		}
	case keyTypePatientAgeInYears.String():
		// ensure that the value is an integer
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		u.fields[key] = intValue
	default:
		return fmt.Errorf("Unrecognized key type %s", key)
	}

	return nil
}

func (u *userFields) get(key string) interface{} {
	return u.fields[key]
}
