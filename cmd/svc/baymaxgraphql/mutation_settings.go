package main

import (
	"fmt"

	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
)

type modifySettingOutput struct {
	ClientMutationID string      `json:"clientMutationId,omitempty"`
	Setting          interface{} `json:"setting"`
	UserErrorMessage string      `json:"userErrorMessage"`
	Result           string      `json:"result"`
}

var stringListInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "StringListInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"list": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	},
)

var booleanInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "BooleanInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"set": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

var selectableItemInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SelectableItemInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"id":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"text": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var selectableItemsInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SelectableItemsInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"items": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(selectableItemInputType))},
		},
	},
)

var modifySettingInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifySettingInputType",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"nodeID": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "Specify the id of an account, entity or organization",
			},
			"key": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The setting key for which the value is being set",
			},
			"subkey": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "The setting subkey for which the value is being set",
			},
			"booleanValue": &graphql.InputObjectFieldConfig{
				Type:        booleanInputType,
				Description: "Boolean Setting value",
			},
			"stringListValue": &graphql.InputObjectFieldConfig{
				Type:        stringListInputType,
				Description: "StringList Setting value",
			},
			"multiSelectValue": &graphql.InputObjectFieldConfig{
				Type:        selectableItemsInputType,
				Description: "MultiSelect Setting value",
			},
			"singleSelectValue": &graphql.InputObjectFieldConfig{
				Type:        selectableItemsInputType,
				Description: "SingleSelect Setting value",
			},
		},
	},
)

const (
	modifySettingResultSuccess      = "SUCCESS"
	modifySettingResultInvalidInput = "INVALID_INPUT"
)

var modifySettingResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ModifySettingResult",
		Description: "Result of modifySetting mutation",
		Values: graphql.EnumValueConfigMap{
			modifySettingResultSuccess: &graphql.EnumValueConfig{
				Value:       modifySettingResultSuccess,
				Description: "Success",
			},
			modifySettingResultInvalidInput: &graphql.EnumValueConfig{
				Value:       modifySettingResultInvalidInput,
				Description: "Invalid input",
			},
		},
	},
)

var modifySettingOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifySettingPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(modifySettingResultType)},
			"setting":          &graphql.Field{Type: settingsInterfaceType},
			"userErrorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifySettingOutput)
			return ok
		},
	},
)

var modifySettingMutation = &graphql.Field{
	Type: graphql.NewNonNull(modifySettingOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifySettingInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
		}

		input, _ := p.Args["input"].(map[string]interface{})
		key, _ := input["key"].(string)
		subkey, _ := input["subkey"].(string)
		nodeID, _ := input["nodeID"].(string)
		mutationID, _ := input["clientMutationId"].(string)

		// TODO: Add a validator for the subkey so as to enforce subkey to be of valid format
		isForwardingList := key == excommsSettings.ConfigKeyForwardingList
		if isForwardingList {
			if subkey == "" {
				return nil, fmt.Errorf("subkey expected but got none")
			}
			pn, err := phone.ParseNumber(subkey)
			if err != nil {
				return nil, fmt.Errorf("unable to parse subkey into valid phone number: %s", err.Error())
			}
			subkey = pn.String()
		}

		// pull config to know what value to expect
		var config *settings.Config
		res, err := svc.settings.GetConfigs(ctx, &settings.GetConfigsRequest{
			Keys: []string{key},
		})
		if err != nil {
			return nil, internalError(ctx, err)
		} else if len(res.Configs) != 1 {
			return nil, fmt.Errorf("Expected 1 config but got %d", len(res.Configs))
		}
		config = res.Configs[0]

		var node interface{}
		prefix := nodePrefix(nodeID)
		switch prefix {
		case "account":
			node, err = lookupAccount(ctx, svc, nodeID)
		case "entity":
			node, err = lookupEntity(ctx, svc, nodeID)
		default:
			return nil, fmt.Errorf("nodeID type %s not supported", prefix)
		}

		// enforce that the node is of a type that is one of the possible
		// owners on the config
		validOwner := false
		for _, po := range config.PossibleOwners {
			switch po {
			case settings.OwnerType_ACCOUNT:
				if _, ok := node.(*account); ok {
					validOwner = true
					break
				}
			case settings.OwnerType_INTERNAL_ENTITY:
				if _, ok := node.(*entity); ok {
					validOwner = true
					break
				}
			case settings.OwnerType_ORGANIZATION:
				if _, ok := node.(*organization); ok {
					validOwner = true
					break
				}
			default:
				return nil, fmt.Errorf("owner type %s for config %s not supported", po.String(), config.Key)
			}
		}
		if !validOwner {
			return nil, fmt.Errorf("nodeID %s cannot modify a setting for config %s", nodeID, config.Key)
		}

		// populate value based on config type
		val := &settings.Value{
			Key: &settings.ConfigKey{
				Key:    key,
				Subkey: subkey,
			},
			Type: config.Type,
		}

		switch config.Type {
		case settings.ConfigType_SINGLE_SELECT:
			value, ok := input["singleSelectValue"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Expected a list of selected items to be set for config %s.%s but got none", key, subkey)
			}

			items, _ := value["items"].([]interface{})
			if len(items) != 1 {
				return nil, fmt.Errorf("Expected 1 item for a single select for config %s.%s but got %d", key, subkey, len(items))
			}
			id, _ := items[0].(map[string]interface{})["id"].(string)
			text, _ := items[0].(map[string]interface{})["text"].(string)
			val.Value = &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               id,
						FreeTextResponse: text,
					},
				},
			}
		case settings.ConfigType_MULTI_SELECT:
			value, ok := input["multiSelectValue"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Expected a list of selected items to be set for config %s.%s but got none", key, subkey)
			}

			items, _ := value["items"].([]interface{})
			if len(items) == 0 {
				return nil, fmt.Errorf("Expected at least 1 item for a multi select item for config %s.%s but got %d", key, subkey, len(items))
			}

			val.Value = &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: make([]*settings.ItemValue, len(items)),
				},
			}

			for i, item := range items {
				itemMap := item.(map[string]interface{})
				id, _ := itemMap["id"].(string)
				text, _ := itemMap["text"].(string)
				val.GetMultiSelect().Items[i] = &settings.ItemValue{
					ID:               id,
					FreeTextResponse: text,
				}
			}
		case settings.ConfigType_STRING_LIST:
			value, ok := input["stringListValue"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Expected a list of strings to be set for config %s.%s but got none", key, subkey)
			}

			list, _ := value["list"].([]interface{})

			val.Value = &settings.Value_StringList{
				StringList: &settings.StringListValue{
					Values: make([]string, len(list)),
				},
			}
			for i, sItem := range list {
				str, ok := sItem.(string)
				if !ok {
					return nil, fmt.Errorf("Expected string in array but got %T for config %s.%s", sItem, key, subkey)
				}

				// TODO: Add validator to the string list to enforce each item in the list to be of a particular type
				if isForwardingList {
					pn, err := phone.Format(str, phone.Pretty)
					if err != nil {
						return &modifySettingOutput{
							ClientMutationID: mutationID,
							UserErrorMessage: "Please enter a valid US phone number",
							Result:           modifySettingResultInvalidInput,
						}, nil
					}

					val.GetStringList().Values[i] = pn
				} else {
					val.GetStringList().Values[i] = str
				}
			}

		case settings.ConfigType_BOOLEAN:
			value, ok := input["booleanValue"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Expected a bool value to be set for config %s.%s instead got none", key, subkey)
			}

			boolVal, _ := value["set"].(bool)
			val.Value = &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{
					Value: boolVal,
				},
			}
		default:
			return nil, fmt.Errorf("Unsupported type %s for config %s.%s", config.Type.String(), key, subkey)
		}

		_, err = svc.settings.SetValue(ctx, &settings.SetValueRequest{
			NodeID: nodeID,
			Value:  val,
		})
		if err != nil {
			if grpc.Code(err) == settings.InvalidUserValue {
				return &modifySettingOutput{
					ClientMutationID: mutationID,
					UserErrorMessage: grpc.ErrorDesc(err),
					Result:           modifySettingResultInvalidInput,
				}, nil
			}
			return nil, internalError(ctx, err)
		}

		var setting interface{}
		switch config.Type {
		case settings.ConfigType_BOOLEAN:
			setting = transformBooleanSettingToResponse(config, val)
		case settings.ConfigType_MULTI_SELECT, settings.ConfigType_SINGLE_SELECT:
			setting = transformMultiSelectToResponse(config, val)
		case settings.ConfigType_STRING_LIST:
			setting = transformStringListSettingToResponse(config, val)
		default:
			return nil, fmt.Errorf("Unsupported type %s", config.Type.String())

		}

		return &modifySettingOutput{
			ClientMutationID: mutationID,
			Setting:          setting,
			Result:           modifySettingResultSuccess,
		}, nil
	},
}
