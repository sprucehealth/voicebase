package sync

import (
	"testing"

	"github.com/sprucehealth/backend/svc/directory"
)

func TestDiffers_Same(t *testing.T) {
	patient := &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	entity := &directory.Entity{
		Type: directory.EntityType_PATIENT,
		Info: &directory.EntityInfo{
			FirstName:   "Joe",
			LastName:    "Schmoe",
			DisplayName: "Joe",
			Gender:      directory.EntityInfo_FEMALE,
			DOB: &directory.Date{
				Day:   11,
				Month: 11,
				Year:  2011,
			},
		},
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "joe@schmoe.com",
			},
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
				Label:       "Home",
			},
		},
	}

	if Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}
}

func TestDiffers_Differs(t *testing.T) {
	// email differs
	patient := &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe+updated@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	entity := &directory.Entity{
		Type: directory.EntityType_PATIENT,
		Info: &directory.EntityInfo{
			FirstName:   "Joe",
			LastName:    "Schmoe",
			DisplayName: "Joe",
			Gender:      directory.EntityInfo_FEMALE,
			DOB: &directory.Date{
				Day:   11,
				Month: 11,
				Year:  2011,
			},
		},
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "joe@schmoe.com",
			},
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+11234567890",
				Label:       "Home",
			},
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

	// name differs
	patient = &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe updated",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

	//  number of phone numbers differs
	patient = &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+12222222222",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

	//  gender differs
	patient = &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+12222222222",
			},
		},
		Gender: GENDER_MALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

	//  DOB differs
	patient = &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+11234567890",
			},
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+12222222222",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  1999,
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

	//  existing phone number differs
	patient = &Patient{
		FirstName:      "Joe",
		LastName:       "Schmoe",
		EmailAddresses: []string{"joe@schmoe.com"},
		PhoneNumbers: []*Phone{
			{
				Type:   PHONE_TYPE_HOME,
				Number: "+12222222222",
			},
		},
		Gender: GENDER_FEMALE,
		DOB: &Patient_Date{
			Day:   11,
			Month: 11,
			Year:  2011,
		},
	}

	if !Differs(patient, entity) {
		t.Fatal("Expected no difference in sync properties between patient and entity")
	}

}
