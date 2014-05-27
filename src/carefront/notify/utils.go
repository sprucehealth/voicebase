package notify

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"reflect"
	"sort"
)

type byCommunicationPreference []*common.CommunicationPreference

func (b byCommunicationPreference) Len() int      { return len(b) }
func (b byCommunicationPreference) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCommunicationPreference) Less(i, j int) bool {

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

	return false
}

func phoneNumberForPatient(patient *common.Patient) string {
	for _, phoneNumber := range patient.PhoneNumbers {
		if phoneNumber.PhoneType == api.PHONE_CELL {
			return patient.PhoneNumbers[0].Phone
		}
	}
	return ""
}

func (n *notificationManager) notifyDoctor(doctor *common.Doctor, ev interface{}) error {
	communicationPreferences, err := n.dataApi.GetCommunicationPreferencesForAccount(doctor.AccountId.Int64())
	if err != nil {
		return err
	}

	// if there is no communication preference assume its best to communicate via SMS
	if len(communicationPreferences) == 0 {
		if err := sendSMSToUser(n.twilioClient, n.fromNumber, doctor.CellPhone, eventToNotificationViewMapping[reflect.TypeOf(ev)].renderSMS(ev, n.dataApi), n.statSMSFailed, n.statSMSSent); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	}

	sort.Sort(byCommunicationPreference(communicationPreferences))
	topCommunicationPreference := communicationPreferences[0]

	switch topCommunicationPreference.CommunicationType {
	case common.Push:
		if err := pushNotificationToUser(n.snsClient, n.notificationConfigs, ev, doctor.AccountId.Int64(), n.dataApi, n.statPushFailed, n.statPushSent); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := sendSMSToUser(n.twilioClient, n.fromNumber, doctor.CellPhone, eventToNotificationViewMapping[reflect.TypeOf(ev)].renderSMS(ev, n.dataApi), n.statSMSFailed, n.statSMSSent); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		// TODO
	}
	return nil
}

func (n *notificationManager) notifyPatient(patient *common.Patient, ev interface{}) error {
	communicationPreferences, err := n.dataApi.GetCommunicationPreferencesForAccount(patient.AccountId.Int64())
	if err != nil {
		return err
	}

	// if there is no communication preference assume its best to communicate via SMS
	if len(communicationPreferences) == 0 {
		if err := sendSMSToUser(n.twilioClient, n.fromNumber, phoneNumberForPatient(patient), eventToNotificationViewMapping[reflect.TypeOf(ev)].renderSMS(ev, n.dataApi), n.statSMSFailed, n.statSMSSent); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	}

	sort.Sort(byCommunicationPreference(communicationPreferences))
	topCommunicationPreference := communicationPreferences[0]

	switch topCommunicationPreference.CommunicationType {
	case common.Push:
		if err := pushNotificationToUser(n.snsClient, n.notificationConfigs, ev, patient.AccountId.Int64(), n.dataApi, n.statPushFailed, n.statPushSent); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := sendSMSToUser(n.twilioClient, n.fromNumber, phoneNumberForPatient(patient), eventToNotificationViewMapping[reflect.TypeOf(ev)].renderSMS(ev, n.dataApi), n.statSMSFailed, n.statSMSSent); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		// TODO
	}
	return nil
}
