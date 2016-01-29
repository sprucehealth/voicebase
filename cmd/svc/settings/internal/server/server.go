package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
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
		configs[i] = transformConfigToModel(cfg)
	}

	// TODO: Validate incoming config
	if err := s.dal.SetConfigs(configs); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	return &settings.RegisterConfigsResponse{}, nil
}

func (s *server) SetValue(ctx context.Context, in *settings.SetValueRequest) (*settings.SetValueResponse, error) {
	if in.Value.Key == nil || in.Value.Key.Key == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "config key not specified")
	}

	// pull up config
	config, err := s.getConfig(in.Value.Key.Key)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	} else if config == nil {
		return nil, grpc.Errorf(codes.NotFound, "config with key %s not found", in.Value.Key.Key)
	}

	// validate value against config
	transformedValue, err := validateValueAgainstConfig(in.Value, config)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	// TODO: Verify entityID is valid?
	// ensure that entityID is populated
	if err := s.dal.SetValues(in.NodeID, []*models.Value{transformedValue}); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
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
		return nil, grpc.Errorf(codes.Internal, err.Error())
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
			return nil, grpc.Errorf(codes.Internal, err.Error())
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
		default:
			return nil, grpc.Errorf(codes.Unimplemented, "config type %s not supported", config.Type)
		}

		transformedValues = append(transformedValues, transformModelToValue(val))
	}

	return &settings.GetValuesResponse{
		Values: transformedValues,
	}, nil
}

func (s *server) GetConfigs(ctx context.Context, in *settings.GetConfigsRequest) (*settings.GetConfigsResponse, error) {
	configs, err := s.dal.GetConfigs(in.Keys)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
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
