package notify

import (
	"carefront/api"
	"carefront/common/config"
	"carefront/libs/aws/sns"

	"github.com/samuel/go-metrics/metrics"
)

func pushNotificationToUser(snsClient *sns.SNS, notificationConfigs map[string]*config.NotificationConfig, event interface{}, accountId int64, dataApi api.DataAPI, statPushFailed, statPushSent metrics.Counter) error {

	return nil
}
