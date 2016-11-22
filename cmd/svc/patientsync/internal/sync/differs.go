package sync

import (
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
)

// Differs returns true if there are sync related properties that
// differ between the patient and entity objects
func Differs(patient *Patient, entity *directory.Entity) bool {
	if patient == nil || entity == nil {
		return true
	}

	if patient.FirstName != entity.Info.FirstName {
		return true
	}

	if patient.LastName != entity.Info.LastName {
		return true
	}

	emailContacts := directory.FilterContacts(entity, directory.ContactType_EMAIL)
	if len(emailContacts) != len(patient.EmailAddresses) {
		return true
	}

	phoneContacts := directory.FilterContacts(entity, directory.ContactType_PHONE)
	if len(phoneContacts) != len(patient.PhoneNumbers) {
		return true
	}

	// check all email addresses are identical
	for _, emailAddress := range patient.EmailAddresses {
		if emailContact := contactForValue(emailAddress, directory.ContactType_EMAIL, emailContacts); emailContact == nil {
			return true
		}
	}

	// check all phone numbers and their labels are identical
	for _, phone := range patient.PhoneNumbers {
		if phoneContact := contactForValue(phone.Number, directory.ContactType_PHONE, phoneContacts); phoneContact == nil {
			return true
		} else if labelFromType(phone.Type) != phoneContact.Label {
			return true
		}
	}

	if patient.DOB != nil {
		if entity.Info.DOB == nil {
			return true
		}

		if *TransformDOB(patient.DOB) != *entity.Info.DOB {
			return true
		}
	}

	if TransformGender(patient.Gender) != entity.Info.Gender {
		return true
	}

	return false
}

func contactForValue(value string, contactType directory.ContactType, contacts []*directory.Contact) *directory.Contact {

	var err error
	if contactType == directory.ContactType_PHONE {
		value, err = phone.Format(value, phone.E164)
		if err != nil {
			return nil
		}
	}

	for _, contact := range contacts {
		contactValue := contact.Value
		if contact.ContactType == directory.ContactType_PHONE {
			contactValue, err = phone.Format(contact.Value, phone.E164)
			if err != nil {
				continue
			}
		}
		if value == contactValue {
			return contact
		}
	}
	return nil
}
