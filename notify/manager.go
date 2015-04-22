package notify

import (
	"net/mail"
	"sort"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

// NotificationManager is responsible for determining how best to route a particular notification to the user based on
// the user's communication preferences.
type NotificationManager struct {
	dataAPI             api.DataAPI
	authAPI             api.AuthAPI
	snsClient           sns.SNSService
	smsAPI              api.SMSAPI
	emailService        email.Service
	fromNumber          string
	notificationConfigs *config.NotificationConfigs
	statSMSSent         *metrics.Counter
	statSMSFailed       *metrics.Counter
	statPushSent        *metrics.Counter
	statPushFailed      *metrics.Counter
	statEmailSent       *metrics.Counter
	statEmailFailed     *metrics.Counter
}

func NewManager(dataAPI api.DataAPI, authAPI api.AuthAPI, snsClient sns.SNSService, smsAPI api.SMSAPI, emailService email.Service, fromNumber string, notificationConfigs *config.NotificationConfigs, statsRegistry metrics.Registry) *NotificationManager {
	manager := &NotificationManager{
		dataAPI:             dataAPI,
		authAPI:             authAPI,
		snsClient:           snsClient,
		smsAPI:              smsAPI,
		emailService:        emailService,
		fromNumber:          fromNumber,
		notificationConfigs: notificationConfigs,
		statSMSSent:         metrics.NewCounter(),
		statSMSFailed:       metrics.NewCounter(),
		statPushSent:        metrics.NewCounter(),
		statPushFailed:      metrics.NewCounter(),
		statEmailSent:       metrics.NewCounter(),
		statEmailFailed:     metrics.NewCounter(),
	}

	statsRegistry.Add("sms/sent", manager.statSMSSent)
	statsRegistry.Add("sms/failed", manager.statSMSFailed)
	statsRegistry.Add("sns/sent", manager.statPushSent)
	statsRegistry.Add("sns/failed", manager.statPushFailed)
	statsRegistry.Add("email/sent", manager.statEmailSent)
	statsRegistry.Add("email/failed", manager.statEmailFailed)

	return manager
}

func (n *NotificationManager) NotifyDoctor(role string, doctorID, accountID int64, msg *Message) error {

	phoneNumbers, err := n.authAPI.GetPhoneNumbersForAccount(accountID)
	if err != nil {
		return err
	}

	var cellPhone string
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.Type == api.PhoneCell {
			cellPhone = phoneNumber.Phone.String()
			break
		}
	}

	return n.sendSMS(cellPhone, msg.ShortMessage)
}

type Message struct {
	ShortMessage string
	EmailType    string
	EmailContext interface{}
}

func (n *NotificationManager) NotifyPatient(patient *common.Patient, msg *Message) error {
	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(patient.AccountID.Int64())
	if err != nil {
		return err
	}
	switch communicationPreference {
	case common.Push:
		if err := n.pushNotificationToUser(patient.AccountID.Int64(), api.RolePatient, msg, 0); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMS(phoneNumberForPatient(patient), msg.ShortMessage); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		to := &mail.Address{Name: patient.FirstName + " " + patient.LastName, Address: patient.Email}
		if err := n.SendEmail(to, msg.EmailType, msg.EmailContext); err != nil {
			return err
		}
	}
	return nil
}

// we are currently determining the way to communicate with the user in a simple order of communication preference
// there will come a point when we need something more complex where we employ different strategies of engagement with the user
// for different notification events; or based on how the user interacts with the notification. We can evolve this over time, given that we
// have the ability to make a decision for every event on how best to communicate with the user
func (n *NotificationManager) determineCommunicationPreferenceBasedOnDefaultConfig(accountID int64) (common.CommunicationType, error) {
	communicationPreferences, err := n.dataAPI.GetCommunicationPreferencesForAccount(accountID)
	if err != nil {
		return common.CommunicationType(""), err
	}

	// if there is no communication preference assume its best to communicate via email
	if len(communicationPreferences) == 0 {
		return common.Email, nil
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(communicationPreferences)))
	return communicationPreferences[0].CommunicationType, nil
}
