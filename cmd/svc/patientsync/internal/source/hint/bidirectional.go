package hint

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/go-hint"
)

func UpdatePatientIfDiffersFromEntity(patientID string, syncConfig *sync.Config, entity *directory.Entity) error {
	practiceKey := syncConfig.GetHint().AccessToken

	patient, err := hint.GetPatient(practiceKey, patientID)
	if err != nil {
		return errors.Errorf("Unable to get patient %s from hint: %s", patientID, err)
	}

	// nothing to do if the patients do not differ between hint and spruce
	if !sync.Differs(transformPatient(patient, syncConfig), entity) {
		golog.Infof("patient %s and entity %s do not differ so nothing to do here", patientID, entity.ID)
		return nil
	}

	// if the patient in Hint was updated after the patient in Spruce,
	// then ignore the Spruce update, assuming that the information in Hint is
	// the latest information
	if uint64(patient.UpdatedAt.Unix()) > entity.LastModifiedTimestamp {
		golog.Infof("Ignoring the update of entity %s in spruce since the patient %s was updated after that", entity.ID, patientID)
		return nil
	}

	// re-add any phone numbers that are not parseable in Spruce, back to the hint patient
	// so that patient information is not 'lost' on hint side
	var unparseablePhoneNumbers []*hint.Phone
	for _, phoneItem := range patient.Phones {
		if _, err := phone.ParseNumber(phoneItem.Number); err != nil {
			unparseablePhoneNumbers = append(unparseablePhoneNumbers, phoneItem)
		}
	}

	hintPatient := transformEntityToHintPatient(patientID, entity)

	if len(unparseablePhoneNumbers) > 0 {
		hintPatient.Phones = append(hintPatient.Phones, unparseablePhoneNumbers...)
	}

	// pretty format all phone numbers when adding back to hint
	for _, phoneItem := range hintPatient.Phones {
		parsedPhone, err := phone.Format(phoneItem.Number, phone.Pretty)
		if err == nil {
			phoneItem.Number = parsedPhone
		}
	}

	_, err = hint.UpdatePatient(practiceKey, patientID, &hint.PatientParams{
		FirstName: hintPatient.FirstName,
		LastName:  hintPatient.LastName,
		Phones:    hintPatient.Phones,
		Email:     hintPatient.Email,
	})
	if err != nil {
		return errors.Trace(err)
	}

	golog.Infof("patient update in Spruce (%s) triggered an update in Hint (%s)", entity.ID, patientID)
	return errors.Trace(err)
}
