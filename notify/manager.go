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

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

// NotificationManager is responsible for determining how best to route a particular notification to the user based on
// the user's communication preferences. The current default is to route to email in the event that the user has no
// preference specified
type NotificationManager struct {
	dataAPI             api.DataAPI
	authAPI             api.AuthAPI
	snsClient           *sns.SNS
	smsAPI              api.SMSAPI
	emailService        email.Service
	fromNumber          string
	fromEmailAddress    string
	notificationConfigs *config.NotificationConfigs
	statSMSSent         metrics.Counter
	statSMSFailed       metrics.Counter
	statPushSent        metrics.Counter
	statPushFailed      metrics.Counter
	statEmailSent       metrics.Counter
	statEmailFailed     metrics.Counter
}

func NewManager(dataAPI api.DataAPI, authAPI api.AuthAPI, snsClient *sns.SNS, smsAPI api.SMSAPI, emailService email.Service, fromNumber, fromEmailAddress string, notificationConfigs *config.NotificationConfigs, statsRegistry metrics.Registry) *NotificationManager {
	manager := &NotificationManager{
		dataAPI:             dataAPI,
		authAPI:             authAPI,
		snsClient:           snsClient,
		smsAPI:              smsAPI,
		emailService:        emailService,
		fromNumber:          fromNumber,
		fromEmailAddress:    fromEmailAddress,
		notificationConfigs: notificationConfigs,
		statSMSSent:         metrics.NewCounter(),
		statSMSFailed:       metrics.NewCounter(),
		statPushSent:        metrics.NewCounter(),
		statPushFailed:      metrics.NewCounter(),
		statEmailSent:       metrics.NewCounter(),
		statEmailFailed:     metrics.NewCounter(),
	}

	statsRegistry.Scope("sms").Add("sms/sent", manager.statSMSSent)
	statsRegistry.Scope("sms").Add("sms/failed", manager.statSMSFailed)
	statsRegistry.Scope("sns").Add("sns/sent", manager.statPushSent)
	statsRegistry.Scope("sns").Add("sns/failed", manager.statPushFailed)
	statsRegistry.Scope("email").Add("email/sent", manager.statEmailSent)
	statsRegistry.Scope("email").Add("email/failed", manager.statEmailFailed)

	return manager
}

func (n *NotificationManager) NotifySupport(toEmail string, event interface{}) error {
	nView := getInternalNotificationViewForEvent(event)
	if nView == nil {
		golog.Errorf("Expected a view to be present for the event %T but it wasn't", event)
		return nil
	}

	emailType, emailCtx, err := nView.renderEmail(event)
	if err != nil {
		return err
	}
	return n.SendEmail(&mail.Address{Address: toEmail}, emailType, emailCtx)
}

func (n *NotificationManager) NotifyDoctor(role string, doctor *common.Doctor, event interface{}) error {
	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(doctor.AccountId.Int64())
	if err != nil {
		golog.Errorf("Unable to get communication preference of doctor: %s", err)
		return err
	}
	switch communicationPreference {
	case common.Push:
		// currently basing the badge count on the doctor app on the total number of pending items
		// in the doctor queue
		notificationCount, err := n.dataAPI.GetPendingItemCountForDoctorQueue(doctor.DoctorId.Int64())
		if err != nil {
			golog.Errorf("Unable to get pending item count for doctor: %s", err)
			return err
		}

		if err := n.pushNotificationToUser(doctor.AccountId.Int64(), role, event, notificationCount); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(doctor.CellPhone.String(), getNotificationViewForEvent(event).renderSMS(role)); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		view := getNotificationViewForEvent(event)
		emailType, emailCtx, err := view.renderEmail(event, role)
		if err != nil {
			return err
		}
		to := &mail.Address{Name: doctor.LongDisplayName, Address: doctor.Email}
		if err := n.SendEmail(to, emailType, emailCtx); err != nil {
			return err
		}
	}
	return nil
}

func (n *NotificationManager) SMSUser(role string, accountID int64, event interface{}) error {

	phoneNumbers, err := n.authAPI.GetPhoneNumbersForAccount(accountID)
	if err != nil {
		return err
	}

	var cellPhone string
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.Type == api.PHONE_CELL {
			cellPhone = phoneNumber.Phone.String()
			break
		}
	}

	return n.sendSMSToUser(cellPhone, getNotificationViewForEvent(event).renderSMS(role))
}

func (n *NotificationManager) NotifyPatient(patient *common.Patient, event interface{}) error {
	communicationPreference, err := n.determineCommunicationPreferenceBasedOnDefaultConfig(patient.AccountId.Int64())
	if err != nil {
		return err
	}
	switch communicationPreference {
	case common.Push:
		if err := n.pushNotificationToUser(patient.AccountId.Int64(), api.PATIENT_ROLE, event, 0); err != nil {
			golog.Errorf("Error sending push to user: %s", err)
			return err
		}
	case common.SMS:
		if err := n.sendSMSToUser(phoneNumberForPatient(patient), getNotificationViewForEvent(event).renderSMS(api.PATIENT_ROLE)); err != nil {
			golog.Errorf("Error sending sms to user: %s", err)
			return err
		}
	case common.Email:
		view := getNotificationViewForEvent(event)
		emailType, emailCtx, err := view.renderEmail(event, api.PATIENT_ROLE)
		if err != nil {
			return err
		}
		to := &mail.Address{Name: patient.FirstName + " " + patient.LastName, Address: patient.Email}
		if err := n.SendEmail(to, emailType, emailCtx); err != nil {
			return err
		}
	}
	return nil
}

// we are currently determining the way to communicate with the user in a simple order of communication preference
// there will come a point when we need something more complex where we employ different strategies of engagement with the user
// for different notification events; or based on how the user interacts with the notification. We can evolve this over time, given that we
// have the ability to make a decision for every event on how best to communicate with the user
func (n *NotificationManager) determineCommunicationPreferenceBasedOnDefaultConfig(accountId int64) (common.CommunicationType, error) {
	communicationPreferences, err := n.dataAPI.GetCommunicationPreferencesForAccount(accountId)
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
