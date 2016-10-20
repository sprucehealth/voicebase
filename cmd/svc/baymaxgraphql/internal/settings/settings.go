package settings

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyCarePlans                = "care_plans_enabled"
	ConfigKeyCreateSecureThread       = "secure_threads_enabled"
	ConfigKeyFilteredTabsInInbox      = "filtered_tabs_in_inbox"
	ConfigKeyShakeToMarkThreadsAsRead = "shake_to_mark_threads_read"
	ConfigKeyTeamConversations        = "team_conversations_enabled"
	ConfigKeyVideoCalling             = "video_calling_enabled"
	ConfigKeyVisitAttachments         = "visit_attachments_enabled"
	ConfigKeyPayments                 = "payments_enabled"
	ConfigKeyScheduledMessages        = "scheduled_messages_enabled"
)

// TeamConversationsConfig represents the config controlling whether or not team conversations is enabled at the org level
var TeamConversationsConfig = &settings.Config{
	Title:          "Enable/disable team conversations",
	AllowSubkeys:   false,
	Key:            ConfigKeyTeamConversations,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// SecureThreadsConfig represents the config controlling whether or not secure conversations are enabled at the org level
var SecureThreadsConfig = &settings.Config{
	Title:          "Enable/disable secure threads",
	AllowSubkeys:   false,
	Key:            ConfigKeyCreateSecureThread,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// ShakeToMarkThreadsAsReadConfig represents the config for controlling whether or not an Organization
// allows its members to shake their devices to mark all threads as read
var ShakeToMarkThreadsAsReadConfig = &settings.Config{
	Title:          "Enable/disable shake to mark threads as read",
	AllowSubkeys:   false,
	Key:            ConfigKeyShakeToMarkThreadsAsRead,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// VisitAttachmentsConfig represents the config for controlling visit attachments to be
// enabled on secure conversations at an org level.
var VisitAttachmentsConfig = &settings.Config{
	Title:          "Enable/Disable visit attachments at thread level",
	AllowSubkeys:   false,
	Key:            ConfigKeyVisitAttachments,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// CarePlansConfig represents the config for configuring care plans to be enabled
// at an org level.
var CarePlansConfig = &settings.Config{
	Title:          "Enable/Disable care plans at thread level",
	AllowSubkeys:   false,
	Key:            ConfigKeyCarePlans,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// FilteredTabsInInboxConfig represents the config for configuring whether filtered
// tabs are enabled or not at an org level for the inbox.
var FilteredTabsInInboxConfig = &settings.Config{
	Title:          "Enable/Disable care plans at thread level",
	AllowSubkeys:   false,
	Key:            ConfigKeyFilteredTabsInInbox,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// VideoCallingConfig represents the config for configuring whether video
// calling is enabled for an organization.
var VideoCallingConfig = &settings.Config{
	Title:          "Enable/Disable video calling at org level",
	AllowSubkeys:   false,
	Key:            ConfigKeyVideoCalling,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// PaymentsConfig represents the config controlling whether or not payments is enabled at the org level
var PaymentsConfig = &settings.Config{
	Title:          "Enable/disable payment support",
	AllowSubkeys:   false,
	Key:            ConfigKeyPayments,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// ScheduledMessagesConfig represents the config controlling whether or not scheduled messages are enabled at the org level
var ScheduledMessagesConfig = &settings.Config{
	Title:          "Enable/disable scheduled messages",
	AllowSubkeys:   false,
	Key:            ConfigKeyScheduledMessages,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: false,
			},
		},
	},
}
