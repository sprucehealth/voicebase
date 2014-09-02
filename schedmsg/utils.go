package schedmsg

import (
	"strings"

	"github.com/sprucehealth/backend/common"
)

const (
	patientNameTag          = "[Patient.FirstName]"
	providerShortDisplayTag = "[Provider.ShortDisplayName]"
)

func fillInTags(message string, patient *common.Patient, doctor *common.Doctor) string {
	msg := strings.Replace(message, patientNameTag, patient.FirstName, -1)
	return strings.Replace(msg, providerShortDisplayTag, doctor.ShortDisplayName, -1)
}
