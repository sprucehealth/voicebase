package notify

import (
	"sort"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common/config"
	"github.com/sprucehealth/backend/cmd/svc/restapi/email"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
)

// NotificationManager is responsible for determining how best to route a particular notification to the user based on
// the user's communication preferences.
type NotificationManager struct {
	dataAPI             api.DataAPI
	authAPI             api.AuthAPI
	snsClient           snsiface.SNSAPI
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

func NewManager(dataAPI api.DataAPI, authAPI api.AuthAPI, snsClient snsiface.SNSAPI, smsAPI api.SMSAPI, emailService email.Service, fromNumber string, notificationConfigs *config.NotificationConfigs, statsRegistry metrics.Registry) *NotificationManager {
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
		if phoneNumber.Type == common.PNTCell {
			cellPhone = phoneNumber.Phone.String()
			break
		}
	}

	if cellPhone == "" {
		golog.Errorf("Unable to get cell number for doctorID %d to send message '%s'", doctorID, msg.ShortMessage)
		return nil
	}

	return n.sendSMS(cellPhone, msg.ShortMessage)
}

// CommunicationPreferenceOption is used to indicate by the caller
// how best to communicate a notification to the intended receiver.
// Options can be combined by using the | operator.
// If caller just wants to ensure that preference gets delivered
// to the first available preference, then use CPFirstUserPreference
type CommunicationPreferenceOption int

const (
	// CPEmail indicates to send the notification via email
	CPEmail CommunicationPreferenceOption = 1 << iota
	// CPPush indicates to send notification via push
	CPPush
	// CPSMS indicates to send notification via sms
	CPSMS
	// CPFirstUserPreference indicates to send notification via first preference
	// of user. This is considered to be the default option.
	CPFirstUserPreference CommunicationPreferenceOption = 0
)

func (o CommunicationPreferenceOption) has(opt CommunicationPreferenceOption) bool {
	if o == CPFirstUserPreference {
		return true
	} else if opt == CPFirstUserPreference {
		return o == opt
	}
	return o&opt == opt
}

// Message is used to indicate the message to send along with
// the communication preference as stated by the caller.
type Message struct {
	ShortMessage string

	// PushID is used specifically for push notifications to make it
	// possible for the client to handle different types of push notifications.
	PushID         string
	EmailType      string
	EmailVars      []mandrill.Var
	CommPreference CommunicationPreferenceOption
}

// NotifyPatient sends the message to the patient based on patient's user preferences.
func (n *NotificationManager) NotifyPatient(patient *common.Patient, msg *Message) error {
	communicationPreferences, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(patient.AccountID.Int64())
	if err != nil {
		return err
	}

	// its possible for the patient to have no communication preferences
	// in the event they denied push notification prompt and also
	// unsubscribed from email notifications.
	if len(communicationPreferences) == 0 {
		return nil
	}

	for _, cp := range communicationPreferences {
		switch cp.CommunicationType {
		case common.Push:
			if !msg.CommPreference.has(CPPush) {
				continue
			}
			if err := n.pushNotificationToUser(patient.AccountID.Int64(), api.RolePatient, msg, 0); err != nil {
				golog.Errorf("Error sending push to user: %s", err)
				return err
			}
		case common.SMS:
			if !msg.CommPreference.has(CPSMS) {
				continue
			}
			if err := n.sendSMS(phoneNumberForPatient(patient), msg.ShortMessage); err != nil {
				golog.Errorf("Error sending sms to user: %s", err)
				return err
			}
		case common.Email:
			if !msg.CommPreference.has(CPEmail) {
				continue
			}
			if err := n.SendEmail(patient.AccountID.Int64(), msg.EmailType, msg.EmailVars); err != nil {
				return err
			}
		}

		if msg.CommPreference.has(CPFirstUserPreference) {
			break
		}
	}

	return nil
}

// we are currently determining the way to communicate with the user in a simple order of communication preference
// there will come a point when we need something more complex where we employ different strategies of engagement with the user
// for different notification events; or based on how the user interacts with the notification. We can evolve this over time, given that we
// have the ability to make a decision for every event on how best to communicate with the user
func (n *NotificationManager) determineCommunicationPreferenceBasedOnDefaultConfig(accountID int64) ([]*common.CommunicationPreference, error) {
	communicationPreferences, err := n.dataAPI.GetCommunicationPreferencesForAccount(accountID)
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(ByCommunicationPreference(communicationPreferences)))
	return communicationPreferences, nil
}
