package service

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	nsettings "github.com/sprucehealth/backend/cmd/svc/notification/internal/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
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
	config *Config,
) Service {
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
	s.registrationWorker.Stop(time.Second * 30)
	s.deregistrationWorker.Stop(time.Second * 30)
	s.notificationWorker.Stop(time.Second * 30)
	return nil
}

func (s *service) processDeviceRegistration(ctx context.Context, data string) error {
	registrationInfo := &notification.DeviceRegistrationInfo{}
	if err := json.Unmarshal([]byte(data), registrationInfo); err != nil {
		return errors.Trace(err)
	}

	// Check to see if we already have this device token registered
	pushConfig, err := s.dl.PushConfigForDeviceToken(registrationInfo.DeviceToken)
	if errors.Cause(err) == dal.ErrNotFound {
		// Generate a new endpoint if we don't already have this device token registered
		endpointARN, err := s.generateEndpointARN(registrationInfo)
		if err != nil {
			return errors.Trace(err)
		} else if endpointARN == "" {
			golog.Errorf("No SNS endpoint ARN generated for %s, %s, %s", registrationInfo.ExternalGroupID, registrationInfo.Platform, registrationInfo.DeviceID)
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
		return errors.Trace(err)
	} else if err != nil {
		return errors.Trace(err)
	}

	// Perform the update here to support shared devices. We don't want to send push for an account that is no longer on a device
	golog.Debugf("Updating existing push config with externalID %s for device registration.", registrationInfo.ExternalGroupID)
	_, err = s.dl.UpdatePushConfig(pushConfig.ID, &dal.PushConfigUpdate{
		ExternalGroupID: &registrationInfo.ExternalGroupID,
		Platform:        &registrationInfo.Platform,
		PlatformVersion: &registrationInfo.PlatformVersion,
		DeviceID:        &registrationInfo.DeviceID,
		AppVersion:      &registrationInfo.AppVersion,
		DeviceToken:     []byte(registrationInfo.DeviceToken),
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
		PlatformApplicationArn: &arn,
		Token: &info.DeviceToken,
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return *createEndpointResponse.EndpointArn, nil
}

func (s *service) processDeviceDeregistration(ctx context.Context, data string) error {
	deregistrationInfo := &notification.DeviceDeregistrationInfo{}
	if err := json.Unmarshal([]byte(data), deregistrationInfo); err != nil {
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
func (s *service) processNotification(ctx context.Context, data string) error {
	n := &notification.Notification{}
	if err := json.Unmarshal([]byte(data), n); err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(s.processPushNotification(ctx, n))
}

// TODO: mraines: This section of code has become incredibly push and new_message specific. It needs a desperate refactor before any other
//   notification work is done.
func (s *service) processPushNotification(ctx context.Context, n *notification.Notification) error {
	// Filter the entity list for org specific settings since the entity is scoped to the org
	entitiesToNotify, err := s.filterNodesWithNotificationsDisabled(ctx, n.EntitiesToNotify)
	if err != nil {
		return errors.Trace(err)
	}
	switch n.Type {
	case notification.DeprecatedNewMessageOnThread:
		// do nothing in this case since we don't have the information to filter properly
	case notification.NewMessageOnInternalThread:
		entitiesToNotify, err = s.filterNodesForThreadActivityPreferences(ctx, entitiesToNotify, n.EntitiesAtReferenced, notification.TeamNotificationPreferencesSettingsKey)
	case notification.NewMessageOnExternalThread:
		entitiesToNotify, err = s.filterNodesForThreadActivityPreferences(ctx, entitiesToNotify, n.EntitiesAtReferenced, notification.PatientNotificationPreferencesSettingsKey)
	case notification.IncomingIPCall:
	default:
		golog.Errorf("Unable to handle unknown notification type %s", n.Type)
		return nil
	}
	if err != nil {
		return errors.Trace(err)
	}

	// Fetch the external ids for these entities and attempt to resolve them to accounts for groups
	externalIDsResp, err := s.directoryClient.ExternalIDs(ctx, &directory.ExternalIDsRequest{
		EntityIDs: entitiesToNotify,
	})
	if err != nil {
		return errors.Trace(err)
	}

	// Map our notification information back to our new external id info
	for _, eID := range externalIDsResp.ExternalIDs {
		if n.ShortMessages != nil {
			n.ShortMessages[eID.ID] = n.ShortMessages[eID.EntityID]
		}
		if n.UnreadCounts != nil {
			n.UnreadCounts[eID.ID] = n.UnreadCounts[eID.EntityID]
		}
	}

	for _, accountID := range externalIDsResp.ExternalIDs {
		if err := s.sendPushNotificationToExternalGroupID(accountID.ID, n); err != nil {
			golog.Errorf(err.Error())
		}
	}
	return nil
}

func (s *service) filterNodesWithNotificationsDisabled(ctx context.Context, nodes []string) ([]string, error) {
	// Filter any nodes who explicitly have notifications disabled from the list
	filteredNodes := make([]string, 0, len(nodes))
	for _, nID := range nodes {
		// TODO: Perhaps we should have a bulk version of this call
		// TODO: It would be great to live in a world where the settings service pushed changed settings to hosts that are interested in them
		resp, err := s.settingsClient.GetValues(ctx, &settings.GetValuesRequest{
			Keys:   []*settings.ConfigKey{{Key: notification.ReceiveNotificationsSettingsKey}},
			NodeID: nID,
		})
		// If we failed to get the notification settings then just fail. We'd rather not notify than notify someone disabled
		if err != nil {
			golog.Errorf("Error while getting notification preference setting for node %s ignoring: %s", nID, err)
			continue
		}
		if len(resp.Values) == 0 {
			golog.Warningf("Expected a value to be returned for settings key %s and node id %s but got 0. Skipping this node", notification.ReceiveNotificationsSettingsKey, nID)
			continue
		}
		if len(resp.Values) != 1 {
			golog.Warningf("Expected only 1 value to be returned for settings key %s and node id %s but got %d. Continuing with first value", notification.ReceiveNotificationsSettingsKey, nID, len(resp.Values))
		}
		if resp.Values[0].GetBoolean().Value {
			filteredNodes = append(filteredNodes, nID)
		}
	}
	return filteredNodes, nil
}

func (s *service) filterNodesForThreadActivityPreferences(ctx context.Context, entityIDs []string, atReferencedEntityIDs map[string]struct{}, activityPreferenceSettingsKey string) ([]string, error) {
	// Filter any nodes who explicitly have notifications disabled from the list
	filteredNodes := make([]string, 0, len(entityIDs))
	for _, eID := range entityIDs {
		// TODO: Perhaps we should have a bulk version of this call
		// TODO: It would be great to live in a world where the settings service pushed changed settings to hosts that are interested in them
		singleSelect, err := settings.GetSingleSelectValue(ctx, s.settingsClient, &settings.GetValuesRequest{
			Keys:   []*settings.ConfigKey{{Key: activityPreferenceSettingsKey}},
			NodeID: eID,
		})
		// If we failed to get the notification settings then just fail. We'd rather not notify than notify someone disabled
		if err != nil {
			golog.Errorf("Error while getting activity preference setting for node %s ignoring: %s", eID, err)
			continue
		}
		switch singleSelect.Item.ID {
		case nsettings.ThreadActivityNotificationPreferenceAllMessages:
			filteredNodes = append(filteredNodes, eID)
		case nsettings.ThreadActivityNotificationPreferenceReferencedOnly:
			if _, ok := atReferencedEntityIDs[eID]; ok {
				filteredNodes = append(filteredNodes, eID)
			}
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

	for _, pushConfig := range pushConfigs {
		var snsNote *snsNotification
		switch pushConfig.Platform {
		case "iOS", "android":
			snsNote = generateNotification(s.config.WebDomain, n, externalGroupID)
		default:
			return errors.Errorf("Cannot send push notification to unhandled platform %q", pushConfig.Platform)
		}

		msg, err := json.Marshal(snsNote)
		if err != nil {
			return errors.Trace(err)
		}

		if _, err := s.snsAPI.Publish(&sns.PublishInput{
			Message:          ptr.String(string(msg)),
			MessageStructure: jsonStructure,
			TargetArn:        &pushConfig.PushEndpoint,
		}); err != nil {
			aerr, ok := err.(awserr.Error)
			if ok && aerr.Code() == endpointDisabledAWSErrCode {
				golog.Infof("Encountered disabled endpoint %+v", pushConfig)
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
		golog.Infof("Sent push '%s' to  %s", string(msg), pushConfig.PushEndpoint)
	}
	return nil
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
	Type           string       `json:"type"`
	OrganizationID string       `json:"organization_id"`
	SavedQueryID   string       `json:"saved_query_id,omitempty"`
	ThreadID       string       `json:"thread_id,omitempty"`
	MessageID      string       `json:"message_id,omitempty"`
	CallID         string       `json:"call_id,omitempty"`
	URL            string       `json:"url"`
}

// https://developer.apple.com/library/ios/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/TheNotificationPayload.html#//apple_ref/doc/uid/TP40008194-CH107-SW1
type iOSPushData struct {
	Alert string `json:"alert"`
	//Badge            int    `json:"badge"`
	ContentAvailable int    `json:"content-available,omitempty"`
	Sound            string `json:"sound"`
}

// https://developers.google.com/cloud-messaging/concept-options
type androidPushNotification struct {
	CollapseKey string           `json:"collapse_key"`
	Priority    string           `json:"priority"`
	PushData    *androidPushData `json:"data"`
}

type androidPushData struct {
	Type           string `json:"type"`
	Background     bool   `json:"background"`
	Message        string `json:"message"`
	URL            string `json:"url"`
	UnreadCount    int    `json:"unread_count"`
	OrganizationID string `json:"organization_id"`
	SavedQueryID   string `json:"saved_query_id,omitempty"`
	ThreadID       string `json:"thread_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`
	CallID         string `json:"call_id,omitempty"`
	PushID         string `json:"push_id"`
}

func generateNotification(webDomain string, n *notification.Notification, targetID string) *snsNotification {
	msg := n.ShortMessages[targetID]

	var url string
	switch n.Type {
	case notification.DeprecatedNewMessageOnThread,
		notification.NewMessageOnInternalThread,
		notification.NewMessageOnExternalThread:
		url = deeplink.ThreadMessageURLShareable(webDomain, n.OrganizationID, n.ThreadID, n.MessageID)
	}

	iOSData := &iOSPushData{
		Alert:            msg,
		ContentAvailable: 1,
	}
	if msg != "" {
		if n.Type == notification.IncomingIPCall {
			iOSData.Sound = "ReceivingCall_Looped.caf"
		} else {
			iOSData.Sound = "default"
		}
	}
	isNotifData, err := json.Marshal(&iOSPushNotification{
		PushData:       iOSData,
		Type:           string(n.Type),
		OrganizationID: n.OrganizationID,
		SavedQueryID:   n.SavedQueryID,
		ThreadID:       n.ThreadID,
		MessageID:      n.MessageID,
		CallID:         n.CallID,
		URL:            url,
	})
	if err != nil {
		golog.Errorf("Error while serializing ios notification data: %s", err)
	}

	// TODO: Perhaps move this into an option for the notification creator
	priority := "normal"
	if n.Type == notification.IncomingIPCall {
		priority = "high"
	}
	androidNotifData, err := json.Marshal(&androidPushNotification{
		CollapseKey: n.CollapseKey,
		Priority:    priority,
		PushData: &androidPushData{
			Type:           string(n.Type),
			Background:     msg == "",
			Message:        msg,
			URL:            url,
			OrganizationID: n.OrganizationID,
			SavedQueryID:   n.SavedQueryID,
			ThreadID:       n.ThreadID,
			MessageID:      n.MessageID,
			CallID:         n.CallID,
			PushID:         n.DedupeKey,
		},
	})
	if err != nil {
		golog.Errorf("Error while serializing android notification data: %s", err)
	}

	return &snsNotification{
		DefaultMessage: msg,
		IOSSandBox:     string(isNotifData),
		IOS:            string(isNotifData),
		Android:        string(androidNotifData),
	}
}
