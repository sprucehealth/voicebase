package server

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/media"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

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
			return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
		}
		if err := s.svc.CanAccess(ctx, mediaID, rd.AccountID); errors.Cause(err) == service.ErrAccessDenied {
			return &media.CanAccessResponse{
				CanAccess: false,
			}, nil
		} else if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return &media.CanAccessResponse{
		CanAccess: true,
	}, nil
}

func (s *server) ClaimMedia(ctx context.Context, rd *media.ClaimMediaRequest) (*media.ClaimMediaResponse, error) {
	ownerType, err := dal.ParseMediaOwnerType(rd.OwnerType.String())
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown owner type: %s", rd.OwnerType)
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
		return nil, errors.Trace(err)
	}
	return &media.ClaimMediaResponse{}, nil
}

func (s *server) CloneMedia(ctx context.Context, req *media.CloneMediaRequest) (*media.CloneMediaResponse, error) {
	id, err := dal.ParseMediaID(req.MediaID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid media ID %s", req.MediaID)
	}
	ownerType, err := dal.ParseMediaOwnerType(req.OwnerType.String())
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown owner type: %s", req.OwnerType)
	}
	if req.OwnerID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Owner ID required")
	}
	meta, err := s.svc.CopyMedia(ctx, ownerType, req.OwnerID, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "media %s not found", id)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	ms, err := s.dl.Medias([]dal.MediaID{meta.MediaID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(ms) == 0 {
		return nil, errors.Errorf("media %s was just created but is not found", meta.MediaID)
	}
	rms, err := s.transformMediasToResponse([]*dal.Media{ms[0]})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &media.CloneMediaResponse{
		MediaInfo: rms[0],
	}, nil
}

func (s *server) MediaInfos(ctx context.Context, rd *media.MediaInfosRequest) (*media.MediaInfosResponse, error) {
	var err error
	ids := make([]dal.MediaID, len(rd.MediaIDs))
	for i, id := range rd.MediaIDs {
		ids[i], err = dal.ParseMediaID(id)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse media ID %s", id)
		}
	}
	m, err := s.dl.Medias(ids)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(m) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "No matches found for provided media ids %v", rd.MediaIDs)
	}
	rms, err := s.transformMediasToResponse(m)
	if err != nil {
		return nil, errors.Trace(err)
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
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if _, err := s.dl.UpdateMedia(mediaID, &dal.MediaUpdate{
		Public: ptr.Bool(rd.Public),
	}); err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := s.MediaInfos(ctx, &media.MediaInfosRequest{
		MediaIDs: []string{rd.MediaID},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if _, ok := resp.MediaInfos[rd.MediaID]; !ok {
		return nil, grpc.Errorf(codes.NotFound, "Expected media info to be returned for media id %s but got %+v", rd.MediaID, resp.MediaInfos)
	}
	return &media.UpdateMediaResponse{
		MediaInfo: resp.MediaInfos[rd.MediaID],
	}, nil
}
