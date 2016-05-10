package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/care"
)

// TreatmentAvailability represents the OTC vs Rx availability for a treatment
type TreatmentAvailability string

const (
	// TreatmentAvailabilityUnknown means the availability is unknown or undefined
	TreatmentAvailabilityUnknown TreatmentAvailability = "UNKNOWN"
	// TreatmentAvailabilityOTC means the medication is available over-the-counter
	TreatmentAvailabilityOTC TreatmentAvailability = "OTC"
	// TreatmentAvailabilityRx means the medication is by prescription only
	TreatmentAvailabilityRx TreatmentAvailability = "RX"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (ta *TreatmentAvailability) Scan(src interface{}) error {
	if src == nil {
		*ta = TreatmentAvailabilityUnknown
		return nil
	}
	var typ string
	switch v := src.(type) {
	case []byte:
		typ = string(v)
	case string:
		typ = v
	default:
		return errors.Trace(fmt.Errorf("unsupported value for TreatmentAvailability: %T", src))
	}
	*ta = TreatmentAvailability(strings.ToUpper(typ))
	return errors.Trace(ta.Validate())
}

// Value implements sql/driver.Valuer
func (ta TreatmentAvailability) Value() (driver.Value, error) {
	return strings.ToUpper(string(ta)), errors.Trace(ta.Validate())
}

// Validate returns nil iff the value of the type is valid
func (ta TreatmentAvailability) Validate() error {
	switch ta {
	case TreatmentAvailabilityUnknown, TreatmentAvailabilityOTC, TreatmentAvailabilityRx:
		return nil
	}
	return errors.Trace(fmt.Errorf("unknown TreatmentAvailability '%s'", string(ta)))
}

func (ta TreatmentAvailability) String() string {
	return string(ta)
}

// CarePlanID is the ID for a care plan
type CarePlanID struct{ model.ObjectID }

func NewCarePlanID() (CarePlanID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return CarePlanID{}, errors.Trace(err)
	}
	return CarePlanID{
		model.ObjectID{
			Prefix:  care.CarePlanIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseCarePlanID(s string) (CarePlanID, error) {
	t := EmptyCarePlanID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptyCarePlanID() CarePlanID {
	return CarePlanID{
		model.ObjectID{
			Prefix:  care.CarePlanIDPrefix,
			IsValid: false,
		},
	}
}

// CarePlanTreatmentID is the ID for a care plan
type CarePlanTreatmentID struct{ model.ObjectID }

func NewCarePlanTreatmentID() (CarePlanTreatmentID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return CarePlanTreatmentID{}, errors.Trace(err)
	}
	return CarePlanTreatmentID{
		model.ObjectID{
			Prefix:  care.CarePlanTreatmentIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseCarePlanTreatmentID(s string) (CarePlanTreatmentID, error) {
	t := EmptyCarePlanTreatmentID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptyCarePlanTreatmentID() CarePlanTreatmentID {
	return CarePlanTreatmentID{
		model.ObjectID{
			Prefix:  care.CarePlanTreatmentIDPrefix,
			IsValid: false,
		},
	}
}

type CarePlan struct {
	ID           CarePlanID
	Name         string
	Treatments   []*CarePlanTreatment
	Instructions []*CarePlanInstruction
	Created      time.Time
	Submitted    *time.Time
	ParentID     string
	CreatorID    string
}

type CarePlanTreatment struct {
	ID                   CarePlanTreatmentID
	MedicationID         string
	EPrescribe           bool
	Name                 string
	Form                 string
	Route                string
	Availability         TreatmentAvailability
	Dosage               string
	DispenseType         string
	DispenseNumber       int
	Refills              int
	SubstitutionsAllowed bool
	DaysSupply           int
	Sig                  string
	PharmacyID           string
	PharmacyInstructions string
}

type CarePlanInstruction struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}
