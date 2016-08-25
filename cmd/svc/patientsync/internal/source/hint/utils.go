package hint

import (
	"encoding/base64"
	"sort"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
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

func transformPatient(hintPatient *hint.Patient) *sync.Patient {
	syncPatient := &sync.Patient{
		ID:        hintPatient.ID,
		FirstName: hintPatient.FirstName,
		LastName:  hintPatient.LastName,
		EmailAddresses: []string{
			hintPatient.Email,
		},
		PhoneNumbers: make([]*sync.Phone, 0, len(hintPatient.Phones)),
		ExternalURL:  hint.PatientURLForProvider(hintPatient.ID),
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

func createSyncEvent(orgID, syncEventsQueueURL string, patients []*sync.Patient, sqsAPI sqsiface.SQSAPI) error {
	syncEvent := &sync.Event{
		Type:                 sync.EVENT_TYPE_PATIENT_ADD,
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
