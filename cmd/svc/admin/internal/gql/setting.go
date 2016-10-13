package gql

import (
	"strconv"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
)

// newSettingType returns a type object representing an entity contact
func newSettingType() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Setting",
			Fields: graphql.Fields{
				"type":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"key":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"subkey":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"subkeyRequired": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
				"value":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"values":         &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
				"validValues":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
			},
		})
}

func getEntitySettings(ctx context.Context, settingsClient settings.SettingsClient, id string) ([]*models.Setting, error) {
	settings, err := settingsClient.GetNodeValues(ctx, &settings.GetNodeValuesRequest{
		NodeID: id,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformSettingsToModel(ctx, settingsClient, settings.Values)
}

// modifySettingInput
type modifySettingInput struct {
	NodeID         string   `gql:"nodeID"`
	Key            string   `gql:"key"`
	Subkey         string   `gql:"subkey"`
	SubkeyRequired bool     `gql:"subkeyRequired"`
	Value          string   `gql:"value"`
	Values         []string `gql:"values"`
}

var modifySettingInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifySettingInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"nodeID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"key":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"subkey": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"value":  &graphql.InputObjectFieldConfig{Type: graphql.String},
			"values": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.String)},
		},
	},
)

type modifySettingOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var modifySettingOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifySettingPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifySettingOutput)
			return ok
		},
	},
)

func newModifySettingField() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(modifySettingOutputType),
		Args: graphql.FieldConfigArgument{
			common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifySettingInputType)},
		},
		Resolve: modifySettingResolve,
	}
}

func modifySettingResolve(p graphql.ResolveParams) (interface{}, error) {
	var in modifySettingInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	resp, err := client.Settings(p).GetConfigs(p.Context, &settings.GetConfigsRequest{
		Keys: []string{in.Key},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(resp.Configs) != 1 {
		return nil, errors.Errorf("Expected 1 config value to be returned but got %+v", resp.Configs)
	}
	config := resp.Configs[0]
	if in.Subkey != "" && !config.AllowSubkeys {
		return nil, errors.Errorf("Subkey %s is not allowed for config key %s as it does not allow Subkeys", in.Subkey, in.Key)
	}
	setReq := &settings.SetValueRequest{
		NodeID: in.NodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    in.Key,
				Subkey: in.Subkey,
			},
			Type: config.Type,
		},
	}
	switch config.Type {
	case settings.ConfigType_SINGLE_SELECT:
		setReq.Value.Value = &settings.Value_SingleSelect{
			SingleSelect: &settings.SingleSelectValue{
				Item: &settings.ItemValue{ID: in.Value},
			},
		}
	case settings.ConfigType_MULTI_SELECT:
		itemValues := make([]*settings.ItemValue, len(in.Values))
		for i, v := range in.Values {
			itemValues[i] = &settings.ItemValue{ID: v}
		}
		setReq.Value.Value = &settings.Value_MultiSelect{
			MultiSelect: &settings.MultiSelectValue{
				Items: itemValues,
			},
		}
	case settings.ConfigType_STRING_LIST:
		setReq.Value.Value = &settings.Value_StringList{
			StringList: &settings.StringListValue{Values: in.Values},
		}
	case settings.ConfigType_BOOLEAN:
		b, err := strconv.ParseBool(in.Value)
		if err != nil {
			return nil, errors.Trace(err)
		}
		setReq.Value.Value = &settings.Value_Boolean{
			Boolean: &settings.BooleanValue{Value: b},
		}
	case settings.ConfigType_INTEGER:
		i, err := strconv.ParseInt(in.Value, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		setReq.Value.Value = &settings.Value_Integer{
			Integer: &settings.IntegerValue{Value: i},
		}
	default:
		return nil, errors.Errorf("Unsupported config type %s", config.Type)
	}

	golog.ContextLogger(p.Context).Debugf("Modifying Setting - NodeID: %s, Key: %s, Subkey %s, Value: %+v", setReq.NodeID, setReq.Value.Key.Key, setReq.Value.Key.Subkey, setReq.Value.Value)
	if _, err := client.Settings(p).SetValue(p.Context, setReq); err != nil {
		return nil, errors.Trace(err)
	}

	return &modifySettingOutput{
		Success: true,
	}, nil
}
