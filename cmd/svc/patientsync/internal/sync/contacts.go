package sync

import "github.com/sprucehealth/backend/svc/directory"

// ContactsFromPatient returns the contact information in a patient
// as a slice of directory contact objects
func ContactsFromPatient(patient *Patient) []*directory.Contact {
	contacts := make([]*directory.Contact, 0, len(patient.PhoneNumbers)+len(patient.EmailAddresses))
	for _, pn := range patient.PhoneNumbers {
		contacts = append(contacts, &directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       pn.Number,
			Label:       labelFromType(pn.Type),
		})
	}

	for _, ea := range patient.EmailAddresses {
		contacts = append(contacts, &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       ea,
		})
	}

	return contacts
}

func labelFromType(phoneType Phone_PhoneType) string {
	switch phoneType {
	case PHONE_TYPE_MOBILE:
		return "Mobile"
	case PHONE_TYPE_HOME:
		return "Home"
	}

	return ""
}
