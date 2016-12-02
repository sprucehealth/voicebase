package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
)

// Keys for settings
const (
	ConfigKeyForwardingList                = "forwarding_list"
	ConfigKeySendCallsToVoicemail          = "send_calls_to_voicemail"
	ConfigKeyVoicemailOption               = "voicemail_option"
	ConfigKeyTranscribeVoicemail           = "transcribe_voicemail"
	ConfigKeyAfterHoursVociemailEnabled    = "afterhours_voicemail_enabled"
	ConfigKeyForwardingListTimeout         = "forwarding_list_timeout"
	ConfigKeyPauseBeforeCallConnect        = "pause_before_call_connect"
	ConfigKeyExposeCaller                  = "expose_caller"
	ConfigKeyCallScreeningEnabled          = "call_screening_enabled"
	ConfigKeyDefaultProvisionedPhoneNumber = "default_provisioned_phone_number"
)

//
//	CALL LIST CONFIGURATION
//

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

var ForwardingListTimeoutConfig = &settings.Config{
	Title:          "Timeout for directing calls to voicemail",
	Key:            ConfigKeyForwardingListTimeout,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	AllowSubkeys:   true,
	Type:           settings.ConfigType_INTEGER,
	Config: &settings.Config_Integer{
		Integer: &settings.IntegerConfig{
			Default: &settings.IntegerValue{
				Value: 15,
			},
		},
	},
}

var ExposeCallerConfig = &settings.Config{
	Title:          "Expose/Hide ID of the actual caller to a Spruce phone number",
	Key:            ConfigKeyExposeCaller,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
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

var CallScreeningConfig = &settings.Config{
	Title:          "Enable/Disable call screening to ensure human answers call on provider side. Disabling this will prevent voicemails from being captured in the Spruce app as all calls will be treated as being answered.",
	Key:            ConfigKeyCallScreeningEnabled,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	AllowSubkeys:   true,
	Type:           settings.ConfigType_BOOLEAN,
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

const (
	VoicemailOptionDefault = "voicemail_option_default"
	VoicemailOptionCustom  = "voicemail_option_custom"
)

var VoicemailOptionConfig = &settings.Config{
	Title:          "Custom or default voicemail",
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

// PauseBeforeCallConnectConfig defines the config for how long to wait
// before connecting an incoming call to an organization. The reason that this
// configuration exists is to work around a known issue with providers forwarding
// google voice numbers into spruce, where that setup only works if there is a 2 second
// pause before the call is connected.
var PauseBeforeCallConnectConfig = &settings.Config{
	Title:          "Number of seconds to pause before connecting the call",
	Key:            ConfigKeyPauseBeforeCallConnect,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	AllowSubkeys:   true,
	Type:           settings.ConfigType_INTEGER,
	Config: &settings.Config_Integer{
		Integer: &settings.IntegerConfig{
			Default: &settings.IntegerValue{
				Value: 0,
			},
		},
	},
}

//
// AFTER HOURS CONFIGURATION
//

var AfterHoursVoicemailEnabledConfig = &settings.Config{
	Title:          "Enable/disable afterhours voicemail",
	Key:            ConfigKeyAfterHoursVociemailEnabled,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
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

//
// ORGANIZATION WIDE CONFIGURATION
//

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

var DefaultProvisionedPhoneNumberConfig = &settings.Config{
	Title:          "Default outgoing provisioned phone number",
	Key:            ConfigKeyDefaultProvisionedPhoneNumber,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	AllowSubkeys:   false,
	Type:           settings.ConfigType_TEXT,
	Config: &settings.Config_Text{
		Text: &settings.TextConfig{},
	},
}
