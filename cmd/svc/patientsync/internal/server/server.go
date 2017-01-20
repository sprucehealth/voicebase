package server

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	psettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/go-hint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type server struct {
	dl                   dal.DAL
	initiateSyncQueueURL string
	sqsAPI               sqsiface.SQSAPI
	settings             settings.SettingsClient
}

func New(
	dl dal.DAL,
	settings settings.SettingsClient,
	initiateSyncQueueURL string,
	sqsAPI sqsiface.SQSAPI) patientsync.PatientSyncServer {
	return &server{
		dl:                   dl,
		settings:             settings,
		initiateSyncQueueURL: initiateSyncQueueURL,
		sqsAPI:               sqsAPI,
	}
}

func (s *server) InitiateSync(ctx context.Context, in *patientsync.InitiateSyncRequest) (*patientsync.InitiateSyncResponse, error) {
	var source sync.Source
	switch in.Source {
	case patientsync.SOURCE_CSV:
		source = sync.SOURCE_CSV
	case patientsync.SOURCE_DRCHRONO:
		source = sync.SOURCE_DRCHRONO
	case patientsync.SOURCE_ELATION:
		source = sync.SOURCE_ELATION
	case patientsync.SOURCE_HINT:
		source = sync.SOURCE_HINT
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown source: %s", in.Source.String())
	}

	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationEntityID required")
	}

	// ensure that the preference matches what is currently stored in the config
	// its possible that the user changed the configuration
	threadTypeVal, err := settings.GetSingleSelectValue(ctx, s.settings, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key: psettings.ConfigKeyThreadTypeOption,
			},
		},
		NodeID: in.OrganizationEntityID,
	})
	if err != nil {
		return nil, errors.Errorf("Unable to get settings %s for %s: %s", psettings.ThreadTypeOptionConfig, in.OrganizationEntityID, err)
	}

	syncConfig, err := s.dl.SyncConfigForOrg(in.OrganizationEntityID, source.String())
	if err != nil {
		return nil, errors.Trace(err)
	}

	// update sync config
	if syncConfig.ThreadCreationType != transformThreadType(threadTypeVal.GetItem().ID) {
		syncConfig.ThreadCreationType = transformThreadType(threadTypeVal.GetItem().ID)
		if err := s.dl.CreateSyncConfig(syncConfig, &syncConfig.GetHint().PracticeID); err != nil {
			return nil, errors.Errorf("Unable to update sync config for %s: %s", in.OrganizationEntityID, err.Error())
		}
	}

	initiate := sync.Initiate{
		OrganizationEntityID: in.OrganizationEntityID,
		Source:               source,
	}

	data, err := initiate.Marshal()
	if err != nil {
		return nil, errors.Errorf("Unable to marshal message: %s", err)
	}

	msg := base64.StdEncoding.EncodeToString(data)
	if _, err := s.sqsAPI.SendMessage(&sqs.SendMessageInput{
		MessageBody: &msg,
		QueueUrl:    &s.initiateSyncQueueURL,
	}); err != nil {
		return nil, errors.Errorf("Unable to post message to start sync: %s", err)
	}

	return &patientsync.InitiateSyncResponse{}, nil
}

func (s *server) ConfigureSync(ctx context.Context, in *patientsync.ConfigureSyncRequest) (*patientsync.ConfigureSyncResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "organization_entity_id missing")
	}

	var config *sync.Config
	var externalID *string
	switch in.Source {
	case patientsync.SOURCE_HINT:
		grant, err := hint.GrantAPIKey(in.Token)
		if err != nil {
			return nil, errors.Errorf("Unable to grant API key for %s with code %s: %s", in.OrganizationEntityID, in.Token, err.Error())
		}
		var refreshToken string
		if grant.RefreshToken != nil {
			refreshToken = *grant.RefreshToken
		}

		var expiresIn uint64
		if grant.ExpiresIn != nil {
			expiresIn = uint64(grant.ExpiresIn.Unix())
		}

		threadTypeVal, err := settings.GetSingleSelectValue(ctx, s.settings, &settings.GetValuesRequest{
			Keys: []*settings.ConfigKey{
				{
					Key: psettings.ConfigKeyThreadTypeOption,
				},
			},
			NodeID: in.OrganizationEntityID,
		})
		if err != nil {
			return nil, errors.Errorf("Unable to get settings %s for %s: %s", psettings.ThreadTypeOptionConfig, in.OrganizationEntityID, err)
		}

		config = &sync.Config{
			OrganizationEntityID: in.OrganizationEntityID,
			Source:               sync.SOURCE_HINT,
			ThreadCreationType:   transformThreadType(threadTypeVal.GetItem().ID),
			Token: &sync.Config_Hint{
				Hint: &sync.HintToken{
					AccessToken:  grant.AccessToken,
					RefreshToken: refreshToken,
					ExpiresIn:    expiresIn,
					PracticeID:   grant.Practice.ID,
				},
			},
		}
		externalID = &grant.Practice.ID
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown configuration for source %s", in.Source)
	}

	if err := s.dl.CreateSyncConfig(config, externalID); err != nil {
		return nil, errors.Errorf("Unable to create sync config for %s: %s", in.OrganizationEntityID, err.Error())
	}

	return &patientsync.ConfigureSyncResponse{}, nil
}

func (s *server) LookupSyncConfiguration(ctx context.Context, in *patientsync.LookupSyncConfigurationRequest) (*patientsync.LookupSyncConfigurationResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "organization_entity_id is missing")
	} else if in.Source == patientsync.SOURCE_UNKNOWN {
		return nil, grpc.Errorf(codes.InvalidArgument, "source required")
	}

	syncConfig, err := s.dl.SyncConfigForOrg(in.OrganizationEntityID, in.Source.String())
	if errors.Cause(err) == dal.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "config not found for %s, %s", in.OrganizationEntityID, in.Source)
	} else if err != nil {
		return nil, errors.Errorf("unable to lookup sync config for %s, %s : %s", in.OrganizationEntityID, in.Source, err)
	}

	syncBookmark, err := s.dl.SyncBookmarkForOrg(in.OrganizationEntityID)
	if errors.Cause(err) != dal.NotFound && err != nil {
		golog.Errorf("unable to lookup sync bookmark for org %s", in.OrganizationEntityID)
	}

	return &patientsync.LookupSyncConfigurationResponse{
		Config: transformSyncConfigurationToResponse(syncConfig, syncBookmark),
	}, nil
}

func (s *server) UpdateSyncConfiguration(ctx context.Context, in *patientsync.UpdateSyncConfigurationRequest) (*patientsync.UpdateSyncConfigurationResponse, error) {
	if in.Source == patientsync.SOURCE_UNKNOWN {
		return nil, grpc.Errorf(codes.InvalidArgument, "source required")
	} else if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "organization_entity_id required")

	}

	syncConfig, err := s.dl.SyncConfigForOrg(in.OrganizationEntityID, in.Source.String())
	if errors.Cause(err) == dal.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "config for %s not found", in.OrganizationEntityID)
	} else if err != nil {
		return nil, errors.Errorf("unable to lookup config for %s: %s", in.OrganizationEntityID, err.Error())
	}

	tagMappings := make([]*sync.TagMappingItem, len(syncConfig.TagMappings))
	for i, item := range in.TagMappings {
		tagMappings[i] = &sync.TagMappingItem{
			Tag: item.Tag,
		}

		switch t := item.Key.(type) {
		case *patientsync.TagMappingItem_ProviderID:
			tagMappings[i].Key = &sync.TagMappingItem_ProviderID{
				ProviderID: t.ProviderID,
			}
		default:
			return nil, errors.Errorf("Unknown key type for tag %s: %T", item.Tag, t)
		}
	}
	syncConfig.TagMappings = tagMappings

	switch in.Source {
	case patientsync.SOURCE_HINT:
		if err := s.dl.CreateSyncConfig(syncConfig, &syncConfig.GetHint().PracticeID); err != nil {
			return nil, errors.Errorf("unable to update sync config for %s: %s", in.OrganizationEntityID, err)
		}
	case patientsync.SOURCE_UNKNOWN:
		return nil, errors.Errorf("Unknown source %s for %s", in.Source, in.OrganizationEntityID)
	}

	syncConfig, err = s.dl.SyncConfigForOrg(in.OrganizationEntityID, in.Source.String())
	if err != nil {
		return nil, errors.Errorf("unable to lookup config after update for %s : %s", in.OrganizationEntityID, err)
	}

	syncBookmark, err := s.dl.SyncBookmarkForOrg(in.OrganizationEntityID)
	if errors.Cause(err) != dal.NotFound && err != nil {
		return nil, errors.Errorf("unable to lookup sync bookmark for %s : %s", in.OrganizationEntityID, err)
	}

	return &patientsync.UpdateSyncConfigurationResponse{
		Config: transformSyncConfigurationToResponse(syncConfig, syncBookmark),
	}, nil
}
