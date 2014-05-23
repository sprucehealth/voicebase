package notify

import (
	"carefront/api"
	"carefront/common/config"
	"carefront/libs/aws/sns"
)

type notificationHandler struct {
	dataApi             api.DataAPI
	notificationConfigs map[string]*config.NotificationConfig
	snsClient           *sns.SNS
}

func NewNotificationHandler(dataApi api.DataAPI, configs map[string]*config.NotificationConfig, snsClient *sns.SNS) *notificationHandler {
	return &notificationHandler{
		dataApi:             dataApi,
		notificationConfigs: configs,
		snsClient:           snsClient,
	}
}
