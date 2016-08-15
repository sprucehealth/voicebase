package service

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
)

// sanitizeAndValidatePatient cleans out any invalid contact information
func sanitizePatient(patient *sync.Patient) {
	phoneNumbers := make([]*sync.Phone, 0, len(patient.PhoneNumbers))
	for _, phoneNumber := range patient.PhoneNumbers {
		if pn, err := phone.ParseNumber(phoneNumber.Number); err == nil {
			phoneNumbers = append(phoneNumbers, &sync.Phone{
				Type:   phoneNumber.Type,
				Number: pn.String(),
			})
		}
	}
	patient.PhoneNumbers = phoneNumbers

	emailAddresses := make([]string, 0, len(patient.EmailAddresses))
	for _, emailAddress := range patient.EmailAddresses {
		if validate.Email(emailAddress) {
			emailAddresses = append(emailAddresses, emailAddress)
		}
	}
	patient.EmailAddresses = emailAddresses

	return
}
