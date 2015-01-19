package notify

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/golog"
)

func (n *NotificationManager) pushNotificationToUser(
	accountID int64,
	role string,
	msg *Message,
	notificationCount int64) error {

	if n.snsClient == nil {
		golog.Errorf("No sns client configured when one was expected")
		return nil
	}

	// identify all devices associated with this user
	pushConfigDataList, err := n.dataAPI.GetPushConfigDataForAccount(accountID)
	if err != nil {
		return err
	}

	// render the notification and push for each device and send to each device
	for _, pushConfigData := range pushConfigDataList {

		// lookup config to use to determine endpoint to push to
		configName := config.DetermineNotificationConfigName(pushConfigData.Platform, pushConfigData.AppType, pushConfigData.AppEnvironment)
		notificationConfig, err := n.notificationConfigs.Get(configName)
		if err != nil {
			return err
		}

		pushEndpoint := pushConfigData.PushEndpoint
		// send push notifications in parallel
		go func() {
			err = n.snsClient.Publish(
				renderNotification(notificationConfig, msg.ShortMessage, notificationCount),
				pushEndpoint)
			if err != nil {
				// don't return err so that we attempt to send push to as many devices as possible
				n.statPushFailed.Inc(1)
				golog.Errorf("Error sending push notification: %s", err)
			} else {
				n.statPushSent.Inc(1)
			}
		}()
	}

	return nil
}

func renderNotification(notificationConfig *config.NotificationConfig, message string, badgeCount int64) *snsNotification {
	snsNote := &snsNotification{
		DefaultMessage: message,
	}
	switch notificationConfig.Platform {
	case common.Android:
		snsNote.Android = &androidPushNotification{
			Message: snsNote.DefaultMessage,
		}

	case common.IOS:
		iosNotification := &iOSPushNotification{
			Badge: badgeCount,
			Alert: snsNote.DefaultMessage,
		}
		if notificationConfig.IsApnsSandbox {
			snsNote.IOSSandBox = iosNotification
		} else {
			snsNote.IOS = iosNotification
		}
	}

	return snsNote
}
