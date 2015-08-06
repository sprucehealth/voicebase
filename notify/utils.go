package notify

import "github.com/sprucehealth/backend/common"

// ByCommunicationPrefernce represents a sorting utility to sort communication preferences
// in the following order of preference: PUSH, SMS, EMAIL
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
		if phoneNumber.Type == common.PNTCell {
			return patient.PhoneNumbers[0].Phone.String()
		}
	}
	return ""
}

// snsNotification represents the generic structure for sending notifications via sns
// Amazon sns requires us to indicate when sending push notifications to APNS_SANDBOX
// vs APNS vs GCM which is why there are individual variables to represent these objects
type snsNotification struct {
	DefaultMessage string               `json:"default"`
	IOSSandBox     *iOSPushNotification `json:"APNS_SANDBOX,omitempty"`
	IOS            *iOSPushNotification `json:"APNS,omitempty"`
	Android        string               `json:"GCM,omitempty"`
}

type iOSPushNotification struct {
	Alert string `json:"alert,omitempty"`
	Badge int64  `json:"badge,omitempty"`
}

type androidPushData struct {
	Message string `json:"message"`
	PushID  string `json:"push_id"`
}
type androidPushNotification struct {
	Data androidPushData `json:"data"`
}
