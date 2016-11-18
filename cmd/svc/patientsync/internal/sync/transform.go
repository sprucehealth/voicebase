package sync

import "github.com/sprucehealth/backend/svc/directory"

func TransformDOB(dob *Patient_Date) *directory.Date {
	if dob == nil {
		return nil
	}

	return &directory.Date{
		Day:   dob.Day,
		Month: dob.Month,
		Year:  dob.Year,
	}
}

func TransformGender(gender Patient_Gender) directory.EntityInfo_Gender {
	switch gender {
	case GENDER_MALE:
		return directory.EntityInfo_MALE
	case GENDER_FEMALE:
		return directory.EntityInfo_FEMALE
	case GENDER_OTHER:
		return directory.EntityInfo_OTHER
	}
	return directory.EntityInfo_UNKNOWN
}

// TransformContacts returns the contact information in a patient
// as a slice of directory contact objects
func TransformContacts(patient *Patient) []*directory.Contact {
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
	case PHONE_TYPE_OFFICE:
		return "Office"
	}

	return ""
}
