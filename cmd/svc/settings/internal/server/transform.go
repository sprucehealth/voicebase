package server

import (
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/svc/settings"
)

func transformConfigToModel(config *settings.Config) *models.Config {
	m := &models.Config{
		Title:          config.Title,
		Description:    config.Description,
		Key:            config.Key,
		AllowSubkeys:   config.AllowSubkeys,
		Type:           models.ConfigType(models.ConfigType_value[config.Type.String()]),
		PossibleOwners: make([]models.OwnerType, len(config.PossibleOwners)),
	}

	for i, po := range config.PossibleOwners {
		m.PossibleOwners[i] = models.OwnerType(models.OwnerType_value[po.String()])
	}

	switch m.Type {
	case models.ConfigType_BOOLEAN:
		m.Config = &models.Config_Boolean{
			Boolean: &models.BooleanConfig{
				Default: &models.BooleanValue{
					Value: config.GetBoolean().GetDefault().Value,
				},
			},
		}
	case models.ConfigType_MULTI_SELECT:
		m.Config = &models.Config_MultiSelect{
			MultiSelect: &models.MultiSelectConfig{
				Items: make([]*models.Item, len(config.GetMultiSelect().Items)),
				Default: &models.MultiSelectValue{
					Items: make([]*models.ItemValue, len(config.GetMultiSelect().Default.Items)),
				},
			},
		}

		for i, item := range config.GetMultiSelect().Items {
			m.GetMultiSelect().Items[i] = &models.Item{
				ID:            item.ID,
				Label:         item.Label,
				AllowFreeText: item.AllowFreeText,
			}
		}

		for i, item := range config.GetMultiSelect().Default.GetItems() {
			m.GetMultiSelect().Default.Items[i] = &models.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case models.ConfigType_SINGLE_SELECT:
		m.Config = &models.Config_SingleSelect{
			SingleSelect: &models.SingleSelectConfig{
				Items: make([]*models.Item, len(config.GetSingleSelect().Items)),
				Default: &models.SingleSelectValue{
					Item: &models.ItemValue{
						ID:               config.GetSingleSelect().GetDefault().Item.ID,
						FreeTextResponse: config.GetSingleSelect().GetDefault().Item.FreeTextResponse,
					},
				},
			},
		}
		for i, item := range config.GetSingleSelect().Items {
			m.GetSingleSelect().Items[i] = &models.Item{
				ID:            item.ID,
				Label:         item.Label,
				AllowFreeText: item.AllowFreeText,
			}
		}
	case models.ConfigType_STRING_LIST:
		m.Config = &models.Config_StringList{
			StringList: &models.StringListConfig{},
		}
		if config.GetStringList().GetDefault() != nil {
			m.GetStringList().Default = &models.StringListValue{
				Values: config.GetStringList().GetDefault().Values,
			}
		}
	}

	return m
}

func transformModelToConfig(config *models.Config) *settings.Config {
	c := &settings.Config{
		Title:          config.Title,
		Description:    config.Description,
		Key:            config.Key,
		AllowSubkeys:   config.AllowSubkeys,
		Type:           settings.ConfigType(settings.ConfigType_value[config.Type.String()]),
		PossibleOwners: make([]settings.OwnerType, len(config.PossibleOwners)),
	}

	for i, po := range config.PossibleOwners {
		c.PossibleOwners[i] = settings.OwnerType(settings.OwnerType_value[po.String()])
	}

	switch config.Type {
	case models.ConfigType_BOOLEAN:
		c.Config = &settings.Config_Boolean{
			Boolean: &settings.BooleanConfig{
				Default: &settings.BooleanValue{
					Value: config.GetBoolean().Default.Value,
				},
			},
		}
	case models.ConfigType_MULTI_SELECT:
		c.Config = &settings.Config_MultiSelect{
			MultiSelect: &settings.MultiSelectConfig{
				Items: make([]*settings.Item, len(config.GetMultiSelect().Items)),
				Default: &settings.MultiSelectValue{
					Items: make([]*settings.ItemValue, len(config.GetMultiSelect().Items)),
				},
			},
		}
		for i, item := range config.GetMultiSelect().Items {
			c.GetMultiSelect().Items[i] = &settings.Item{
				ID:            item.ID,
				AllowFreeText: item.AllowFreeText,
				Label:         item.Label,
			}
		}
		for i, item := range config.GetMultiSelect().Default.Items {
			c.GetMultiSelect().Default.Items[i] = &settings.ItemValue{
				ID:               item.ID,
				FreeTextResponse: item.FreeTextResponse,
			}
		}
	case models.ConfigType_SINGLE_SELECT:
		c.Config = &settings.Config_SingleSelect{
			SingleSelect: &settings.SingleSelectConfig{
				Items: make([]*settings.Item, len(config.GetSingleSelect().Items)),
				Default: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               config.GetSingleSelect().Default.Item.ID,
						FreeTextResponse: config.GetSingleSelect().Default.Item.FreeTextResponse,
					},
				},
			},
		}
		for i, item := range config.GetSingleSelect().Items {
			c.GetSingleSelect().Items[i] = &settings.Item{
				ID:            item.ID,
				AllowFreeText: item.AllowFreeText,
				Label:         item.Label,
			}
		}
	case models.ConfigType_STRING_LIST:
	}

	return c
}

func transformModelToValue(value *models.Value) *settings.Value {
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
	case models.ConfigType_STRING_LIST:
		v.Value = &settings.Value_StringList{
			StringList: &settings.StringListValue{},
		}
		if value.GetStringList() != nil {
			v.GetStringList().Values = value.GetStringList().Values
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
		if value.GetSingleSelect().GetItem() != nil {
			v.Value = &models.Value_SingleSelect{
				SingleSelect: &models.SingleSelectValue{
					Item: &models.ItemValue{
						ID:               value.GetSingleSelect().GetItem().ID,
						FreeTextResponse: value.GetSingleSelect().GetItem().FreeTextResponse,
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
