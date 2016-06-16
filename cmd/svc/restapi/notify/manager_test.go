package notify

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common/config"
	"github.com/sprucehealth/backend/cmd/svc/restapi/email"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/libs/test"
)

type mockDataAPI_notificationManager struct {
	api.DataAPI
	communicationPreferences []*common.CommunicationPreference
	pushConfigList           []*common.PushConfigData
}

type mockSMSAPI struct {
	smsSent bool
}

func (m *mockSMSAPI) Send(fromNumber, toNumber, text string) error {
	m.smsSent = true
	return nil
}

type mockSNS struct {
	snsiface.SNSAPI
	pushSent bool
}

func (m *mockSNS) Publish(s *sns.PublishInput) (*sns.PublishOutput, error) {
	m.pushSent = true
	return nil, nil
}

type mockEmail struct {
	emailSent bool
}

func (m *mockEmail) Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt email.Option) ([]*mandrill.SendMessageResponse, error) {
	m.emailSent = true
	return nil, nil
}

func (m *mockDataAPI_notificationManager) GetCommunicationPreferencesForAccount(accountID int64) ([]*common.CommunicationPreference, error) {
	return m.communicationPreferences, nil
}
func (m *mockDataAPI_notificationManager) GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error) {
	return m.pushConfigList, nil
}

func TestPatientNotification_SinglePreference(t *testing.T) {
	conc.Testing = true
	mda := &mockDataAPI_notificationManager{
		communicationPreferences: []*common.CommunicationPreference{
			{
				CommunicationType: common.Email,
			},
			{
				CommunicationType: common.Push,
			},
		},
		pushConfigList: []*common.PushConfigData{
			{
				Platform:       device.IOS,
				AppType:        "Patient",
				AppEnvironment: "Dev",
			},
		},
	}

	email := &mockEmail{}
	push := &mockSNS{}
	sms := &mockSMSAPI{}
	configs := config.NotificationConfigs(map[string]*config.NotificationConfig{
		"iOS-Patient-Dev": {},
	})

	n := NewManager(mda, nil, push, sms, email, "2068888888", &configs, metrics.NewRegistry())

	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage: "Hello",
	}); err != nil {
		t.Fatalf(err.Error())
	}

	// shouldve sent the patient a push and nothing else
	test.Equals(t, true, push.pushSent)
	test.Equals(t, false, email.emailSent)
	test.Equals(t, false, sms.smsSent)

	mda.communicationPreferences = []*common.CommunicationPreference{
		{
			CommunicationType: common.Email,
		},
	}

	push.pushSent = false
	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage: "Hello",
	}); err != nil {
		t.Fatalf(err.Error())
	}

	// message should've been sent only to email
	test.Equals(t, false, push.pushSent)
	test.Equals(t, true, email.emailSent)
	test.Equals(t, false, sms.smsSent)

	mda.communicationPreferences = []*common.CommunicationPreference{
		{
			CommunicationType: common.SMS,
		},
	}

	email.emailSent = false
	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage: "Hello",
	}); err != nil {
		t.Fatalf(err.Error())
	}

	// message should've been sent only to sms
	test.Equals(t, false, push.pushSent)
	test.Equals(t, false, email.emailSent)
	test.Equals(t, true, sms.smsSent)

	// should not be sent anywhere if no preference stated for user
	mda.communicationPreferences = nil

	sms.smsSent = false
	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage: "Hello",
	}); err != nil {
		t.Fatalf(err.Error())
	}

	test.Equals(t, false, push.pushSent)
	test.Equals(t, false, email.emailSent)
	test.Equals(t, false, sms.smsSent)
}

func TestPatientNotification_MultiplePreference(t *testing.T) {
	conc.Testing = true
	mda := &mockDataAPI_notificationManager{
		communicationPreferences: []*common.CommunicationPreference{
			{
				CommunicationType: common.Email,
			},
			{
				CommunicationType: common.Push,
			},
			{
				CommunicationType: common.SMS,
			},
		},
		pushConfigList: []*common.PushConfigData{
			{
				Platform:       device.IOS,
				AppType:        "Patient",
				AppEnvironment: "Dev",
			},
		},
	}

	email := &mockEmail{}
	push := &mockSNS{}
	sms := &mockSMSAPI{}
	configs := config.NotificationConfigs(map[string]*config.NotificationConfig{
		"iOS-Patient-Dev": {},
	})

	n := NewManager(mda, nil, push, sms, email, "2068888888", &configs, metrics.NewRegistry())

	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage:   "Hello",
		CommPreference: CPEmail | CPPush,
	}); err != nil {
		t.Fatalf(err.Error())
	}

	test.Equals(t, true, push.pushSent)
	test.Equals(t, true, email.emailSent)
	test.Equals(t, false, sms.smsSent)

	push.pushSent = false
	email.emailSent = false
	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage:   "Hello",
		CommPreference: CPEmail,
	}); err != nil {
		t.Fatalf(err.Error())
	}

	// message should've been sent only to email
	test.Equals(t, false, push.pushSent)
	test.Equals(t, true, email.emailSent)
	test.Equals(t, false, sms.smsSent)

	email.emailSent = false
	if err := n.NotifyPatient(&common.Patient{}, &Message{
		ShortMessage:   "Hello",
		CommPreference: CPEmail | CPSMS | CPPush,
	}); err != nil {
		t.Fatalf(err.Error())
	}

	test.Equals(t, true, push.pushSent)
	test.Equals(t, true, email.emailSent)
	test.Equals(t, true, sms.smsSent)
}
