package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/settings"
)

// Config represents the configurations required to operate the notification service
type Config struct {
	DeviceRegistrationSQSURL        string
	DeviceDeregistrationSQSURL      string
	NotificationSQSURL              string
	AppleDeviceRegistrationSNSARN   string
	AndriodDeviceRegistrationSNSARN string
	SQSAPI                          sqsiface.SQSAPI
	SNSAPI                          snsiface.SNSAPI
	WebDomain                       string
}

// Service defines the interface that the notification service provides
type Service interface {
	Start()
	Shutdown() error
}

// Note: This is currently very push centric. Concerns will be seperated out as new notification types are supported.
type service struct {
	config               *Config
	dl                   dal.DAL
	snsAPI               snsiface.SNSAPI
	directoryClient      directory.DirectoryClient
	settingsClient       settings.SettingsClient
	registrationWorker   worker.Worker
	deregistrationWorker worker.Worker
	notificationWorker   worker.Worker
}

// New returns an initialized instance of service
func New(
	dl dal.DAL,
	directoryClient directory.DirectoryClient,
	settingsClient settings.SettingsClient,
	config *Config) Service {
	golog.Debugf("Initializing Notification Service with Config: %+v", config)
	s := &service{
		config:          config,
		snsAPI:          config.SNSAPI,
		dl:              dl,
		directoryClient: directoryClient,
		settingsClient:  settingsClient,
	}
	// TODO: Prioritize working through the deregister queue before starting to serve notifications
	s.deregistrationWorker = awsutil.NewSQSWorker(config.SQSAPI, config.DeviceDeregistrationSQSURL, s.processDeviceDeregistration)
	s.registrationWorker = awsutil.NewSQSWorker(config.SQSAPI, config.DeviceRegistrationSQSURL, s.processDeviceRegistration)
	s.notificationWorker = awsutil.NewSQSWorker(config.SQSAPI, config.NotificationSQSURL, s.processNotification)
	return s
}

// Start begins the background workers that the notification services utilizes
func (s *service) Start() {
	golog.Debugf("Starting the Notification service and background workers")
	s.registrationWorker.Start()
	s.deregistrationWorker.Start()
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

	// Check to see if we already have this device registered
	pushConfig, err := s.dl.PushConfigForDeviceID(registrationInfo.DeviceID)
	if api.IsErrNotFound(err) {
		// Generate a new endpoint if we don't already have this device registered
		endpointARN, err := s.generateEndpointARN(registrationInfo)
		if err != nil {
			return errors.Trace(err)
		} else if endpointARN == "" {
			golog.Warningf("No SNS endpoint ARN generated for %s, %s, %s", registrationInfo.ExternalGroupID, registrationInfo.Platform, registrationInfo.DeviceID)
			return nil
		}

		// Insert the newly created endpoint
		golog.Debugf("Inserting new push config with endpoint %s for device registration", endpointARN)
		_, err = s.dl.InsertPushConfig(&dal.PushConfig{
			ExternalGroupID: registrationInfo.ExternalGroupID,
			Platform:        registrationInfo.Platform,
			PlatformVersion: registrationInfo.PlatformVersion,
			AppVersion:      registrationInfo.AppVersion,
			DeviceID:        registrationInfo.DeviceID,
			DeviceToken:     []byte(registrationInfo.DeviceToken),
			PushEndpoint:    endpointARN,
			Device:          registrationInfo.Device,
			DeviceModel:     registrationInfo.DeviceModel,
		})
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}

	// This device is already registered but let's check to see if our token has changed
	if string(pushConfig.DeviceToken) != registrationInfo.DeviceToken {
		// If our token has changed, update the endpoint
		if err := s.updateEndpoint(pushConfig.PushEndpoint, registrationInfo.DeviceToken); err != nil {
			return errors.Trace(err)
		}
	}

	golog.Debugf("Updating existing push config with externalID %s for device registration.", registrationInfo.ExternalGroupID)
	_, err = s.dl.UpdatePushConfig(pushConfig.ID, &dal.PushConfigUpdate{
		ExternalGroupID: ptr.String(registrationInfo.ExternalGroupID),
		Platform:        ptr.String(registrationInfo.Platform),
		PlatformVersion: ptr.String(registrationInfo.PlatformVersion),
		DeviceID:        ptr.String(registrationInfo.DeviceID),
		AppVersion:      ptr.String(registrationInfo.AppVersion),
		DeviceToken:     []byte(registrationInfo.DeviceToken),
	})
	if err != nil {
		return errors.Trace(err)
	}

	return errors.Trace(err)
}

const snsEndpointEnabledAttributeKey = "Enabled"
const snsEndpointTokenAttributeKey = "Token"

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
		// http://docs.aws.amazon.com/sns/latest/api/API_SetEndpointAttributes.html
		Attributes: map[string]*string{snsEndpointEnabledAttributeKey: ptr.String("true")},
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return *createEndpointResponse.EndpointArn, nil
}

func (s *service) updateEndpoint(endpointARN, deviceToken string) error {
	_, err := s.snsAPI.SetEndpointAttributes(&sns.SetEndpointAttributesInput{
		EndpointArn: ptr.String(endpointARN),
		// http://docs.aws.amazon.com/sns/latest/api/API_SetEndpointAttributes.html
		Attributes: map[string]*string{
			snsEndpointEnabledAttributeKey: ptr.String("true"),
			snsEndpointTokenAttributeKey:   ptr.String(deviceToken),
		},
	})
	return errors.Trace(err)
}

func (s *service) processDeviceDeregistration(data []byte) error {
	deregistrationInfo := &notification.DeviceDeregistrationInfo{}
	if err := json.Unmarshal(data, deregistrationInfo); err != nil {
		return errors.Trace(err)
	}
	golog.Debugf("Processing device deregistration event: %+v", deregistrationInfo)

	//Remove the push config for this device id
	_, err := s.dl.DeletePushConfigForDeviceID(deregistrationInfo.DeviceID)
	return errors.Trace(err)
}

var jsonStructure = ptr.String("json")

// TODO: Set and examine communication preferences for caller
// NOTE: This is an initial version of what PUSH notifications can look like. Will discuss with the client team about what we want the formal mature version to be. This is mainly a POC and validation regarding PUSH with Baymax
func (s *service) processNotification(data []byte) error {
	n := &notification.Notification{}
	if err := json.Unmarshal(data, n); err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(s.processPushNotification(n))
}

func (s *service) processPushNotification(n *notification.Notification) error {
	entitiesToNotify, err := s.filterNodesWithNotificationsDisabled(n.EntitiesToNotify)
	if err != nil {
		return errors.Trace(err)
	}

	// Fetch the external ids for these entities and attempt to resolve them to accounts for groups
	externalIDsResp, err := s.directoryClient.ExternalIDs(context.Background(), &directory.ExternalIDsRequest{
		EntityIDs: entitiesToNotify,
	})
	if err != nil {
		return errors.Trace(err)
	}

	accountIDsToNotify, err := s.filterNodesWithNotificationsDisabled(accountIDsFromExternalIDs(externalIDsResp.ExternalIDs))
	if err != nil {
		return errors.Trace(err)
	}
	for _, accountID := range accountIDsToNotify {
		if err := s.sendPushNotificationToExternalGroupID(accountID, n); err != nil {
			golog.Errorf(err.Error())
		}
	}
	return nil
}

func (s *service) filterNodesWithNotificationsDisabled(nodes []string) ([]string, error) {
	// Filter any nodes who explicitly have notifications disabled from the list
	filteredNodes := make([]string, 0, len(nodes))
	for _, nID := range nodes {
		// TODO: Perhaps we should have a bulk version of this call
		// TODO: It would be great to live in a world where the settings service pushed changed settings to hosts that are interested in them
		resp, err := s.settingsClient.GetValues(context.TODO(), &settings.GetValuesRequest{
			Keys:   []*settings.ConfigKey{{Key: notification.ReceiveNotificationsSettingsKey}},
			NodeID: nID,
		})
		// If we failed to get the notification settings then just fail. We'd rather not notify than notify someone disabled
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(resp.Values) != 1 {
			golog.Warningf("Expected only 1 value to be returned for settings key %s and node id %s but got %d. Continuing with first value", notification.ReceiveNotificationsSettingsKey, nID, nodes)
		}
		if resp.Values[0].GetBoolean().Value {
			filteredNodes = append(filteredNodes, nID)
		}
	}
	return filteredNodes, nil
}

const endpointDisabledAWSErrCode = "EndpointDisabled"

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
			snsNote = generateNotification(s.config.WebDomain, n, pushConfig)
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
			aerr, ok := err.(awserr.Error)
			if ok && aerr.Code() == endpointDisabledAWSErrCode {
				golog.Debugf("Encountered disabled endpoint %s", pushConfig.ID)
				// If an endpoint has been disabled then make an attempt to delete it since it is no longer valid
				conc.Go(func() {
					if _, err := s.dl.DeletePushConfig(pushConfig.ID); err != nil {
						golog.Errorf("Encountered error while attempting delete of disabled endpoint %s: %s", pushConfig.ID, err)
					}
				})
			} else {
				golog.Errorf(err.Error())
				// continue so that we do a best effort to publish to all endpoints.
				continue
			}
		}
	}
	return nil
}

func accountIDsFromExternalIDs(eIDs []*directory.ExternalID) []string {
	var accountIDs []string
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
	DefaultMessage string `json:"default"`
	IOSSandBox     string `json:"APNS_SANDBOX,omitempty"`
	IOS            string `json:"APNS,omitempty"`
	Android        string `json:"GCM,omitempty"`
}

type iOSPushNotification struct {
	PushData       *iOSPushData `json:"aps"`
	OrganizationID string       `json:"organization_id"`
	SavedQueryID   string       `json:"saved_query_id"`
	ThreadID       string       `json:"thread_id"`
	MessageID      string       `json:"message_id"`
}

type iOSPushData struct {
	Alert string `json:"alert"`
	URL   string `json:"url"`
}

type androidPushNotification struct {
	PushData *androidPushData `json:"data"`
}

type androidPushData struct {
	Message        string `json:"message"`
	URL            string `json:"url"`
	OrganizationID string `json:"organization_id"`
	SavedQueryID   string `json:"saved_query_id"`
	ThreadID       string `json:"thread_id"`
	MessageID      string `json:"message_id"`
	PushID         string `json:"push_id"`
}

func generateNotification(webDomain string, n *notification.Notification, pushConfig *dal.PushConfig) *snsNotification {
	url := deeplink.ThreadMessageURLShareable(webDomain, n.OrganizationID, n.ThreadID, n.MessageID)
	isNotifData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Alert: n.ShortMessage,
			URL:   url,
		},
		OrganizationID: n.OrganizationID,
		SavedQueryID:   n.SavedQueryID,
		ThreadID:       n.ThreadID,
		MessageID:      n.MessageID,
	})
	if err != nil {
		golog.Errorf("Error while serializing ios notification data: %s", err)
	}
	androidNotifData, err := json.Marshal(&androidPushNotification{
		PushData: &androidPushData{
			Message:        n.ShortMessage,
			URL:            url,
			OrganizationID: n.OrganizationID,
			SavedQueryID:   n.SavedQueryID,
			ThreadID:       n.ThreadID,
			MessageID:      n.MessageID,
			PushID:         fmt.Sprintf("thread:%s", n.ThreadID),
		},
	})
	if err != nil {
		golog.Errorf("Error while serializing android notification data: %s", err)
	}

	return &snsNotification{
		DefaultMessage: n.ShortMessage,
		IOSSandBox:     string(isNotifData),
		IOS:            string(isNotifData),
		Android:        string(androidNotifData),
	}
}
