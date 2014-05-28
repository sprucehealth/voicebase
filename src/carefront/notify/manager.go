package notify

import (
	"carefront/api"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/aws/sns"
	"carefront/libs/golog"
	"sort"

	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

// NotificationManager is responsible for determining how best to route a particular notification to the user based on
// the user's communication preferences. The current default is to route to SMS in the event that the user has no
// preference specified
type NotificationManager struct {
	dataApi             api.DataAPI
	snsClient           *sns.SNS
	twilioClient        *twilio.Client
	fromNumber          string
	notificationConfigs *config.NotificationConfigs
	statSMSSent         metrics.Counter
	statSMSFailed       metrics.Counter
	statPushSent        metrics.Counter
	statPushFailed      metrics.Counter
}

func NewManager(dataApi api.DataAPI, snsClient *sns.SNS, twilioClient *twilio.Client, fromNumber string, notificationConfigs *config.NotificationConfigs, statsRegistry metrics.Registry) *NotificationManager {

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

	if statsRegistry != nil {
		statsRegistry.Scope("twilio").Add("sms/sent", manager.statSMSSent)
		statsRegistry.Scope("twilio").Add("sms/failed", manager.statSMSFailed)
		statsRegistry.Scope("sns").Add("push/sent", manager.statPushSent)
		statsRegistry.Scope("sns").Add("push/failed", manager.statPushFailed)
	}

	return manager

}

func (n *NotificationManager) NotifyDoctor(doctor *common.Doctor, event interface{}) error {

	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(doctor.AccountId.Int64())
	if err != nil {
		return err
	}
	switch communicationPreference {
	case common.Push:
		// currently basing the badge count on the doctor app on the total number of pending items
		// in the doctor queue
		notificationCount, err := n.dataApi.GetPendingItemCountForDoctorQueue(doctor.DoctorId.Int64())
		if err != nil {
			return err
		}

		if err := n.pushNotificationToUser(doctor.AccountId.Int64(), event, notificationCount); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(doctor.CellPhone, getNotificationViewForEvent(event).renderSMS(event, n.dataApi)); err != nil {
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
		// currently basing the badge count on the number of notifications in the patient's health log.
		notificationCount, err := n.dataApi.GetNotificationCountForPatient(patient.PatientId.Int64())
		if err != nil {
			return err
		}

		if err := n.pushNotificationToUser(patient.AccountId.Int64(), event, notificationCount); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(phoneNumberForPatient(patient), getNotificationViewForEvent(event).renderSMS(event, n.dataApi)); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		// TODO
	}
	return nil
}

// we are currently determining the way to communicate with the user in a simple order of communication preference
// there will come a point when we need something more complex where we employ different strategies of engagement with the user
// for different notification events; or based on how the user interacts with the notification. We can evolve this over time, given that we
// have the ability to make a decision for every event on how best to communicate with the user
func (n *NotificationManager) determineCommunicationPreferenceBasedOnDefaultConfig(accountId int64) (common.CommunicationType, error) {
	communicationPreferences, err := n.dataApi.GetCommunicationPreferencesForAccount(accountId)
	if err != nil {
		return common.CommunicationType(""), err
	}

	// if there is no communication preference assume its best to communicate via SMS
	if len(communicationPreferences) == 0 {
		return common.SMS, nil
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(communicationPreferences)))
	return communicationPreferences[0].CommunicationType, nil
}
