package notify

import "carefront/common/config"

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
		notificationConfig, err := n.notificationConfigs.Get(configName)
		if err != nil {
			return err
		}

		err = n.snsClient.Publish(getNotificationViewForEvent(event).renderPush(notificationConfig, event, n.dataApi, notificationCount), notificationConfig.SNSApplicationEndpoint)
		if err != nil {
			n.statPushFailed.Inc(1)
			return err
		} else {
			n.statPushSent.Inc(1)
		}
	}

	return nil
}
