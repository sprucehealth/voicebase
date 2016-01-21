package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
)

// Config represents the configurations required to operate the notification service
type Config struct {
	DeviceRegistrationSQSURL        string
	NotificationSQSURL              string
	AppleDeviceRegistrationSNSARN   string
	AndriodDeviceRegistrationSNSARN string
	SNSAPI                          snsiface.SNSAPI
	SQSAPI                          sqsiface.SQSAPI
}

type notificationDAL interface {
	Transact(trans func(dal dal.DAL) error) (err error)
	InsertPushConfig(model *dal.PushConfig) (dal.PushConfigID, error)
	PushConfig(id dal.PushConfigID) (*dal.PushConfig, error)
	PushConfigForDeviceID(deviceID string) (*dal.PushConfig, error)
	PushConfigsForExternalGroupID(externalGroupID string) ([]*dal.PushConfig, error)
	UpdatePushConfig(id dal.PushConfigID, update *dal.PushConfigUpdate) (int64, error)
	DeletePushConfig(id dal.PushConfigID) (int64, error)
}

// Service defines the interface that the notification service provides
type Service interface {
	Start()
	Shutdown() error
}

// Note: This is currently very push centric. Concerns will be seperated out as new notification types are supported.
type service struct {
	config             *Config
	dl                 notificationDAL
	snsAPI             snsiface.SNSAPI
	directoryClient    directory.DirectoryClient
	registrationWorker worker.Worker
	notificationWorker worker.Worker
}

// New returns an initialized instance of service
func New(dl notificationDAL, directoryClient directory.DirectoryClient, config *Config) Service {
	golog.Debugf("Initializing Notification Service with Config: %+v", config)
	s := &service{
		config:          config,
		dl:              dl,
		directoryClient: directoryClient,
		snsAPI:          config.SNSAPI,
	}
	s.registrationWorker = awsutil.NewSQSWorker(config.SQSAPI, config.DeviceRegistrationSQSURL, s.processDeviceRegistration)
	s.notificationWorker = awsutil.NewSQSWorker(config.SQSAPI, config.NotificationSQSURL, s.processNotification)
	return s
}

// Start begins the background workers that the notification services utilizes
func (s *service) Start() {
	golog.Debugf("Starting the Notification service and background workers")
	s.registrationWorker.Start()
	s.notificationWorker.Start()
}

// Shutdown cleanly shut down the service
func (s *service) Shutdown() error {
	golog.Debugf("Shutting down the Notification service and background workers")
	// TODO
	return nil
}

func (s *service) processDeviceRegistration(data []byte) error {
	registrationInfo := &notification.DeviceRegistrationInfo{}
	if err := json.Unmarshal(data, registrationInfo); err != nil {
		return errors.Trace(err)
	}
	golog.Debugf("Processing device registration event: %+v", registrationInfo)

	endpointARN, err := s.generateEndpointARN(registrationInfo)
	if err != nil {
		return errors.Trace(err)
	} else if endpointARN == "" {
		golog.Warningf("No SNS endpoint ARN generated for %s, %s, %s", registrationInfo.ExternalGroupID, registrationInfo.Platform, registrationInfo.DeviceID)
		return nil
	}

	// If we already have a config for this device then just update the endpoint, do we need to regenerate the endpoint?
	pushConfig, err := s.dl.PushConfigForDeviceID(registrationInfo.DeviceID)
	if api.IsErrNotFound(err) {
		golog.Debugf("Inserting new push config with endpoint %s for device registration event: %+v", endpointARN, registrationInfo)
		if _, err := s.dl.InsertPushConfig(&dal.PushConfig{
			ExternalGroupID: registrationInfo.ExternalGroupID,
			Platform:        registrationInfo.Platform,
			PlatformVersion: registrationInfo.PlatformVersion,
			AppVersion:      registrationInfo.AppVersion,
			DeviceID:        registrationInfo.DeviceID,
			DeviceToken:     []byte(registrationInfo.DeviceToken),
			PushEndpoint:    endpointARN,
			Device:          registrationInfo.Device,
			DeviceModel:     registrationInfo.DeviceModel,
		}); err != nil {
			return errors.Trace(err)
		}
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}

	golog.Debugf("Updating existing push config with endpoint %s for device registration event: %+v", endpointARN, registrationInfo)
	_, err = s.dl.UpdatePushConfig(pushConfig.ID, &dal.PushConfigUpdate{
		DeviceToken:     []byte(registrationInfo.DeviceToken),
		ExternalGroupID: ptr.String(registrationInfo.ExternalGroupID),
		Platform:        ptr.String(registrationInfo.Platform),
		PlatformVersion: ptr.String(registrationInfo.PlatformVersion),
		AppVersion:      ptr.String(registrationInfo.AppVersion),
		PushEndpoint:    ptr.String(endpointARN),
	})

	return errors.Trace(err)
}

func (s *service) generateEndpointARN(info *notification.DeviceRegistrationInfo) (string, error) {
	var arn string
	switch info.Platform {
	case "iOS":
		arn = s.config.AppleDeviceRegistrationSNSARN
	case "android":
		arn = s.config.AndriodDeviceRegistrationSNSARN
	default:
		golog.Warningf("Cannot register unknown platform %s for push notifications", info.Platform)
		return "", nil
	}
	if arn == "" {
		golog.Errorf("No SNS arn provided to register device %s, %s, %s", info.ExternalGroupID, info.Platform, info.DeviceID)
		return "", nil
	}
	createEndpointResponse, err := s.snsAPI.CreatePlatformEndpoint(&sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(arn),
		Token: ptr.String(info.DeviceToken),
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return *createEndpointResponse.EndpointArn, nil
}

var jsonStructure = ptr.String("json")

// TODO: Set and examine communication preferences for caller
// NOTE: This is an initial version of what PUSH notifications can look like. Will discuss with the client team about what we want the formal mature version to be. This is mainly a POC and validation regarding PUSH with Baymax
func (s *service) processNotification(data []byte) error {
	n := &notification.Notification{}
	if err := json.Unmarshal(data, n); err != nil {
		return errors.Trace(err)
	}
	golog.Debugf("Processing notification event: %+v", n)
	return errors.Trace(s.processPushNotification(n))
}

func (s *service) processPushNotification(n *notification.Notification) error {
	// Fetch the external ids for these entities and attempt to resolve them to accounts for groups
	externalIDsResp, err := s.directoryClient.ExternalIDs(context.Background(), &directory.ExternalIDsRequest{
		EntityIDs: n.EntitiesToNotify,
	})
	if err != nil {
		return errors.Trace(err)
	}
	accountIDs := accountIDsFromExternalIDs(externalIDsResp.ExternalIDs)
	for _, accountID := range accountIDs {
		if err := s.sendPushNotificationToExternalGroupID(accountID, n); err != nil {
			golog.Errorf(err.Error())
		}
	}
	return nil
}

func (s *service) sendPushNotificationToExternalGroupID(externalGroupID string, n *notification.Notification) error {
	pushConfigs, err := s.dl.PushConfigsForExternalGroupID(externalGroupID)
	if err != nil {
		return errors.Trace(err)
	}

	// TODO: Account for partial failure here. If some configs succeed and others don't
	for _, pushConfig := range pushConfigs {
		var snsNote *snsNotification
		switch pushConfig.Platform {
		case "iOS", "android":
			snsNote = generateNotification(n, pushConfig)
		default:
			return errors.Trace(fmt.Errorf("Cannot send push notification to unknown platform %q for push notifications", pushConfig.Platform))
		}

		msg, err := json.Marshal(snsNote)
		if err != nil {
			return errors.Trace(err)
		}

		if _, err := s.snsAPI.Publish(&sns.PublishInput{
			Message:          ptr.String(string(msg)),
			MessageStructure: jsonStructure,
			TargetArn:        ptr.String(pushConfig.PushEndpoint),
		}); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func accountIDsFromExternalIDs(eIDs []*directory.ExternalID) []string {
	var accountIDs []string
	// TODO: Fix this ID parsing once we move to the new format and stop prefixing from graphql
	for _, eID := range eIDs {
		i := strings.IndexByte(eID.ID, '_')
		if i != -1 {
			prefix := eID.ID[:(i + 1)]
			switch prefix {
			case auth.AccountIDPrefix:
				accountIDs = append(accountIDs, eID.ID)
			}
		}
	}
	return accountIDs
}

// http://docs.aws.amazon.com/sns/latest/dg/mobile-push-send-custommessage.html
type snsNotification struct {
	DefaultMessage string                   `json:"default"`
	IOSSandBox     *iOSPushNotification     `json:"APNS_SANDBOX,omitempty"`
	IOS            *iOSPushNotification     `json:"APNS,omitempty"`
	Android        *androidPushNotification `json:"GCM,omitempty"`
	ThreadID       string                   `json:"thread_id"`
	URL            string                   `json:"url"`
}

type iOSPushNotification struct {
	PushData *iOSPushData `json:"aps"`
}

type iOSPushData struct {
	Alert string `json:"alert"`
}

type androidPushNotification struct {
	PushData *androidPushData `json:"data"`
}

type androidPushData struct {
	Message string `json:"message"`
}

func generateNotification(n *notification.Notification, pushConfig *dal.PushConfig) *snsNotification {
	url := threadActivityURL(n.OrganizationID, n.ThreadID)
	iOSNotif := &iOSPushNotification{
		PushData: &iOSPushData{
			Alert: n.ShortMessage,
		},
	}
	return &snsNotification{
		DefaultMessage: n.ShortMessage,
		IOSSandBox:     iOSNotif,
		IOS:            iOSNotif,
		Android: &androidPushNotification{
			PushData: &androidPushData{
				Message: n.ShortMessage,
			},
		},
		ThreadID: n.ThreadID,
		URL:      url,
	}
}

// TODO: Perhaps the notification provider should create this
func threadActivityURL(organizationID, threadID string) string {
	return fmt.Sprintf("https://baymax.com/org/%s/thread/%s/", organizationID, threadID)
}
