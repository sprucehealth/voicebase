package models

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/settings"
)

type valueAndConfig struct {
	Value  *settings.Value
	Config *settings.Config
}

// Setting represents the values contained in the settings service
type Setting struct {
	Type           string   `json:"type"`
	Key            string   `json:"key"`
	Subkey         string   `json:"subkey"`
	SubkeyRequired bool     `json:"subkeyRequired"`
	Value          string   `json:"value"`
	Values         []string `json:"values"`
	ValidValues    []string `json:"validValues"`
}

// TransformSettingsToModel transforms the internal setting into something understood by graphql
func TransformSettingsToModel(ctx context.Context, settingsClient settings.SettingsClient, vs []*settings.Value) ([]*Setting, error) {
	mss := make([]*Setting, len(vs))
	vcs, err := getValuesAndConfigs(ctx, settingsClient, vs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for i, vc := range vcs {
		mss[i], err = transformSettingToModel(ctx, vc)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return mss, nil
}

// TODO: Should likely get these in bulk for the list
func getValuesAndConfigs(ctx context.Context, settingsClient settings.SettingsClient, vs []*settings.Value) ([]*valueAndConfig, error) {
	keys := make([]string, len(vs))
	for i, v := range vs {
		keys[i] = v.Key.Key
	}
	resp, err := settingsClient.GetConfigs(ctx, &settings.GetConfigsRequest{
		Keys: keys,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	configMap := make(map[string]*settings.Config)
	for _, c := range resp.Configs {
		configMap[c.Key] = c
	}
	valueAndConfigs := make([]*valueAndConfig, len(vs))
	for i, v := range vs {
		valueAndConfigs[i] = &valueAndConfig{
			Value:  v,
			Config: configMap[v.Key.Key],
		}
	}
	return valueAndConfigs, nil
}

// TransformSettingToModel transforms the internal setting into something understood by graphql
func transformSettingToModel(ctx context.Context, vc *valueAndConfig) (*Setting, error) {
	ms := &Setting{
		Type:           vc.Value.Type.String(),
		Key:            vc.Value.Key.Key,
		Subkey:         vc.Value.Key.Subkey,
		SubkeyRequired: vc.Config.AllowSubkeys,
	}
	var err error
	switch vc.Value.Type {
	case settings.ConfigType_BOOLEAN:
		ms.Value = fmt.Sprintf("%v", vc.Value.GetBoolean().Value)
	case settings.ConfigType_SINGLE_SELECT:
		ms.Value = vc.Value.GetSingleSelect().Item.FreeTextResponse
		ms.ValidValues, err = getConfigValidValues(ctx, vc.Config)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case settings.ConfigType_MULTI_SELECT:
		values := make([]string, len(vc.Value.GetMultiSelect().Items))
		for i, si := range vc.Value.GetMultiSelect().Items {
			values[i] = si.FreeTextResponse
		}
		ms.Values = values
		ms.ValidValues, err = getConfigValidValues(ctx, vc.Config)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case settings.ConfigType_STRING_LIST:
		values := make([]string, len(vc.Value.GetStringList().Values))
		for i, si := range vc.Value.GetStringList().Values {
			values[i] = si
		}
		ms.Values = values
	case settings.ConfigType_INTEGER:
		ms.Value = fmt.Sprintf("%d", vc.Value.GetInteger().Value)
	default:
		golog.Errorf("Unknown setting type %s", vc.Value.Type)
	}
	return ms, nil
}

func getConfigValidValues(ctx context.Context, config *settings.Config) ([]string, error) {
	var validItems []*settings.Item
	switch config.Type {
	case settings.ConfigType_SINGLE_SELECT:
		validItems = config.GetSingleSelect().Items
	case settings.ConfigType_MULTI_SELECT:
		validItems = config.GetMultiSelect().Items
	}

	var validValues []string
	for _, i := range validItems {
		validValues = append(validValues, i.ID)
	}
	return validValues, nil
}
