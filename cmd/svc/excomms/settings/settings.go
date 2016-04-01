package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
)

const (
	ConfigKeyForwardingList       = "forwarding_list"
	ConfigKeyVoicemailOption      = "voicemail_option"
	ConfigKeySendCallsToVoicemail = "send_calls_to_voicemail"
	ConfigKeyTranscribeVoicemail  = "transcribe_voicemail"
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

const (
	VoicemailOptionDefault = "voicemail_option_default"
	VoicemailOptionCustom  = "voicemail_option_custom"
)

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
					ID:    VoicemailOptionDefault,
					Label: "Default",
				},
				{
					ID:               VoicemailOptionCustom,
					Label:            "Custom",
					AllowFreeText:    true,
					FreeTextRequired: true,
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: VoicemailOptionDefault,
				},
			},
		},
	},
}

var SendCallsToVoicemailConfig = &settings.Config{
	Title:          "Send all calls to voicemail",
	Key:            ConfigKeySendCallsToVoicemail,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION, settings.OwnerType_INTERNAL_ENTITY},
	AllowSubkeys:   true,
	Type:           settings.ConfigType_BOOLEAN,
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: false,
			},
		},
	},
}

var TranscribeVoicemailConfig = &settings.Config{
	Title:          "Transcribe Voicemails",
	Key:            ConfigKeyTranscribeVoicemail,
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
