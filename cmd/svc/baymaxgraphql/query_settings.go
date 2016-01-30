package main

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/svc/settings"
)

var settingsInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "Setting",
		Fields: graphql.Fields{
			"key":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":      &graphql.Field{Type: graphql.String},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"value":       &graphql.Field{Type: graphql.NewNonNull(settingValueInterfaceType)},
		},
	})

var stringListType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "StringListSetting",
		Interfaces: []*graphql.Interface{
			settingsInterfaceType,
		},
		Fields: graphql.Fields{
			"key":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":      &graphql.Field{Type: graphql.String},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"value":       &graphql.Field{Type: graphql.NewNonNull(settingValueInterfaceType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*stringListSetting)
			return ok
		},
	},
)

var booleanSettingType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "BooleanSetting",
		Interfaces: []*graphql.Interface{
			settingsInterfaceType,
		},
		Fields: graphql.Fields{
			"key":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":      &graphql.Field{Type: graphql.String},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"value":       &graphql.Field{Type: graphql.NewNonNull(settingValueInterfaceType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*booleanSetting)
			return ok
		},
	},
)

var selectableItemType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SelectableItem",
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"label":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"allowFreeText": &graphql.Field{Type: graphql.Boolean},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*selectableItem)
			return ok
		},
	},
)

var selectSettingType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SelectSetting",
		Interfaces: []*graphql.Interface{
			settingsInterfaceType,
		},
		Fields: graphql.Fields{
			"key":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":      &graphql.Field{Type: graphql.String},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"options":     &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(selectableItemType))},
			"value":       &graphql.Field{Type: graphql.NewNonNull(settingValueInterfaceType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*selectSetting)
			return ok
		},
	},
)

var settingValueInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "SettingValue",
		Fields: graphql.Fields{
			"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey": &graphql.Field{Type: graphql.String},
		},
	})

var stringListSettingValueType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "StringListSettingValue",
		Interfaces: []*graphql.Interface{
			settingValueInterfaceType,
		},
		Fields: graphql.Fields{
			"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey": &graphql.Field{Type: graphql.String},
			"list":   &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*stringListSettingValue)
			return ok
		},
	},
)

var booleanSettingValueType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "BooleanSettingValue",
		Interfaces: []*graphql.Interface{
			settingValueInterfaceType,
		},
		Fields: graphql.Fields{
			"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey": &graphql.Field{Type: graphql.String},
			"set":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*booleanSettingValue)
			return ok
		},
	},
)

var selectableItemValueType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SelectableItemValue",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"text": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*selectableItemValue)
			return ok
		},
	},
)

var selectableSettingValueType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SelectableSettingValue",
		Interfaces: []*graphql.Interface{
			settingValueInterfaceType,
		},
		Fields: graphql.Fields{
			"key":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey": &graphql.Field{Type: graphql.String},
			"items":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(selectableItemValueType))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*selectableSettingValue)
			return ok
		},
	},
)

var settingsQuery = &graphql.Field{
	Type: graphql.NewNonNull(settingsInterfaceType),
	Args: graphql.FieldConfigArgument{
		"key":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		"subkey": &graphql.ArgumentConfig{Type: graphql.String},
		"nodeID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated
		}

		key, _ := p.Args["key"].(string)
		subkey, _ := p.Args["subkey"].(string)
		nodeID, _ := p.Args["nodeID"].(string)

		par := conc.NewParallel()

		// get config
		var config *settings.Config
		par.Go(func() error {
			res, err := svc.settings.GetConfigs(ctx, &settings.GetConfigsRequest{
				Keys: []string{key},
			})
			if err != nil {
				return err
			} else if len(res.Configs) != 1 {
				return fmt.Errorf("Expected 1 config but got %d", len(res.Configs))
			}
			config = res.Configs[0]
			return nil
		})

		// get value
		var value *settings.Value
		par.Go(func() error {
			res, err := svc.settings.GetValues(ctx, &settings.GetValuesRequest{
				Keys: []*settings.ConfigKey{
					{
						Key:    key,
						Subkey: subkey,
					},
				},
				NodeID: nodeID,
			})
			if err != nil {
				return err
			} else if len(res.Values) != 1 {
				return fmt.Errorf("Expected 1 value but got %d", len(res.Values))
			}

			value = res.Values[0]
			return nil
		})

		if err := par.Wait(); err != nil {
			return nil, internalError(err)
		}

		var setting interface{}
		switch config.Type {
		case settings.ConfigType_BOOLEAN:
			setting = transformBooleanSettingToResponse(config, value)
		case settings.ConfigType_MULTI_SELECT, settings.ConfigType_SINGLE_SELECT:
			setting = transformMultiSelectToResponse(config, value)
		case settings.ConfigType_STRING_LIST:
			setting = transformStringListSettingToResponse(config, value)
		default:
			return nil, fmt.Errorf("Unsupported type %s", config.Type)
		}

		return setting, nil
	},
}
