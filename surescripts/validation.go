package surescripts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
)

// following constants are defined by surescripts requirements
const (
	maxLongFieldLength             = 35
	maxShortFieldLength            = 10
	MaxPharmacyNotesLength         = 210
	MaxPatientInstructionsLength   = 140
	MaxNumberRefillsMaxValue       = 99
	MaxDaysSupplyMaxValue          = 999
	MaxRefillRequestCommentLength  = 70
	MaxMedicationDescriptionLength = 105
)

func ValidatePatientInformation(patient *common.Patient, addressValidator address.Validator, dataAPI api.DataAPI) error {
	if patient.FirstName == "" {
		return errors.New("First name is required")
	}

	if patient.LastName == "" {
		return errors.New("Last name is required")
	}

	if patient.DOB.Month == 0 || patient.DOB.Year == 0 || patient.DOB.Day == 0 {
		return errors.New("DOB is invalid. Please enter in right format")
	}

	if patient.PatientAddress == nil {
		return errors.New("Patient address is required")
	}

	if patient.PatientAddress.AddressLine1 == "" {
		return errors.New("AddressLine1 of address is required")
	}

	if patient.PatientAddress.City == "" {
		return errors.New("City in address is required")
	}

	if patient.PatientAddress.State == "" {
		return errors.New("State in address is required")
	}

	if len(patient.Prefix) > maxShortFieldLength {
		return fmt.Errorf("Prefix cannot be longer than %d characters in length", maxShortFieldLength)
	}

	if len(patient.Suffix) > maxShortFieldLength {
		return fmt.Errorf("Suffix cannot be longer than %d characters in length", maxShortFieldLength)
	}

	if len(patient.FirstName) > maxLongFieldLength {
		return fmt.Errorf("First name cannot be longer than %d characters", maxLongFieldLength)
	}

	if len(patient.MiddleName) > maxLongFieldLength {
		return fmt.Errorf("Middle name cannot be longer than %d characters", maxLongFieldLength)
	}

	if len(patient.LastName) > maxLongFieldLength {
		return fmt.Errorf("Last name cannot be longer than %d characters", maxLongFieldLength)
	}

	if err := ValidateAddress(patient.PatientAddress, addressValidator, dataAPI); err != nil {
		return err
	}

	if len(patient.PhoneNumbers) == 0 {
		return errors.New("Atleast one phone number is required")
	}

	for _, phoneNumber := range patient.PhoneNumbers {
		if err := phoneNumber.Phone.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func ValidateAddress(a *common.Address, addressValidator address.Validator, dataAPI api.DataAPI) error {
	if len(a.AddressLine1) > maxLongFieldLength {
		return fmt.Errorf("Address line 1 must be %d characters or less", maxLongFieldLength)
	}

	if len(a.AddressLine2) > maxLongFieldLength {
		return fmt.Errorf("Address line 2 must be %d characters or less", maxLongFieldLength)
	}

	if len(a.City) > maxLongFieldLength {
		return fmt.Errorf("City must be %d characters or less", maxLongFieldLength)
	}

	return address.ValidateAddress(dataAPI, a, addressValidator)
}

func TrimSpacesFromPatientFields(patient *common.Patient) {
	patient.FirstName = strings.TrimSpace(patient.FirstName)
	patient.LastName = strings.TrimSpace(patient.LastName)
	patient.MiddleName = strings.TrimSpace(patient.MiddleName)
	patient.Suffix = strings.TrimSpace(patient.Suffix)
	patient.Prefix = strings.TrimSpace(patient.Prefix)
	patient.PatientAddress.AddressLine1 = strings.TrimSpace(patient.PatientAddress.AddressLine1)
	patient.PatientAddress.AddressLine2 = strings.TrimSpace(patient.PatientAddress.AddressLine2)
	patient.PatientAddress.City = strings.TrimSpace(patient.PatientAddress.City)
	patient.PatientAddress.State = strings.TrimSpace(patient.PatientAddress.State)
}
