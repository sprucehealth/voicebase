package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
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
			_, ok := value.(*models.StringListSetting)
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
			_, ok := value.(*models.BooleanSetting)
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
			_, ok := value.(*models.SelectableItem)
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
			_, ok := value.(*models.SelectSetting)
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
			_, ok := value.(*models.StringListSettingValue)
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
			_, ok := value.(*models.BooleanSettingValue)
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
			_, ok := value.(*models.SelectableItemValue)
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
			_, ok := value.(*models.SelectableSettingValue)
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
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		key, _ := p.Args["key"].(string)
		subkey, _ := p.Args["subkey"].(string)
		nodeID, _ := p.Args["nodeID"].(string)

		isForwardingList := key == excommsSettings.ConfigKeyForwardingList
		if isForwardingList {
			pn, err := phone.Format(subkey, phone.E164)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			subkey = pn
		}

		// ensure that the send all calls to voicemail setting is being queried for at the org level
		// and not the entity level
		// TODO: Remove this workaround once we feel confident that there are no < v1.2 clients on iOS and < v1.1 on android
		// out in the wild.
		if key == excommsSettings.ConfigKeySendCallsToVoicemail {
			entity, err := ram.Entity(ctx, nodeID, []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS}, 1)
			if err != nil {
				return nil, fmt.Errorf("Unable to get entity %s: %s", nodeID, err.Error())
			}
			if entity.Type == directory.EntityType_INTERNAL {
				for _, membership := range entity.Memberships {
					if membership.Type == directory.EntityType_ORGANIZATION {
						nodeID = membership.ID
						for _, contact := range membership.Contacts {
							if contact.Provisioned && contact.ContactType == directory.ContactType_PHONE {
								subkey = contact.Value
								break
							}
						}
						break
					}
				}
			}
		}

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
			return nil, errors.InternalError(ctx, err)
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
