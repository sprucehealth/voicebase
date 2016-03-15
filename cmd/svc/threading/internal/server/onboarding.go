package server

import (
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/onboarding"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

// CreateOnboardingThread create a new onboarding thread
func (s *threadsServer) CreateOnboardingThread(ctx context.Context, in *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}

	var threadID models.ThreadID
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		nextMsg, summary, err := onboarding.Message(0, false, s.webDomain, in.OrganizationID, nil)
		if err != nil {
			return errors.Trace(err)
		}
		if nextMsg == "" {
			return errors.Trace(errors.New("empty first message for onboarding"))
		}
		threadID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:     in.OrganizationID,
			PrimaryEntityID:    in.PrimaryEntityID,
			LastMessageSummary: summary,
			Type:               models.ThreadTypeSetup,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := dl.PostMessage(ctx, &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.PrimaryEntityID,
			Internal:     false,
			Text:         nextMsg,
			Summary:      summary,
		}); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.CreateOnboardingState(ctx, threadID, in.OrganizationID))
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, errors.Trace(err).Error())
	}
	thread, err := s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateOnboardingThreadResponse{
		Thread: th,
	}, nil
}
