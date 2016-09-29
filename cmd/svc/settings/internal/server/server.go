package server

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

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
		configs[i] = transformConfigToModel(cfg)
	}

	// TODO: Validate incoming config
	if err := s.dal.SetConfigs(configs); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &settings.RegisterConfigsResponse{}, nil
}

func (s *server) SetValue(ctx context.Context, in *settings.SetValueRequest) (*settings.SetValueResponse, error) {
	if in.Value.Key == nil || in.Value.Key.Key == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "config key not specified")
	}

	// pull up config
	config, err := s.getConfig(in.Value.Key.Key)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if config == nil {
		return nil, grpcErrorf(codes.NotFound, "config with key %s not found", in.Value.Key.Key)
	}

	// validate value against config
	transformedValue, err := validateValueAgainstConfig(in.Value, config)
	if err != nil {
		if grpc.Code(err) == settings.InvalidUserValue {
			return nil, err
		}
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	// TODO: Verify entityID is valid?
	// ensure that entityID is populated
	if err := s.dal.SetValues(in.NodeID, []*models.Value{transformedValue}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &settings.SetValueResponse{}, nil
}

func (s *server) GetValues(ctx context.Context, in *settings.GetValuesRequest) (*settings.GetValuesResponse, error) {
	keys := make([]*models.ConfigKey, len(in.Keys))
	for i, k := range in.Keys {
		keys[i] = transformKeyToModel(k)
	}

	values, err := s.dal.GetValues(in.NodeID, keys)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
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
			return nil, grpcErrorf(codes.Internal, err.Error())
		} else if config == nil {
			return nil, grpcErrorf(codes.NotFound, "config with key %s not found", k.Key)
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
			if config.GetBoolean().Default != nil {
				val.Value = &models.Value_Boolean{
					Boolean: config.GetBoolean().Default,
				}
			}
		case models.ConfigType_SINGLE_SELECT:
			if config.GetSingleSelect().Default != nil {
				val.Value = &models.Value_SingleSelect{
					SingleSelect: config.GetSingleSelect().Default,
				}
			}
		case models.ConfigType_MULTI_SELECT:
			if config.GetMultiSelect().Default != nil {
				val.Value = &models.Value_MultiSelect{
					MultiSelect: config.GetMultiSelect().Default,
				}
			}
		case models.ConfigType_STRING_LIST:
			if config.GetStringList().Default != nil {
				val.Value = &models.Value_StringList{
					StringList: config.GetStringList().Default,
				}
			}
		case models.ConfigType_INTEGER:
			if config.GetInteger().Default != nil {
				val.Value = &models.Value_Integer{
					Integer: config.GetInteger().Default,
				}
			}
		default:
			return nil, grpcErrorf(codes.Unimplemented, "config type %s not supported", config.Type)
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
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	values, err := s.dal.GetNodeValues(in.NodeID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
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
			if c.GetBoolean().Default != nil {
				val.Value = &models.Value_Boolean{
					Boolean: c.GetBoolean().Default,
				}
			}
		case models.ConfigType_SINGLE_SELECT:
			if c.GetSingleSelect().Default != nil {
				val.Value = &models.Value_SingleSelect{
					SingleSelect: c.GetSingleSelect().Default,
				}
			}
		case models.ConfigType_MULTI_SELECT:
			if c.GetMultiSelect().Default != nil {
				val.Value = &models.Value_MultiSelect{
					MultiSelect: c.GetMultiSelect().Default,
				}
			}
		case models.ConfigType_STRING_LIST:
			if c.GetStringList().Default != nil {
				val.Value = &models.Value_StringList{
					StringList: c.GetStringList().Default,
				}
			}
		case models.ConfigType_INTEGER:
			if c.GetInteger().Default != nil {
				val.Value = &models.Value_Integer{
					Integer: c.GetInteger().Default,
				}
			}
		default:
			return nil, grpcErrorf(codes.Unimplemented, "config type %s not supported", c.Type)
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
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	transformedConfigs := make([]*settings.Config, len(configs))
	for i, c := range configs {
		transformedConfigs[i] = transformModelToConfig(c)
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
