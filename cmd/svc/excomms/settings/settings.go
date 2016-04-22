package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
)

const (
	ConfigKeyForwardingList           = "forwarding_list"
	ConfigKeySendCallsToVoicemail     = "send_calls_to_voicemail"
	ConfigKeyVoicemailOption          = "voicemail_option"
	ConfigKeyTranscribeVoicemail      = "transcribe_voicemail"
	ConfigKeyIncomingCallOption       = "incoming_call_option"
	ConfigKeyAfterHoursGreetingOption = "afterhours_greeting_option"
)

const (
	IncomingCallOptionCallForwardingList   = "call_forwarding_list"
	IncomingCallOptionAfterHoursCallTriage = "afterhours_call_triage"
)

//
// TOP LEVEL INCOMING CALL CONFIGURATION
//

var IncomingCallBehaviorConfig = &settings.Config{
	Title:          "Incoming call behavior",
	Description:    "How to act on an incoming call",
	Key:            ConfigKeyIncomingCallOption,
	AllowSubkeys:   true,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_SingleSelect{
		SingleSelect: &settings.SingleSelectConfig{
			Items: []*settings.Item{
				{
					ID:    IncomingCallOptionCallForwardingList,
					Label: "Call Forwarding List",
				},
				{
					ID:    IncomingCallOptionAfterHoursCallTriage,
					Label: "Direct to after hours phone tree",
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: IncomingCallOptionCallForwardingList,
				},
			},
		},
	},
}

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

//
// AFTER HOURS CONFIGURATION
//

const (
	AfterHoursGreetingOptionDefault = "afterhours_greeting_option_default"
	AfterHoursGreetingOptionCustom  = "afterhours_greeting_option_custom"
)

var AfterHoursGreetingOptionConfig = &settings.Config{
	Title:          "After hours greeting: Custom or voicemail?",
	Key:            ConfigKeyAfterHoursGreetingOption,
	AllowSubkeys:   true,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_SingleSelect{
		SingleSelect: &settings.SingleSelectConfig{
			Items: []*settings.Item{
				{
					ID:    AfterHoursGreetingOptionDefault,
					Label: "Default",
				},
				{
					ID:               AfterHoursGreetingOptionCustom,
					Label:            "Custom",
					AllowFreeText:    true,
					FreeTextRequired: true,
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: AfterHoursGreetingOptionDefault,
				},
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
