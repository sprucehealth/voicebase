package invite

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyOrganizationCode                           = "organization_code_enabled"
	ConfigKeyTwoFactorVerificationForSecureConversation = "two_factor_verification_secure_conversation"
	ConfigKeyPatientInviteChannelPreference             = "patient_invite_channel_preference"
)

// OrganizationCodeConfig represents the config for configuring whether organization codes are available for the org
var OrganizationCodeConfig = &settings.Config{
	Title:          "Enable/Disable organization code availability at org level",
	AllowSubkeys:   false,
	Key:            ConfigKeyOrganizationCode,
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

// TwoFactorVerificationForSecureConversationConfig represents the config
// controlling whether or not the organization requires two factor verification
// of patient information during secure conversation creation.
var TwoFactorVerificationForSecureConversationConfig = &settings.Config{
	Title:          "Require two factor verification of patient information (email and phone) during secure conversation creation.",
	AllowSubkeys:   false,
	Key:            ConfigKeyTwoFactorVerificationForSecureConversation,
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

const (
	PatientInviteChannelPreferenceEmail = "patient_invite_channel_preference_email"
	PatientInviteChannelPreferenceSMS   = "patient_invite_channel_preference_sms"
)

// PatientInviteChannelPreferenceConfig represents the config
// controlling whether patient invite preference for an organization
// is email or sms when both pieces of information are provided, but only
// one of them is required.
var PatientInviteChannelPreferenceConfig = &settings.Config{
	Title:          "Invite delivery channel preference when only phone or email requires for patient invites.",
	Key:            ConfigKeyPatientInviteChannelPreference,
	AllowSubkeys:   false,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_SingleSelect{
		SingleSelect: &settings.SingleSelectConfig{
			Items: []*settings.Item{
				{
					ID:    PatientInviteChannelPreferenceEmail,
					Label: "Email",
				},
				{
					ID:    PatientInviteChannelPreferenceSMS,
					Label: "SMS",
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: PatientInviteChannelPreferenceEmail,
				},
			},
		},
	},
}
