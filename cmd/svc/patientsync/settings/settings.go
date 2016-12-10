package settings

import "github.com/sprucehealth/backend/svc/settings"

const (
	ThreadTypeOptionStandard    = "thread_type_option_standard"
	ThreadTypeOptionSecure      = "thread_type_option_secure"
	ConfigKeyThreadTypeOption   = "patient_sync_thread_type"
	ConfigKeyAutoInvitePatients = "auto_invite_patients_on_sync"
)

// ThreadTypeOptionConfig specifies the type of threads to create for an organization
// in the event of a patient sync from an EMR or external data source.
var ThreadTypeOptionConfig = &settings.Config{
	Title:          "Specify the types of threads to create for a patientsync",
	Key:            ConfigKeyThreadTypeOption,
	AllowSubkeys:   false,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_SingleSelect{
		SingleSelect: &settings.SingleSelectConfig{
			Items: []*settings.Item{
				{
					ID:    ThreadTypeOptionStandard,
					Label: "Standard conversations",
				},
				{
					ID:    ThreadTypeOptionSecure,
					Label: "Secure conversations",
				},
			},
			Default: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: ThreadTypeOptionSecure,
				},
			},
		},
	},
}

// AutoInvitePatientsOnSyncConfig specifies whether or not to auto invite
// patients to use Spruce when they are imported into Spruce from external source.
var AutoInvitePatientsOnSyncConfig = &settings.Config{
	Title:          "Flag to auto invite patients to Spruce on sync",
	Key:            ConfigKeyAutoInvitePatients,
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
