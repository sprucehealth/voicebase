package settings

import (
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
)

// ReceiveNotificationsConfig represents the config controlling if notifications are disabled or not
var ReceiveNotificationsConfig = &settings.Config{
	Title:          "Receive notifications",
	AllowSubkeys:   false,
	Key:            notification.ReceiveNotificationsSettingsKey,
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

// NotificationPreferenceConfig represents the config controlling when notifications are sent
var NotificationPreferenceConfig = &settings.Config{
	Title:          "Notification Preference",
	AllowSubkeys:   false,
	Key:            notification.NotificationPreferencesSettingsKey,
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
