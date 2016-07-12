package server

import (
	"fmt"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/media"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrf = grpc.Errorf

func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "Media - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

func grpcError(err error) error {
	if grpc.Code(err) == codes.Unknown {
		return grpcErrorf(codes.Internal, err.Error())
	}
	return err
}

type server struct {
	dl             dal.DAL
	svc            service.Service
	mediaAPIDomain string
}

// New returns an initialized instance of server
func New(dl dal.DAL, svc service.Service, mediaAPIDomain string) media.MediaServer {
	srv := &server{
		dl:             dl,
		svc:            svc,
		mediaAPIDomain: mediaAPIDomain,
	}
	return srv
}

func (s *server) CanAccess(ctx context.Context, rd *media.CanAccessRequest) (*media.CanAccessResponse, error) {
	for _, mID := range rd.MediaIDs {
		mediaID, err := dal.ParseMediaID(mID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, err.Error())
		}
		if err := s.svc.CanAccess(ctx, mediaID, rd.AccountID); errors.Cause(err) == service.ErrAccessDenied {
			return &media.CanAccessResponse{
				CanAccess: false,
			}, nil
		} else if err != nil {
			return nil, grpcError(err)
		}
	}
	return &media.CanAccessResponse{
		CanAccess: true,
	}, nil
}

func (s *server) ClaimMedia(ctx context.Context, rd *media.ClaimMediaRequest) (*media.ClaimMediaResponse, error) {
	ownerType, err := dal.ParseMediaOwnerType(rd.OwnerType.String())
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown owner type: %s", rd.OwnerType)
	}
	if err := s.dl.Transact(func(dl dal.DAL) error {
		for _, id := range rd.MediaIDs {
			mID, err := dal.ParseMediaID(id)
			if err != nil {
				return fmt.Errorf("Unable to parse media ID %s", id)
			}
			_, err = dl.UpdateMedia(mID, &dal.MediaUpdate{
				OwnerType: &ownerType,
				OwnerID:   ptr.String(rd.OwnerID),
			})
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, grpcError(err)
	}
	return &media.ClaimMediaResponse{}, nil
}

func (s *server) MediaInfos(ctx context.Context, rd *media.MediaInfosRequest) (*media.MediaInfosResponse, error) {
	var err error
	ids := make([]dal.MediaID, len(rd.MediaIDs))
	for i, id := range rd.MediaIDs {
		ids[i], err = dal.ParseMediaID(id)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse media ID %s", id)
		}
	}
	m, err := s.dl.Medias(ids)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if len(m) == 0 {
		return nil, grpcErrorf(codes.NotFound, "No matches found for provided media ids %v", rd.MediaIDs)
	}
	rms, err := s.transformMediasToResponse(m)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	mMap := make(map[string]*media.MediaInfo, len(rms))
	for _, rm := range rms {
		mMap[rm.ID] = rm
	}
	return &media.MediaInfosResponse{
		MediaInfos: mMap,
	}, nil
}

func (s *server) UpdateMedia(ctx context.Context, rd *media.UpdateMediaRequest) (*media.UpdateMediaResponse, error) {
	mediaID, err := dal.ParseMediaID(rd.MediaID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	if _, err := s.dl.UpdateMedia(mediaID, &dal.MediaUpdate{
		Public: ptr.Bool(rd.Public),
	}); err != nil {
		return nil, grpcError(err)
	}
	resp, err := s.MediaInfos(ctx, &media.MediaInfosRequest{
		MediaIDs: []string{rd.MediaID},
	})
	if err != nil {
		return nil, grpcError(err)
	}
	if _, ok := resp.MediaInfos[rd.MediaID]; !ok {
		return nil, grpcErrorf(codes.NotFound, "Expected media info to be returned for media id %s but got %+v", rd.MediaID, resp.MediaInfos)
	}
	return &media.UpdateMediaResponse{
		MediaInfo: resp.MediaInfos[rd.MediaID],
	}, nil
}
