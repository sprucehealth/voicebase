package server

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
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

	switch cfg := config.Config.(type) {
	case *models.Config_MultiSelect:
		// ensure that at least one multi-select option is specified
		if transformedValue.GetMultiSelect() == nil || len(transformedValue.GetMultiSelect().GetItems()) == 0 {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpc.Errorf(settings.InvalidUserValue, "Please select an option")
		}

		// ensure that all options selected in multi-select are from the list of available options
		for _, item := range transformedValue.GetMultiSelect().GetItems() {
			// ensure that id is present in the config
			present := false
			var optionSelected *models.Item
			for _, option := range cfg.MultiSelect.Items {
				if option.ID == item.ID {
					present = true
					optionSelected = option
					break
				}
			}
			if !present {
				return nil, fmt.Errorf("Option selected is not one of the selectable options for value %s", transformedValue.Key)
			}

			if optionSelected.AllowFreeText && optionSelected.FreeTextRequired && len(strings.TrimSpace(item.FreeTextResponse)) == 0 {
				return nil, fmt.Errorf("Selection requires free text but no free text set for value %s", transformedValue.Key)
			}
		}

	case *models.Config_SingleSelect:
		// ensure that one singe-select option is specified
		if transformedValue.GetSingleSelect() == nil || transformedValue.GetSingleSelect().Item == nil {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpc.Errorf(settings.InvalidUserValue, "Please select an option")
		}

		present := false
		var optionSelected *models.Item
		for _, option := range cfg.SingleSelect.Items {
			if option.ID == transformedValue.GetSingleSelect().Item.ID {
				present = true
				optionSelected = option
				break
			}
		}
		if !present {
			return nil, fmt.Errorf("Option selected is not one of the selectable options for value %s", transformedValue.Key)
		}

		if optionSelected.AllowFreeText && optionSelected.FreeTextRequired && len(strings.TrimSpace(transformedValue.GetSingleSelect().Item.FreeTextResponse)) == 0 {
			return nil, fmt.Errorf("Selection requires free text but no free text set for value %s", transformedValue.Key)
		}

	case *models.Config_Boolean:
		if transformedValue.GetBoolean() == nil {
			return nil, fmt.Errorf("No boolean value specified for %s", transformedValue.Key)
		}
	case *models.Config_Integer:
		if transformedValue.GetInteger() == nil {
			return nil, fmt.Errorf("No integer value specified for %s", transformedValue.Key)
		}
	case *models.Config_Text:
		text := transformedValue.GetText()
		if text == nil {
			return nil, fmt.Errorf("No text value specified for %s", transformedValue.Key)
		}
		if validatedText, err := validateTextRequirements(cfg.Text.Requirements, text.Value); err != nil {
			if e, ok := errors.Cause(err).(errInvalidValue); ok {
				return nil, grpc.Errorf(settings.InvalidUserValue, e.Error())
			}
			return nil, errors.Trace(err)
		} else {
			transformedValue.GetText().Value = validatedText
		}
	case *models.Config_StringList:
		if transformedValue.GetStringList() == nil || len(transformedValue.GetStringList().Values) == 0 {
			if config.OptionalValue {
				return transformedValue, nil
			}
			return nil, grpc.Errorf(settings.InvalidUserValue, "Please specify at least one entry")
		}
		values := transformedValue.GetStringList().Values
		// remove empty entries
		nonEmptyValues := make([]string, 0, len(values))
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v != "" {
				nonEmptyValues = append(nonEmptyValues, v)
			}
		}
		if len(nonEmptyValues) == 0 && !config.OptionalValue {
			return nil, grpc.Errorf(settings.InvalidUserValue, "Please specify at least one entry")
		}
		transformedValue.GetStringList().Values = nonEmptyValues

		if err := validateStringListRequirements(cfg.StringList.Requirements, transformedValue.GetStringList()); err != nil {
			if e, ok := errors.Cause(err).(errInvalidValue); ok {
				return nil, grpc.Errorf(settings.InvalidUserValue, e.Error())
			}
			return nil, errors.Trace(err)
		}
	default:
		return nil, errors.Errorf("Unsupported config type %T", cfg)
	}

	return transformedValue, nil
}

type errInvalidValue struct {
	value  string
	reason string
}

func (e errInvalidValue) Error() string {
	if e.value != "" {
		return fmt.Sprintf("The value %q %s", e.value, e.reason)
	}
	return fmt.Sprintf("The provided value is invalid, %s", e.reason)
}

func validateStringListRequirements(req *models.StringListRequirements, val *models.StringListValue) error {
	if req == nil {
		return nil
	}

	if req.MinValues > 0 && len(val.Values) < int(req.MinValues) {
		return errInvalidValue{reason: fmt.Sprintf("must have at least %d values", req.MinValues)}
	}
	if req.MaxValues > 0 && len(val.Values) > int(req.MaxValues) {
		return errInvalidValue{reason: fmt.Sprintf("must have at most %d values", req.MaxValues)}
	}
	if req.TextRequirements != nil {
		for i, v := range val.Values {
			if validatedText, err := validateTextRequirements(req.TextRequirements, v); err != nil {
				return errors.Trace(err)
			} else {
				val.Values[i] = validatedText
			}
		}
	}
	return nil
}

func validateTextRequirements(req *models.TextRequirements, text string) (string, error) {
	if req == nil {
		return "", nil
	}
	if req.MatchRegexp != "" {
		// The config should have been validated much earlier so the regex compile shouldn't ever fail
		re, err := regexp.Compile(req.MatchRegexp)
		if err != nil {
			golog.Errorf("Regular expression %q is invalid: %s", req.MatchRegexp, err)
		} else if !re.MatchString(text) {
			return "", errInvalidValue{value: text, reason: "does not match expected format"}
		}
	}
	switch req.TextType {
	case models.TextType_PHONE:
		pn, err := phone.ParseNumber(text)
		if err != nil {
			golog.Errorf("Invalid US phone number '%s' : %s", text, err)
			return "", errInvalidValue{value: text, reason: "Invalid US phone number"}
		}
		return pn.String(), nil
	}
	return text, nil
}
