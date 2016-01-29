package main

import (
	"github.com/sprucehealth/backend/svc/settings"
)

const (
	configKeyReceiveNotifications   = "receive_notifications"
	configKeyNotificationPreference = "notification_preference"
)

var receiveNotificationsConfig = &settings.Config{
	Title:          "Receive notifications",
	AllowSubkeys:   false,
	Key:            configKeyReceiveNotifications,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

var notificationPreferenceConfig = &settings.Config{
	Title:          "Notification Preference",
	AllowSubkeys:   false,
	Key:            configKeyNotificationPreference,
	Type:           settings.ConfigType_MULTI_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	Config: &settings.Config_MultiSelect{
		MultiSelect: &settings.MultiSelectConfig{
			Items: []*settings.Item{
				{
					ID:    "notification_preference_all",
					Label: "All activity",
				},
				{
					ID:    "notification_preference_referenced_only",
					Label: "Notify for @references only",
				},
			},
			Default: &settings.MultiSelectValue{
				Items: []*settings.ItemValue{
					{
						ID: "notification_preference_all",
					},
				},
			},
		},
	},
}
