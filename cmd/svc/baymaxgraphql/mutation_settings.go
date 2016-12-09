package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type modifySettingOutput struct {
	ClientMutationID string      `json:"clientMutationId,omitempty"`
	Success          bool        `json:"success"`
	ErrorCode        string      `json:"errorCode,omitempty"`
	ErrorMessage     string      `json:"errorMessage,omitempty"`
	Setting          interface{} `json:"setting"`
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

var selectableItemArrayInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SelectableItemArrayInput",
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
			"selectValue": &graphql.InputObjectFieldConfig{
				Type:        selectableItemArrayInputType,
				Description: "SelectSetting value",
			},
		},
	},
)

const (
	modifySettingErrorCodeInvalidInput = "INVALID_INPUT"
)

var modifySettingErrorCodeEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ModifySettingErrorCode",
		Description: "Result of modifySetting mutation",
		Values: graphql.EnumValueConfigMap{
			modifySettingErrorCodeInvalidInput: &graphql.EnumValueConfig{
				Value:       modifySettingErrorCodeInvalidInput,
				Description: "Invalid input",
			},
		},
	},
)

var modifySettingOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifySettingPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: modifySettingErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"setting":          &graphql.Field{Type: settingsInterfaceType},
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
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input, _ := p.Args["input"].(map[string]interface{})
		key := input["key"].(string)
		subkey, _ := input["subkey"].(string)
		nodeID := input["nodeID"].(string)
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
			return nil, errors.InternalError(ctx, err)
		} else if len(res.Configs) != 1 {
			return nil, fmt.Errorf("Expected 1 config but got %d", len(res.Configs))
		}
		config = res.Configs[0]

		var node interface{}
		switch nodeIDPrefix(nodeID) {
		case "account":
			node, err = lookupAccount(ctx, ram, nodeID)
		case "entity":
			node, err = lookupEntity(ctx, svc, ram, nodeID)
		default:
			return nil, fmt.Errorf("type of node ID '%s' unknown", nodeID)
		}
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		// enforce that the node is of a type that is one of the possible
		// owners on the config
		validOwner := false
		for _, po := range config.PossibleOwners {
			switch po {
			case settings.OwnerType_ACCOUNT:
				if _, ok := node.(*models.ProviderAccount); ok {
					validOwner = true
					break
				}
			case settings.OwnerType_INTERNAL_ENTITY:
				if _, ok := node.(*models.Entity); ok {
					validOwner = true
					break
				}
			case settings.OwnerType_ORGANIZATION:
				if _, ok := node.(*models.Organization); ok {
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
			value, ok := input["selectValue"].(map[string]interface{})
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
			value, ok := input["selectValue"].(map[string]interface{})
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
							Success:          false,
							ErrorCode:        modifySettingErrorCodeInvalidInput,
							ErrorMessage:     "Please enter a valid US phone number",
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

			// if the send all calls to voicemail is being set at the entity level, set it at the org level instead
			// TODO: Remove this workaround once we feel confident that there are no < v1.2 clients on iOS and < v1.1 on android
			// out in the wild.
			if key == excommsSettings.ConfigKeySendCallsToVoicemail {
				if _, ok := node.(*models.Entity); ok {
					entity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						Key: &directory.LookupEntitiesRequest_EntityID{
							EntityID: nodeID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             1,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
						},
						Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
						ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
					})
					if err != nil {
						return nil, fmt.Errorf("Unable to lookup entity for %s: %s", nodeID, err.Error())
					}

					for _, membership := range entity.Memberships {
						if membership.Type == directory.EntityType_ORGANIZATION {
							nodeID = membership.ID
							for _, contact := range membership.Contacts {
								if contact.Provisioned && contact.ContactType == directory.ContactType_PHONE {
									val.Key.Subkey = contact.Value
									break
								}
							}
							break
						}
					}
				}
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
			if grpc.Code(err) == settings.InvalidUserValue || grpc.Code(err) == codes.InvalidArgument {
				return &modifySettingOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        modifySettingErrorCodeInvalidInput,
					ErrorMessage:     grpc.ErrorDesc(err),
				}, nil
			}
			return nil, errors.InternalError(ctx, err)
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

		if err := handleSavedQueryBackwards(ctx, ram, nodeID, val); err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("Error while setting old config for %s", val.Key.Key))
		}

		return &modifySettingOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Setting:          setting,
		}, nil
	},
}

func handleSavedQueryBackwards(ctx context.Context, ram raccess.ResourceAccessor, nodeID string, value *settings.Value) error {
	switch value.Key.Key {
	case notification.PatientNotificationPreferencesSettingsKey, notification.TeamNotificationPreferencesSettingsKey:
		// If it's something we care about then get the saved queries for the entity
		sqs, err := ram.SavedQueries(ctx, nodeID)
		if err != nil {
			return errors.Trace(err)
		}
		settingValue := value.GetSingleSelect().Item.ID
		switch value.Key.Key {
		case notification.PatientNotificationPreferencesSettingsKey:
			patientSQ := savedQueryFromList(ctx, sqs, "Patient")
			if patientSQ == nil {
				golog.Errorf("Unable to find patient saved query for nodeID %s - aborting backwards compatibility change for notifications", nodeID)
				return nil
			}
			switch settingValue {
			case notification.ThreadActivityNotificationPreferenceAllMessages:
				if _, err := ram.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
					SavedQueryID:         patientSQ.ID,
					NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE,
				}); err != nil {
					return errors.Trace(err)
				}
			case notification.ThreadActivityNotificationPreferenceReferencedOnly, notification.ThreadActivityNotificationPreferenceOff:
				if _, err := ram.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
					SavedQueryID:         patientSQ.ID,
					NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE,
				}); err != nil {
					return errors.Trace(err)
				}
			}
		case notification.TeamNotificationPreferencesSettingsKey:
			teamSQ := savedQueryFromList(ctx, sqs, "Team")
			if teamSQ == nil {
				golog.Errorf("Unable to find team saved query for nodeID %s - aborting backwards compatibility change for notifications", nodeID)
				return nil
			}
			switch settingValue {
			case notification.ThreadActivityNotificationPreferenceAllMessages:
				if _, err := ram.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
					SavedQueryID:         teamSQ.ID,
					NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE,
				}); err != nil {
					return errors.Trace(err)
				}
			case notification.ThreadActivityNotificationPreferenceReferencedOnly, notification.ThreadActivityNotificationPreferenceOff:
				if _, err := ram.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
					SavedQueryID:         teamSQ.ID,
					NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE,
				}); err != nil {
					return errors.Trace(err)
				}
			}
		}
	}
	return nil
}

func savedQueryFromList(ctx context.Context, savedQueries []*threading.SavedQuery, title string) *threading.SavedQuery {
	for _, sq := range savedQueries {
		if strings.EqualFold(sq.ShortTitle, title) {
			return sq
		}
	}
	return nil
}
