package notify

import (
	"carefront/api"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/aws/sns"
	"carefront/libs/golog"
	"reflect"
	"sort"

	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

type NotificationManager struct {
	dataApi             api.DataAPI
	snsClient           *sns.SNS
	twilioClient        *twilio.Client
	fromNumber          string
	notificationConfigs map[string]*config.NotificationConfig
	statSMSSent         metrics.Counter
	statSMSFailed       metrics.Counter
	statPushSent        metrics.Counter
	statPushFailed      metrics.Counter
}

func NewManager(dataApi api.DataAPI, snsClient *sns.SNS, twilioClient *twilio.Client, fromNumber string, notificationConfigs map[string]*config.NotificationConfig, statsRegistry metrics.Registry) *NotificationManager {

	manager := &NotificationManager{
		dataApi:             dataApi,
		snsClient:           snsClient,
		twilioClient:        twilioClient,
		fromNumber:          fromNumber,
		notificationConfigs: notificationConfigs,
		statSMSSent:         metrics.NewCounter(),
		statSMSFailed:       metrics.NewCounter(),
		statPushSent:        metrics.NewCounter(),
		statPushFailed:      metrics.NewCounter(),
	}

	statsRegistry.Scope("twilio").Add("sms/sent", manager.statSMSSent)
	statsRegistry.Scope("twilio").Add("sms/failed", manager.statSMSFailed)
	statsRegistry.Scope("sns").Add("push/sent", manager.statPushSent)
	statsRegistry.Scope("sns").Add("push/failed", manager.statPushFailed)

	return manager

}

func (n *NotificationManager) NotifyDoctor(doctor *common.Doctor, event interface{}) error {

	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(doctor.AccountId.Int64())
	if err != nil {
		return err
	}
	switch communicationPreference {
	case common.Push:
		notificationCount, err := n.dataApi.GetPendingItemCountForDoctorQueue(doctor.DoctorId.Int64())
		if err != nil {
			return err
		}

		if err := n.pushNotificationToUser(event, notificationCount); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(doctor.CellPhone, eventToNotificationViewMapping[reflect.TypeOf(event)].renderSMS(event, n.dataApi)); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		// TODO
	}
	return nil
}

func (n *NotificationManager) NotifyPatient(patient *common.Patient, event interface{}) error {

	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(patient.AccountId.Int64())
	if err != nil {
		return err
	}
	switch communicationPreference {
	case common.Push:
		notificationCount, err := n.dataApi.GetNotificationCountForPatient(patient.PatientId.Int64())
		if err != nil {
			return err
		}

		if err := n.pushNotificationToUser(event, notificationCount); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(phoneNumberForPatient(patient), eventToNotificationViewMapping[reflect.TypeOf(event)].renderSMS(event, n.dataApi)); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		// TODO
	}
	return nil
}

func (n *NotificationManager) determineCommunicationPreferenceBasedOnDefaultConfig(accountId int64) (common.CommunicationType, error) {
	communicationPreferences, err := n.dataApi.GetCommunicationPreferencesForAccount(accountId)
	if err != nil {
		return common.CommunicationType{}, err
	}

	// if there is no communication preference assume its best to communicate via SMS
	if len(communicationPreferences) == 0 {
		return common.SMS, nil
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(communicationPreferences)))
	return communicationPreferences[0].CommunicationType, nil
}
