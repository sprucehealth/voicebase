package hint

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
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
	if !sync.Differs(transformPatient(patient), entity) {
		return nil
	}

	hintPatient := transformEntityToHintPatient(patientID, entity)

	_, err = hint.UpdatePatient(practiceKey, patientID, &hint.PatientParams{
		FirstName: hintPatient.FirstName,
		LastName:  hintPatient.LastName,
		Phones:    hintPatient.Phones,
		Email:     hintPatient.Email,
	})
	if err != nil {
		return errors.Trace(err)
	}

	golog.Debugf("patient update in Spruce (%s) triggered an update in Hint (%s)", entity.ID, patientID)
	return errors.Trace(err)
}
