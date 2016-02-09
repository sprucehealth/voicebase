package server

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/settings"
)

var grpcErrorf = grpc.Errorf

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
		// ensure that at least one multi-select option is specified
		if transformedValue.GetMultiSelect() == nil || len(transformedValue.GetMultiSelect().GetItems()) == 0 {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpcErrorf(settings.InvalidUserValue, "Please select an option")
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

		// ensure that one singe-select option is specified
		if transformedValue.GetSingleSelect() == nil || transformedValue.GetSingleSelect().Item == nil {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpcErrorf(settings.InvalidUserValue, "Please select an option")
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
		if transformedValue.GetStringList() == nil || len(transformedValue.GetStringList().Values) == 0 {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpcErrorf(settings.InvalidUserValue, "Please specify at least one entry")
		} else {
			// ensure that there is at least one valid entry
			atLeastOneValidEntry := false
			for _, item := range transformedValue.GetStringList().Values {
				if len(strings.TrimSpace(item)) != 0 {
					atLeastOneValidEntry = true
					break
				}
			}
			if !atLeastOneValidEntry && !config.OptionalValue {
				return nil, grpcErrorf(settings.InvalidUserValue, "Please specify at least one entry")
			}
		}

	default:
		return nil, fmt.Errorf("Unsupported config type %s", config.Type.String())
	}

	return transformedValue, nil
}