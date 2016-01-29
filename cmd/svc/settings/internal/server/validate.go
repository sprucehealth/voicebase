package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/settings"
)

func validateValueAgainstConfig(value *settings.Value, config *models.Config) (*models.Value, error) {

	if value.Key == nil {
		return nil, errors.New("key required for value")
	} else if !config.AllowSubkeys && value.Key.Subkey != "" {
		return nil, fmt.Errorf("no subkeys allowed for %s but subkey specified for value %s", config.Key, value.Key)
	} else if value.Key.Key != config.Key {
		return nil, fmt.Errorf("config key %s does not match value key %s", config.Key, value.Key.Key)
	} else if config.AllowSubkeys && value.Key.Subkey == "" {
		return nil, fmt.Errorf("no subkey specified when one is required for config %s", config.Key)
	}

	transformedValue := transformValueToModel(value)
	transformedValue.Config = config

	switch config.Type {
	case models.ConfigType_MULTI_SELECT:
		if transformedValue.GetMultiSelect() == nil {
			return nil, fmt.Errorf("No options selected for value %s when expected one to be set", transformedValue.Key)
		} else if len(transformedValue.GetMultiSelect().Items) == 0 {
			return nil, fmt.Errorf("No options selected for value %s when expected one to be set", transformedValue.Key)
		}

		// ensure that all options selected in multi-select are from the list of available options
		for _, item := range transformedValue.GetMultiSelect().GetItems() {
			// ensure that id is present in the config
			present := false
			for _, option := range config.GetMultiSelect().Items {
				if option.ID == item.ID {
					present = true
					break
				}
			}
			if !present {
				return nil, fmt.Errorf("Option selected is not one of the selectable options for value %s", transformedValue.Key)
			}
		}
	case models.ConfigType_SINGLE_SELECT:
		if transformedValue.GetSingleSelect() == nil {
			return nil, fmt.Errorf("No options selected for value %s when expected one to be set", transformedValue.Key)
		} else if transformedValue.GetSingleSelect().Item == nil {
			return nil, fmt.Errorf("No options selected for value %s when expected one to be set", transformedValue.Key)
		}
		present := false
		for _, option := range config.GetSingleSelect().Items {
			if option.ID == transformedValue.GetSingleSelect().Item.ID {
				present = true
				break
			}
		}
		if !present {
			return nil, fmt.Errorf("Option selected is not one of the selectable options for value %s", transformedValue.Key)
		}
	case models.ConfigType_BOOLEAN:
		if transformedValue.GetBoolean() == nil {
			return nil, fmt.Errorf("No boolean value specified for %s", transformedValue.Key)
		}
	case models.ConfigType_STRING_LIST:
		if transformedValue.GetStringList() == nil {
			return nil, fmt.Errorf("Expected string list to be defined in value %s", transformedValue.Key)
		}
	default:
		return nil, fmt.Errorf("Unsupported config type %s", config.Type.String())
	}

	return transformedValue, nil
}
