package server

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type server struct {
	dal dal.DAL
}

func New(dal dal.DAL) settings.SettingsServer {
	return &server{
		dal: dal,
	}
}

func (s *server) RegisterConfigs(ctx context.Context, in *settings.RegisterConfigsRequest) (*settings.RegisterConfigsResponse, error) {
	configs := make([]*models.Config, len(in.Configs))
	for i, cfg := range in.Configs {
		var err error
		configs[i], err = transformConfigToModel(cfg)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	// TODO: Validate incoming config
	if err := s.dal.SetConfigs(configs); err != nil {
		return nil, errors.Trace(err)
	}
	return &settings.RegisterConfigsResponse{}, nil
}

func (s *server) SetValue(ctx context.Context, in *settings.SetValueRequest) (*settings.SetValueResponse, error) {
	if in.Value.Key == nil || in.Value.Key.Key == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "config key not specified")
	}
	if in.NodeID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "NodeID required")
	}

	// pull up config
	config, err := s.getConfig(in.Value.Key.Key)
	if err != nil {
		return nil, errors.Trace(err)
	} else if config == nil {
		return nil, grpc.Errorf(codes.NotFound, "config with key %s not found", in.Value.Key.Key)
	}

	// validate value against config
	transformedValue, err := validateValueAgainstConfig(in.Value, config)
	if err != nil {
		if grpc.Code(err) == settings.InvalidUserValue {
			return nil, err
		}
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	// TODO: Verify entityID is valid?
	// ensure that entityID is populated
	if err := s.dal.SetValues(in.NodeID, []*models.Value{transformedValue}); err != nil {
		return nil, errors.Trace(err)
	}

	return &settings.SetValueResponse{}, nil
}

func (s *server) GetValues(ctx context.Context, in *settings.GetValuesRequest) (*settings.GetValuesResponse, error) {
	if in.NodeID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "NodeID required")
	}

	keys := make([]*models.ConfigKey, len(in.Keys))
	for i, k := range in.Keys {
		keys[i] = transformKeyToModel(k)
	}

	values, err := s.dal.GetValues(in.NodeID, keys)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// create a value map to quickly check if data is present
	valueMap := make(map[string]*models.Value, len(values))
	for _, v := range values {
		valueMap[v.Key.String()] = v
	}

	// check if value is present, if not, assign default
	transformedValues := make([]*settings.Value, 0, len(in.Keys))
	for _, k := range in.Keys {
		if valueMap[k.String()] != nil {
			transformedValues = append(transformedValues, transformModelToValue(valueMap[k.String()]))
			continue
		}

		// lookup config
		config, err := s.getConfig(k.Key)
		if err != nil {
			return nil, errors.Trace(err)
		} else if config == nil {
			return nil, grpc.Errorf(codes.NotFound, "config with key %s not found", k.Key)
		}

		// assign default
		val := &models.Value{
			Config: config,
			Key: &models.ConfigKey{
				Key:    k.Key,
				Subkey: k.Subkey,
			},
		}
		switch config.Type {
		case models.ConfigType_BOOLEAN:
			if def := config.GetBoolean().Default; def != nil {
				val.Value = &models.Value_Boolean{
					Boolean: def,
				}
			}
		case models.ConfigType_SINGLE_SELECT:
			if def := config.GetSingleSelect().Default; def != nil {
				val.Value = &models.Value_SingleSelect{
					SingleSelect: def,
				}
			}
		case models.ConfigType_MULTI_SELECT:
			if def := config.GetMultiSelect().Default; def != nil {
				val.Value = &models.Value_MultiSelect{
					MultiSelect: def,
				}
			}
		case models.ConfigType_STRING_LIST:
			if def := config.GetStringList().Default; def != nil {
				val.Value = &models.Value_StringList{
					StringList: def,
				}
			}
		case models.ConfigType_INTEGER:
			if def := config.GetInteger().Default; def != nil {
				val.Value = &models.Value_Integer{
					Integer: def,
				}
			}
		case models.ConfigType_TEXT:
			if def := config.GetText().Default; def != nil {
				val.Value = &models.Value_Text{
					Text: def,
				}
			}
		default:
			return nil, grpc.Errorf(codes.Unimplemented, "config type %s not supported", config.Type)
		}

		transformedValues = append(transformedValues, transformModelToValue(val))
	}

	return &settings.GetValuesResponse{
		Values: transformedValues,
	}, nil
}

func (s *server) GetNodeValues(ctx context.Context, in *settings.GetNodeValuesRequest) (*settings.GetNodeValuesResponse, error) {
	configs, err := s.dal.GetAllConfigs()
	if err != nil {
		return nil, errors.Trace(err)
	}

	values, err := s.dal.GetNodeValues(in.NodeID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// create a value map to quickly check if data is present
	valueMap := make(map[string]*models.Value, len(values))
	for _, v := range values {
		valueMap[v.Key.Key+v.Key.Subkey] = v
	}

	// TODO: Filter these settings by possible owner types (ORGANIZATION,INTERNAL,etc)
	// Build out all the possible settings for the node with the total config set
	transformedValues := make([]*settings.Value, 0, len(configs))
	for _, c := range configs {
		if v, ok := valueMap[c.Key]; ok {
			transformedValues = append(transformedValues, transformModelToValue(v))
			delete(valueMap, c.Key)
			continue
		}

		// assign default
		val := &models.Value{
			Config: c,
			Key: &models.ConfigKey{
				Key: c.Key,
			},
		}
		switch c.Type {
		case models.ConfigType_BOOLEAN:
			if def := c.GetBoolean().Default; def != nil {
				val.Value = &models.Value_Boolean{
					Boolean: def,
				}
			}
		case models.ConfigType_SINGLE_SELECT:
			if def := c.GetSingleSelect().Default; def != nil {
				val.Value = &models.Value_SingleSelect{
					SingleSelect: def,
				}
			}
		case models.ConfigType_MULTI_SELECT:
			if def := c.GetMultiSelect().Default; def != nil {
				val.Value = &models.Value_MultiSelect{
					MultiSelect: def,
				}
			}
		case models.ConfigType_STRING_LIST:
			if def := c.GetStringList().Default; def != nil {
				val.Value = &models.Value_StringList{
					StringList: def,
				}
			}
		case models.ConfigType_INTEGER:
			if def := c.GetInteger().Default; def != nil {
				val.Value = &models.Value_Integer{
					Integer: def,
				}
			}
		case models.ConfigType_TEXT:
			if def := c.GetText().Default; def != nil {
				val.Value = &models.Value_Text{
					Text: def,
				}
			}
		default:
			return nil, grpc.Errorf(codes.Unimplemented, "config type %s not supported", c.Type)
		}

		transformedValues = append(transformedValues, transformModelToValue(val))
	}

	for _, v := range valueMap {
		transformedValues = append(transformedValues, transformModelToValue(v))
	}

	return &settings.GetNodeValuesResponse{
		Values: transformedValues,
	}, nil
}

func (s *server) GetConfigs(ctx context.Context, in *settings.GetConfigsRequest) (*settings.GetConfigsResponse, error) {
	configs, err := s.dal.GetConfigs(in.Keys)
	if err != nil {
		return nil, errors.Trace(err)
	}

	transformedConfigs := make([]*settings.Config, len(configs))
	for i, c := range configs {
		var err error
		transformedConfigs[i], err = transformModelToConfig(c)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return &settings.GetConfigsResponse{
		Configs: transformedConfigs,
	}, nil
}

func (s *server) getConfig(key string) (*models.Config, error) {
	configs, err := s.dal.GetConfigs([]string{key})
	if err != nil {
		return nil, nil
	} else if len(configs) == 0 {
		return nil, nil
	} else if len(configs) > 1 {
		return nil, fmt.Errorf("expected 1 config instead got %d", len(configs))
	}

	return configs[0], nil
}
