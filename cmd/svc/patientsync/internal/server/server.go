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
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/hint"
	"github.com/sprucehealth/backend/svc/patientsync"
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
}

func New(
	dl dal.DAL,
	initiateSyncQueueURL string,
	sqsAPI sqsiface.SQSAPI) patientsync.PatientSyncServer {
	return &server{
		dl:                   dl,
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
		config = &sync.Config{
			OrganizationEntityID: in.OrganizationEntityID,
			Source:               sync.SOURCE_HINT,
			ThreadCreationType:   transformThreadType(in.ThreadType),
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

	return nil, nil
}
