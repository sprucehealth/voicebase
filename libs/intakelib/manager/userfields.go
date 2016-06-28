package manager

import "fmt"

// keyType defines the only acceptable keys that can be present inside userFields
type keyType string

const (
	// keyTypePatientGender is used by the client to indicate the patient's gender
	keyTypePatientGender keyType = "gender"

	// keyTypeIsPatientPharmacySet is used by the client to indicate whether
	// or not the patient's preferred pharmacy has been set.
	keyTypeIsPatientPharmacySet keyType = "is_pharmacy_set"
)

func (k keyType) String() string {
	return string(k)
}

type userFields struct {
	fields map[string][]byte
}

// set adds the value to its internal map for the specified key if the
// key is supported and the value is an expected one for the given key.
func (u *userFields) set(key string, value []byte) error {
	if u.fields == nil {
		u.fields = make(map[string][]byte)
	}

	switch key {
	case keyTypePatientGender.String():
		switch string(value) {
		case "male", "female", "other":
		default:
			return fmt.Errorf("Unrecognized value for key type `gender`. Only accepted values are male, female and other.")
		}
		u.fields[key] = value
	case keyTypeIsPatientPharmacySet.String():
		switch string(value) {
		case "true", "false":
		default:
			return fmt.Errorf("Unrecognized value for key type `is_pharmacy_set`. Only accepted values are true and false.")
		}
		u.fields[key] = value
	default:
		return fmt.Errorf("Unrecognized key type %s", key)
	}

	return nil
}

func (u *userFields) get(key string) []byte {
	return u.fields[key]
}
