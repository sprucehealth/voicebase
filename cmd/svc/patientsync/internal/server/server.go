package server

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

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
)

var grpcErrf = grpc.Errorf

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "PaitentSync - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

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
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown source: %s", in.Source.String())
	}

	if in.OrganizationEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationEntityID required")
	}

	initiate := sync.Initiate{
		OrganizationEntityID: in.OrganizationEntityID,
		Source:               source,
	}

	data, err := initiate.Marshal()
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "Unable to marshal message: %s", err)
	}

	msg := base64.StdEncoding.EncodeToString(data)
	if _, err := s.sqsAPI.SendMessage(&sqs.SendMessageInput{
		MessageBody: &msg,
		QueueUrl:    &s.initiateSyncQueueURL,
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, "Unable to post message to start sync: %s", err)
	}

	return &patientsync.InitiateSyncResponse{}, nil
}

func (s *server) ConfigureSync(ctx context.Context, in *patientsync.ConfigureSyncRequest) (*patientsync.ConfigureSyncResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "organization_entity_id missing")
	}

	var config *sync.Config
	var externalID *string
	switch in.Source {
	case patientsync.SOURCE_HINT:
		grant, err := hint.GrantAPIKey(in.Token)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, "Unable to grant API key for %s with code %s: %s", in.OrganizationEntityID, in.Token, err.Error())
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
			return nil, grpcErrorf(codes.Internal, "Unable to get settings %s for %s: %s", psettings.ThreadTypeOptionConfig, in.OrganizationEntityID, err)
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
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown configuration for source %s", in.Source)
	}

	if err := s.dl.CreateSyncConfig(config, externalID); err != nil {
		return nil, grpcErrorf(codes.Internal, "Unable to create sync config for %s: %s", in.OrganizationEntityID, err.Error())
	}

	return &patientsync.ConfigureSyncResponse{}, nil
}

func (s *server) LookupSyncConfiguration(ctx context.Context, in *patientsync.LookupSyncConfigurationRequest) (*patientsync.LookupSyncConfigurationResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "organization_entity_id is missing")
	} else if in.Source == patientsync.SOURCE_UNKNOWN {
		return nil, grpcErrorf(codes.InvalidArgument, "source required")
	}

	syncConfig, err := s.dl.SyncConfigForOrg(in.OrganizationEntityID, in.Source.String())
	if errors.Cause(err) == dal.NotFound {
		return nil, grpcErrorf(codes.NotFound, "config not found for %s, %s", in.OrganizationEntityID, in.Source)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, "unable to lookup sync config for %s, %s : %s", in.OrganizationEntityID, in.Source, err)
	}

	var threadType patientsync.ThreadCreationType
	switch syncConfig.ThreadCreationType {
	case sync.THREAD_CREATION_TYPE_SECURE:
		threadType = patientsync.THREAD_CREATION_TYPE_SECURE
	case sync.THREAD_CREATION_TYPE_STANDARD:
		threadType = patientsync.THREAD_CREATION_TYPE_STANDARD

	}

	return &patientsync.LookupSyncConfigurationResponse{
		ThreadCreationType: threadType,
		PracticeID:         syncConfig.GetHint().PracticeID,
	}, nil
}
