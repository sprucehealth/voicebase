package server

import (
	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var grpcErrorf = grpc.Errorf

type server struct {
	layoutStore layout.Storage
	dal         dal.DAL
	layout      layout.LayoutClient
}

func New(dal dal.DAL, layoutClient layout.LayoutClient, layoutStore layout.Storage) care.CareServer {
	return &server{
		layoutStore: layoutStore,
		dal:         dal,
		layout:      layoutClient,
	}
}

func (s *server) CreateVisit(ctx context.Context, in *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.LayoutVersionID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "layout_version_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	}

	visitToCreate := &models.Visit{
		Name:            in.Name,
		LayoutVersionID: in.LayoutVersionID,
		EntityID:        in.EntityID,
	}

	_, err := s.dal.CreateVisit(ctx, visitToCreate)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.CreateVisitResponse{
		Visit: transformVisitToResponse(visitToCreate),
	}, nil
}

func (s *server) GetVisit(ctx context.Context, in *care.GetVisitRequest) (*care.GetVisitResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "id required")
	}

	visitID, err := models.ParseVisitID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit id: %s", err.Error())
	}

	v, err := s.dal.Visit(ctx, visitID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit %s not found", visitID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.GetVisitResponse{
		Visit: transformVisitToResponse(v),
	}, nil
}
