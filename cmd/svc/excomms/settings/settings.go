package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
)

const (
	ConfigKeyForwardingList       = "forwarding_list"
	ConfigKeyVoicemailOption      = "voicemail_option"
	ConfigKeySendCallsToVoicemail = "send_calls_to_voicemail"
)

var NumbersToRingConfig = &settings.Config{
	Title:        "Numbers to ring",
	Description:  "You can add up to five phone numbers",
	Key:          ConfigKeyForwardingList,
	AllowSubkeys: true,
	Type:         settings.ConfigType_STRING_LIST,
	Config: &settings.Config_StringList{
		StringList: &settings.StringListConfig{},
	},
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
}

var VoicemailOptionConfig = &settings.Config{
	Title:          "Set outgoing message",
	Key:            ConfigKeyVoicemailOption,
	AllowSubkeys:   true,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_SingleSelect{
		SingleSelect: &settings.SingleSelectConfig{
			Items: []*settings.Item{
				{
					ID:    "voicemail_option_default",
					Label: "Default",
				},
				{
					ID:    "voicemail_option_custom",
					Label: "Custom",
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: "voicemail_option_default",
				},
			},
		},
	},
}

var SendCallsToVoicemailConfig = &settings.Config{
	Title:          "Send all calls to voicemail",
	Key:            ConfigKeySendCallsToVoicemail,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	AllowSubkeys:   false,
	Type:           settings.ConfigType_BOOLEAN,
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: false,
			},
		},
	},
}
