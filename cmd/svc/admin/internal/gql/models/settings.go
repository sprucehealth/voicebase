package models

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/settings"
)

// Setting represents the values contained in the settings service
type Setting struct {
	Type   string `json:"type"`
	Key    string `json:"key"`
	Subkey string `json:"subkey"`
	// TODO: Perhaps move to a more granular value representation. For now just strings
	Value string `json:"value"`
}

// TransformSettingsToModel transforms the internal setting into something understood by graphql
func TransformSettingsToModel(ss []*settings.Value) []*Setting {
	mss := make([]*Setting, len(ss))
	for i, s := range ss {
		mss[i] = TransformSettingToModel(s)
	}
	return mss
}

// TransformSettingToModel transforms the internal setting into something understood by graphql
func TransformSettingToModel(s *settings.Value) *Setting {
	ms := &Setting{
		Type:   s.Type.String(),
		Key:    s.Key.Key,
		Subkey: s.Key.Subkey,
	}
	switch s.Type {
	case settings.ConfigType_BOOLEAN:
		ms.Value = fmt.Sprintf("%v", s.GetBoolean().Value)
	case settings.ConfigType_SINGLE_SELECT:
		ms.Value = s.GetSingleSelect().Item.FreeTextResponse
	case settings.ConfigType_MULTI_SELECT:
		values := make([]string, len(s.GetMultiSelect().Items))
		for i, si := range s.GetMultiSelect().Items {
			values[i] = si.FreeTextResponse
		}
		ms.Value = strings.Join(values, ", ")
	case settings.ConfigType_STRING_LIST:
		values := make([]string, len(s.GetStringList().Values))
		for i, si := range s.GetStringList().Values {
			values[i] = si
		}
		ms.Value = strings.Join(values, ", ")
	case settings.ConfigType_INTEGER:
		ms.Value = fmt.Sprintf("%d", s.GetInteger().Value)
	default:
		golog.Errorf("Unknown setting type %s", s.Type)
	}
	return ms
}
