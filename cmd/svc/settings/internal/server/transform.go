package server

import (
	"regexp"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/settings"
)

func transformConfigToModel(config *settings.Config) (*models.Config, error) {
	m := &models.Config{
		Title:          config.Title,
		Description:    config.Description,
		Key:            config.Key,
		AllowSubkeys:   config.AllowSubkeys,
		Type:           models.ConfigType(models.ConfigType_value[config.Type.String()]),
		PossibleOwners: make([]models.OwnerType, len(config.PossibleOwners)),
		OptionalValue:  config.OptionalValue,
	}

	for i, po := range config.PossibleOwners {
		m.PossibleOwners[i] = models.OwnerType(models.OwnerType_value[po.String()])
	}

	switch config := config.Config.(type) {
	case *settings.Config_Boolean:
		m.Config = &models.Config_Boolean{
			Boolean: &models.BooleanConfig{
				Default: &models.BooleanValue{
					Value: config.Boolean.Default.Value,
				},
			},
		}
	case *settings.Config_Integer:
		m.Config = &models.Config_Integer{
			Integer: &models.IntegerConfig{
				Default: &models.IntegerValue{
					Value: config.Integer.Default.Value,
				},
			},
		}
	case *settings.Config_Text:
		req, err := transformTextRequirementsToModel(config.Text.Requirements)
		if err != nil {
			return nil, errors.Trace(err)
		}
		m.Config = &models.Config_Text{
			Text: &models.TextConfig{
				Default: &models.TextValue{
					Value: config.Text.Default.Value,
				},
				Requirements: req,
			},
		}
	case *settings.Config_MultiSelect:
		m.Config = &models.Config_MultiSelect{
			MultiSelect: &models.MultiSelectConfig{
				Items: make([]*models.Item, len(config.MultiSelect.Items)),
				Default: &models.MultiSelectValue{
					Items: make([]*models.ItemValue, len(config.MultiSelect.Default.Items)),
				},
			},
		}

		for i, item := range config.MultiSelect.Items {
			m.GetMultiSelect().Items[i] = &models.Item{
				ID:               item.ID,
				Label:            item.Label,
				AllowFreeText:    item.AllowFreeText,
				FreeTextRequired: item.FreeTextRequired,
			}
		}

		for i, item := range config.MultiSelect.Default.Items {
			m.GetMultiSelect().Default.Items[i] = &models.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case *settings.Config_SingleSelect:
		m.Config = &models.Config_SingleSelect{
			SingleSelect: &models.SingleSelectConfig{
				Items: make([]*models.Item, len(config.SingleSelect.Items)),
				Default: &models.SingleSelectValue{
					Item: &models.ItemValue{
						ID:               config.SingleSelect.Default.Item.ID,
						FreeTextResponse: config.SingleSelect.Default.Item.FreeTextResponse,
					},
				},
			},
		}
		for i, item := range config.SingleSelect.Items {
			m.GetSingleSelect().Items[i] = &models.Item{
				ID:               item.ID,
				Label:            item.Label,
				AllowFreeText:    item.AllowFreeText,
				FreeTextRequired: item.FreeTextRequired,
			}
		}
	case *settings.Config_StringList:
		req, err := transformStringListRequirementsToModel(config.StringList.Requirements)
		if err != nil {
			return nil, errors.Trace(err)
		}
		m.Config = &models.Config_StringList{
			StringList: &models.StringListConfig{
				Requirements: req,
			},
		}
		if config.StringList.Default != nil {
			m.GetStringList().Default = &models.StringListValue{
				Values: config.StringList.Default.Values,
			}
		}
	default:
		return nil, errors.Errorf("unknown setting config type %T", config)
	}
	return m, nil
}

func transformStringListRequirementsToModel(req *settings.StringListRequirements) (*models.StringListRequirements, error) {
	if req == nil {
		return nil, nil
	}
	textReq, err := transformTextRequirementsToModel(req.TextRequirements)
	if err != nil {
		return nil, err
	}
	return &models.StringListRequirements{
		TextRequirements: textReq,
		MinValues:        req.MinValues,
		MaxValues:        req.MaxValues,
	}, nil
}

func transformTextRequirementsToModel(req *settings.TextRequirements) (*models.TextRequirements, error) {
	if req == nil {
		return nil, nil
	}
	// Make sure regexp is valid
	if req.MatchRegexp != "" {
		_, err := regexp.Compile(req.MatchRegexp)
		if err != nil {
			return nil, errors.Errorf("Regular expression %q is invalid: %s", req.MatchRegexp, err)
		}
	}

	var textType models.TextType
	switch req.TextType {
	case settings.TextType_PHONE:
		textType = models.TextType_PHONE
	case settings.TextType_FREE_TEXT:
		textType = models.TextType_FREE_TEXT
	default:
		return nil, errors.Errorf("unknown text type %s", req.TextType)
	}

	return &models.TextRequirements{
		MatchRegexp: req.MatchRegexp,
		TextType:    textType,
	}, nil
}

func transformModelToConfig(config *models.Config) (*settings.Config, error) {
	c := &settings.Config{
		Title:          config.Title,
		Description:    config.Description,
		Key:            config.Key,
		AllowSubkeys:   config.AllowSubkeys,
		Type:           settings.ConfigType(settings.ConfigType_value[config.Type.String()]),
		PossibleOwners: make([]settings.OwnerType, len(config.PossibleOwners)),
		OptionalValue:  config.OptionalValue,
	}

	for i, po := range config.PossibleOwners {
		c.PossibleOwners[i] = settings.OwnerType(settings.OwnerType_value[po.String()])
	}

	switch config := config.Config.(type) {
	case *models.Config_Boolean:
		c.Config = &settings.Config_Boolean{
			Boolean: &settings.BooleanConfig{
				Default: &settings.BooleanValue{
					Value: config.Boolean.Default.Value,
				},
			},
		}
	case *models.Config_Integer:
		c.Config = &settings.Config_Integer{
			Integer: &settings.IntegerConfig{
				Default: &settings.IntegerValue{
					Value: config.Integer.Default.Value,
				},
			},
		}
	case *models.Config_Text:
		requirements, err := transformTextRequirementsToResponse(config.Text.Requirements)
		if err != nil {
			return nil, errors.Trace(err)
		}
		c.Config = &settings.Config_Text{
			Text: &settings.TextConfig{
				Default: &settings.TextValue{
					Value: config.Text.Default.Value,
				},
				Requirements: requirements,
			},
		}
	case *models.Config_MultiSelect:
		c.Config = &settings.Config_MultiSelect{
			MultiSelect: &settings.MultiSelectConfig{
				Items: make([]*settings.Item, len(config.MultiSelect.Items)),
				Default: &settings.MultiSelectValue{
					Items: make([]*settings.ItemValue, len(config.MultiSelect.Default.Items)),
				},
			},
		}
		for i, item := range config.MultiSelect.Items {
			c.GetMultiSelect().Items[i] = &settings.Item{
				ID:               item.ID,
				AllowFreeText:    item.AllowFreeText,
				Label:            item.Label,
				FreeTextRequired: item.FreeTextRequired,
			}
		}
		for i, item := range config.MultiSelect.Default.Items {
			c.GetMultiSelect().Default.Items[i] = &settings.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case *models.Config_SingleSelect:
		c.Config = &settings.Config_SingleSelect{
			SingleSelect: &settings.SingleSelectConfig{
				Items: make([]*settings.Item, len(config.SingleSelect.Items)),
				Default: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               config.SingleSelect.Default.Item.ID,
						FreeTextResponse: config.SingleSelect.Default.Item.FreeTextResponse,
					},
				},
			},
		}
		for i, item := range config.SingleSelect.Items {
			c.GetSingleSelect().Items[i] = &settings.Item{
				ID:               item.ID,
				AllowFreeText:    item.AllowFreeText,
				Label:            item.Label,
				FreeTextRequired: item.FreeTextRequired,
			}
		}
	case *models.Config_StringList:
		requirements, err := transformStringListRequirementsToResponse(config.StringList.Requirements)
		if err != nil {
			return nil, errors.Trace(err)
		}

		c.Config = &settings.Config_StringList{
			StringList: &settings.StringListConfig{
				Requirements: requirements,
			},
		}
		if config.StringList.Default != nil {
			c.GetStringList().Default = &settings.StringListValue{
				Values: config.StringList.Default.Values,
			}
		}
	default:
		return nil, errors.Errorf("unknown config type %T", config)
	}

	return c, nil
}

func transformStringListRequirementsToResponse(req *models.StringListRequirements) (*settings.StringListRequirements, error) {
	if req == nil {
		return nil, nil
	}

	textRequirements, err := transformTextRequirementsToResponse(req.TextRequirements)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &settings.StringListRequirements{
		TextRequirements: textRequirements,
		MinValues:        req.MinValues,
		MaxValues:        req.MaxValues,
	}, nil
}

func transformTextRequirementsToResponse(req *models.TextRequirements) (*settings.TextRequirements, error) {
	if req == nil {
		return nil, nil
	}

	var textType settings.TextType
	switch req.TextType {
	case models.TextType_PHONE:
		textType = settings.TextType_PHONE
	case models.TextType_FREE_TEXT:
		textType = settings.TextType_FREE_TEXT
	default:
		return nil, errors.Errorf("unknown text type %s", req.TextType)
	}

	return &settings.TextRequirements{
		MatchRegexp: req.MatchRegexp,
		TextType:    textType,
	}, nil
}

func transformModelToValue(value *models.Value, currentConfig *models.Config) *settings.Value {
	v := &settings.Value{
		Key: &settings.ConfigKey{
			Key:    value.Key.Key,
			Subkey: value.Key.Subkey,
		},
		Type: settings.ConfigType(settings.ConfigType_value[value.Config.Type.String()]),
	}

	switch value.Config.Type {
	case models.ConfigType_BOOLEAN:
		v.Value = &settings.Value_Boolean{
			Boolean: &settings.BooleanValue{},
		}
		if value.GetBoolean() != nil {
			v.GetBoolean().Value = value.GetBoolean().Value
		}
	case models.ConfigType_INTEGER:
		v.Value = &settings.Value_Integer{
			Integer: &settings.IntegerValue{},
		}
		if value.GetInteger() != nil {
			v.GetInteger().Value = value.GetInteger().Value
		}
	case models.ConfigType_TEXT:
		v.Value = &settings.Value_Text{
			Text: &settings.TextValue{},
		}
		if w := value.GetText(); w != nil {
			v.GetText().Value = w.Value
			switch currentConfig.GetText().GetRequirements().TextType {
			case models.TextType_PHONE:
				pn, err := phone.Format(value.GetText().Value, phone.Pretty)
				if err != nil {
					golog.Errorf("Unable to parse phone number %s : %s", v.GetText().Value, err)
				}
				v.GetText().DisplayValue = pn
			}

		}

	case models.ConfigType_STRING_LIST:
		v.Value = &settings.Value_StringList{
			StringList: &settings.StringListValue{},
		}
		if value.GetStringList() != nil {
			v.GetStringList().Values = value.GetStringList().Values
			v.GetStringList().DisplayValues = make([]string, len(v.GetStringList().Values))
			for i, value := range v.GetStringList().Values {
				v.GetStringList().DisplayValues[i] = value
			}

			switch currentConfig.GetStringList().GetRequirements().TextRequirements.TextType {
			case models.TextType_PHONE:
				for i, value := range v.GetStringList().Values {
					pn, err := phone.Format(value, phone.Pretty)
					if err != nil {
						golog.Errorf("Unable to parse phone number %s : %s", value, err)
					}
					v.GetStringList().DisplayValues[i] = pn
				}
			}
		}

	case models.ConfigType_MULTI_SELECT:
		v.Value = &settings.Value_MultiSelect{
			MultiSelect: &settings.MultiSelectValue{
				Items: make([]*settings.ItemValue, len(value.GetMultiSelect().Items)),
			},
		}

		for i, item := range value.GetMultiSelect().Items {
			v.GetMultiSelect().Items[i] = &settings.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case models.ConfigType_SINGLE_SELECT:
		v.Value = &settings.Value_SingleSelect{
			SingleSelect: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID:               value.GetSingleSelect().Item.ID,
					FreeTextResponse: value.GetSingleSelect().Item.FreeTextResponse,
				},
			},
		}
	}

	return v
}

func transformValueToModel(value *settings.Value) *models.Value {
	v := &models.Value{
		Key: &models.ConfigKey{
			Key:    value.Key.Key,
			Subkey: value.Key.Subkey,
		},
	}

	switch value.Type {
	case settings.ConfigType_BOOLEAN:
		if value.GetBoolean() != nil {
			v.Value = &models.Value_Boolean{
				Boolean: &models.BooleanValue{
					Value: value.GetBoolean().Value,
				},
			}
		}
	case settings.ConfigType_INTEGER:
		if value.GetInteger() != nil {
			v.Value = &models.Value_Integer{
				Integer: &models.IntegerValue{
					Value: value.GetInteger().Value,
				},
			}
		}
	case settings.ConfigType_TEXT:
		if w := value.GetText(); w != nil {
			v.Value = &models.Value_Text{
				Text: &models.TextValue{
					Value: w.Value,
				},
			}
		}
	case settings.ConfigType_MULTI_SELECT:
		v.Value = &models.Value_MultiSelect{
			MultiSelect: &models.MultiSelectValue{
				Items: make([]*models.ItemValue, len(value.GetMultiSelect().Items)),
			},
		}

		for i, item := range value.GetMultiSelect().GetItems() {
			v.GetMultiSelect().Items[i] = &models.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case settings.ConfigType_SINGLE_SELECT:
		if item := value.GetSingleSelect().GetItem(); item != nil {
			v.Value = &models.Value_SingleSelect{
				SingleSelect: &models.SingleSelectValue{
					Item: &models.ItemValue{
						ID:               item.ID,
						FreeTextResponse: item.FreeTextResponse,
					},
				},
			}
		}
	case settings.ConfigType_STRING_LIST:
		if value.GetStringList() != nil {
			v.Value = &models.Value_StringList{
				StringList: &models.StringListValue{
					Values: value.GetStringList().Values,
				},
			}
		}
	}

	return v
}

func transformKeyToModel(key *settings.ConfigKey) *models.ConfigKey {
	return &models.ConfigKey{
		Key:    key.Key,
		Subkey: key.Subkey,
	}
}

func transformModelToKey(key *models.ConfigKey) *settings.ConfigKey {
	return &settings.ConfigKey{
		Key:    key.Key,
		Subkey: key.Subkey,
	}
}
