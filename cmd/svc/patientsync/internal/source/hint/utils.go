package hint

import (
	"encoding/base64"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/go-hint"
)

// byPhoneNumber sorts the hint phone numbers by surfacing the mobile numbers
// to the front (in case of ascending) or by collecting them at the end of the slice
// (in case of descending)
type byPhoneNumber []*hint.Phone

func (b byPhoneNumber) Len() int      { return len(b) }
func (b byPhoneNumber) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byPhoneNumber) Less(i, j int) bool {
	// populate all the mobile numbers to the top
	if b[i].Type == hint.PhoneTypeMobile {
		return true
	}
	return false
}

func syncPhoneTypeFromHint(hintPhone *hint.Phone) sync.Phone_PhoneType {
	switch hintPhone.Type {
	case hint.PhoneTypeMobile:
		return sync.PHONE_TYPE_MOBILE
	case hint.PhoneTypeOffice:
		return sync.PHONE_TYPE_OFFICE
	case hint.PhoneTypeHome:
		return sync.PHONE_TYPE_HOME
	}

	return sync.PHONE_TYPE_UNKNOWN
}

func hintPhoneTypeFromEntityLabel(label string) string {
	switch strings.ToLower(label) {
	case "Mobile":
		return hint.PhoneTypeMobile
	case "Office":
		return hint.PhoneTypeOffice
	case "Home":
		return hint.PhoneTypeHome
	}

	return label
}

func transformGender(gender string) sync.Patient_Gender {
	switch gender {
	case "male":
		return sync.GENDER_MALE
	case "female":
		return sync.GENDER_FEMALE
	case "other":
		return sync.GENDER_OTHER
	}
	return sync.GENDER_UNKNOWN
}

func transformDOB(dob string) (*sync.Patient_Date, error) {
	if dob == "" {
		return nil, nil
	}

	parsedDOB, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return nil, errors.Errorf("Unable to transform dob '%s' into object: %s", dob, err)
	}

	return &sync.Patient_Date{
		Day:   uint32(parsedDOB.Day()),
		Month: uint32(parsedDOB.Month()),
		Year:  uint32(parsedDOB.Year()),
	}, nil
}

func transformPatient(hintPatient *hint.Patient) *sync.Patient {
	dob, err := transformDOB(hintPatient.DOB)
	if err != nil {
		golog.Errorf("Unable to transform dob, ignoring: %s", err.Error())
	}
	syncPatient := &sync.Patient{
		ID:        hintPatient.ID,
		FirstName: hintPatient.FirstName,
		LastName:  hintPatient.LastName,
		EmailAddresses: []string{
			hintPatient.Email,
		},
		Gender:           transformGender(hintPatient.Gender),
		DOB:              dob,
		PhoneNumbers:     make([]*sync.Phone, 0, len(hintPatient.Phones)),
		ExternalURL:      hint.PatientURLForProvider(hintPatient.ID),
		CreatedTime:      uint64(hintPatient.CreatedAt.Unix()),
		LastModifiedTime: uint64(hintPatient.UpdatedAt.Unix()),
	}

	// sort the hint phone numbers to surface the mobile phone numbers at the top
	sort.Sort(byPhoneNumber(hintPatient.Phones))
	for _, hintPhone := range hintPatient.Phones {
		syncPatient.PhoneNumbers = append(syncPatient.PhoneNumbers, &sync.Phone{
			Number: hintPhone.Number,
			Type:   syncPhoneTypeFromHint(hintPhone),
		})
	}

	return syncPatient
}

func transformEntityToHintPatient(id string, entity *directory.Entity) *hint.Patient {
	hintPatient := &hint.Patient{
		ID:        id,
		FirstName: entity.Info.FirstName,
		LastName:  entity.Info.LastName,
	}

	// for now don't populate gender and age given that the information is unlikely to change in our database
	// but likely to change in hint
	emails := directory.FilterContacts(entity, directory.ContactType_EMAIL)
	phoneNumbers := directory.FilterContacts(entity, directory.ContactType_PHONE)

	if len(emails) > 0 {
		hintPatient.Email = emails[0].Value
	}

	hintPatient.Phones = make([]*hint.Phone, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		hintPatient.Phones = append(hintPatient.Phones, &hint.Phone{
			Type:   hintPhoneTypeFromEntityLabel(phoneNumber.Label),
			Number: phoneNumber.Value,
		})
	}

	return hintPatient
}

func createSyncEvent(
	orgID, syncEventsQueueURL string,
	patients []*sync.Patient,
	sqsAPI sqsiface.SQSAPI) error {
	syncEvent := &sync.Event{
		Source:               sync.SOURCE_HINT,
		OrganizationEntityID: orgID,
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: patients,
			},
		},
	}
	data, err := syncEvent.Marshal()
	if err != nil {
		return errors.Trace(err)
	}
	msg := base64.StdEncoding.EncodeToString(data)
	if _, err := sqsAPI.SendMessage(&sqs.SendMessageInput{
		MessageBody: &msg,
		QueueUrl:    &syncEventsQueueURL,
	}); err != nil {
		return errors.Trace(err)
	}
	return nil
}
