package notify

import (
	"carefront/common/config"
	"fmt"
)

func (n *NotificationManager) pushNotificationToUser(accountId int64, event interface{}, notificationCount int64) error {
	if n.snsClient == nil {
		return nil
	}

	// identify all devices associated with this user
	pushConfigDatas, err := n.dataApi.GetPushConfigDataForAccount(accountId)
	if err != nil {
		return err
	}

	// render the notification and push for each device and send to each device
	for _, pushConfigData := range pushConfigDatas {

		// lookup config to use to determine endpoint to push to
		configName := config.DetermineNotificationConfigName(pushConfigData.Platform, pushConfigData.AppType, pushConfigData.AppEnvironment)
		notificationConfig, ok := n.notificationConfigs[configName]
		if !ok {
			return fmt.Errorf("Unable to determine notification config to use")
		}

		err := n.snsClient.Publish(getNotificationViewForEvent(event).renderPush(pushConfigData.Platform, event, n.dataApi, notificationCount), notificationConfig.SNSApplicationEndpoint)
		if err != nil {
			n.statPushFailed.Inc(1)
			return err
		} else {
			n.statPushSent.Inc(1)
		}
	}

	return nil
}
