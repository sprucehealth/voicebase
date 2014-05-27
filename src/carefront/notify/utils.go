package notify

import (
	"carefront/api"
	"carefront/common"
)

type ByCommunicationPreference []*common.CommunicationPreference

func (b ByCommunicationPreference) Len() int      { return len(b) }
func (b ByCommunicationPreference) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByCommunicationPreference) Less(i, j int) bool {

	if b[i].CommunicationType == common.Push {
		return false
	}

	if b[j].CommunicationType == common.Push {
		return true
	}

	if b[i].CommunicationType == common.SMS {
		return false
	}

	if b[j].CommunicationType == common.SMS {
		return true
	}

	return true
}

func phoneNumberForPatient(patient *common.Patient) string {
	for _, phoneNumber := range patient.PhoneNumbers {
		if phoneNumber.PhoneType == api.PHONE_CELL {
			return patient.PhoneNumbers[0].Phone
		}
	}
	return ""
}
