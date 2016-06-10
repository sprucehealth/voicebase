package server

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/layout/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/ptr"
	samlparser "github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type server struct {
	dal   dal.DAL
	store layout.Storage
}

func New(dal dal.DAL, store layout.Storage) layout.LayoutServer {
	return &server{
		dal:   dal,
		store: store,
	}
}

var grpcErrorf = grpc.Errorf

func (s *server) ListVisitLayouts(ctx context.Context, in *layout.ListVisitLayoutsRequest) (*layout.ListVisitLayoutsResponse, error) {
	if in.VisitCategoryID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_category_id required")
	}

	visitCategoryID, err := models.ParseVisitCategoryID(in.VisitCategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitLayouts, err := s.dal.VisitLayouts(visitCategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	res := &layout.ListVisitLayoutsResponse{
		VisitLayouts: make([]*layout.VisitLayout, len(visitLayouts)),
	}

	for i, visitLayout := range visitLayouts {
		res.VisitLayouts[i] = transformVisitLayoutToResponse(visitLayout, nil)
	}

	return res, nil
}

func (s *server) ListVisitCategories(context.Context, *layout.ListVisitCategoriesRequest) (*layout.ListVisitCategoriesResponse, error) {
	categories, err := s.dal.VisitCategories()
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	res := &layout.ListVisitCategoriesResponse{
		Categories: make([]*layout.VisitCategory, len(categories)),
	}

	for i, category := range categories {
		res.Categories[i] = transformCategoryToResponse(category)
	}

	return res, nil
}

func (s *server) CreateVisitLayout(ctx context.Context, in *layout.CreateVisitLayoutRequest) (*layout.CreateVisitLayoutResponse, error) {
	// validate
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout name required")
	} else if in.CategoryID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout category_id required")
	} else if in.SAML == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout saml required")
	} else if in.InternalName == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout internal name required")
	}

	categoryID, err := models.ParseVisitCategoryID(in.CategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse category_id: %s", err.Error())
	}

	activeVersion, err := s.processSAML(in.SAML)
	if err != nil {
		return nil, err
	}

	var visitLayout *models.VisitLayout
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		visitLayout = &models.VisitLayout{
			Name:         in.Name,
			InternalName: in.InternalName,
			CategoryID:   categoryID,
		}

		visitLayoutID, err := dl.CreateVisitLayout(ctx, visitLayout)
		if err != nil {
			return errors.Trace(err)
		}
		activeVersion.VisitLayoutID = visitLayoutID

		_, err = dl.CreateVisitLayoutVersion(ctx, activeVersion)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.CreateVisitLayoutResponse{
		VisitLayout: transformVisitLayoutToResponse(visitLayout, activeVersion),
	}, nil
}

func (s *server) UpdateVisitLayout(ctx context.Context, in *layout.UpdateVisitLayoutRequest) (*layout.UpdateVisitLayoutResponse, error) {
	if in.VisitLayoutID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout_id required")
	}

	visitLayoutID, err := models.ParseVisitLayoutID(in.VisitLayoutID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "cannot parse visit_layout_id: %s", err.Error())
	}

	update := &dal.VisitLayoutUpdate{}
	if in.UpdateName {
		if in.Name == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "name required for visit layout")
		}
		update.Name = ptr.String(in.Name)
	}
	if in.UpdateCategory {
		if in.CategoryID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "category_id required for visit layout update")
		}
		categoryID, err := models.ParseVisitCategoryID(in.CategoryID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "unable to parse category id")
		}
		update.CategoryID = &categoryID
	}

	var layoutVersionToCreate *models.VisitLayoutVersion

	if in.UpdateSAML {
		if in.SAML == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "saml required for visit layout update")
		}

		layoutVersionToCreate, err = s.processSAML(in.SAML)
		if err != nil {
			return nil, err
		}
	}

	// ensure that visit layout exists
	if _, err := s.dal.VisitLayout(ctx, visitLayoutID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit layout does not exist")
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {

		rowsUpdated, err := dl.UpdateVisitLayout(ctx, visitLayoutID, update)
		if err != nil {
			return errors.Trace(err)
		} else if rowsUpdated > 1 {
			return errors.Trace(fmt.Errorf("Expected 1 row to be updated for visit layout %s but got %d", visitLayoutID, rowsUpdated))
		}

		layoutVersionToCreate.VisitLayoutID = visitLayoutID
		if layoutVersionToCreate != nil {
			_, err := dl.CreateVisitLayoutVersion(ctx, layoutVersionToCreate)
			if err != nil {
				return errors.Trace(err)
			}

		}
		return nil

	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitLayout, err := s.dal.VisitLayout(ctx, visitLayoutID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	activeVersion, err := s.dal.ActiveVisitLayoutVersion(ctx, visitLayoutID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.UpdateVisitLayoutResponse{
		VisitLayout: transformVisitLayoutToResponse(visitLayout, activeVersion),
	}, nil
}

func (s *server) GetVisitLayout(ctx context.Context, in *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout_id required")
	}

	visitLayoutID, err := models.ParseVisitLayoutID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "cannot parse visit_layout: %s", err.Error())
	}

	visitLayout, err := s.dal.VisitLayout(ctx, visitLayoutID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit_layout with id %s not found", visitLayoutID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	activeVersion, err := s.dal.ActiveVisitLayoutVersion(ctx, visitLayoutID)
	if err != nil {
		if errors.Cause(err) != dal.ErrNotFound {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	return &layout.GetVisitLayoutResponse{
		VisitLayout: transformVisitLayoutToResponse(visitLayout, activeVersion),
	}, nil
}

func (s *server) GetVisitLayoutVersion(ctx context.Context, in *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error) {
	if in.ID == "" && in.VisitLayoutID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "either id or visit_layout_id required")
	}

	var visitLayoutVersion *models.VisitLayoutVersion
	if in.VisitLayoutID != "" {
		visitLayoutID, err := models.ParseVisitLayoutID(in.VisitLayoutID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "invalid visit_layout_id '%s': '%s'", in.VisitLayoutID, err.Error())
		}

		visitLayoutVersion, err = s.dal.ActiveVisitLayoutVersion(ctx, visitLayoutID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	} else {
		visitLayoutVersionID, err := models.ParseVisitLayoutVersionID(in.ID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "invalid visit_layout_version_id '%s' : %s", in.ID, err.Error())
		}

		visitLayoutVersion, err = s.dal.VisitLayoutVersion(ctx, visitLayoutVersionID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	return &layout.GetVisitLayoutVersionResponse{
		VisitLayoutVersion: transformVisitLayoutVersionToResponse(visitLayoutVersion),
	}, nil
}

func (s *server) DeleteVisitLayout(ctx context.Context, in *layout.DeleteVisitLayoutRequest) (*layout.DeleteVisitLayoutResponse, error) {
	if in.VisitLayoutID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout_id required")
	}

	visitLayoutID, err := models.ParseVisitLayoutID(in.VisitLayoutID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	rowsUpdated, err := s.dal.UpdateVisitLayout(ctx, visitLayoutID, &dal.VisitLayoutUpdate{
		Deleted: ptr.Bool(true),
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if rowsUpdated > 1 {
		return nil, grpcErrorf(codes.Internal, "Expected 1 layout to be deleted for visit layout %s but %d rows were deleted", visitLayoutID, rowsUpdated)
	}

	return &layout.DeleteVisitLayoutResponse{}, nil
}

func (s *server) CreateVisitCategory(ctx context.Context, in *layout.CreateVisitCategoryRequest) (*layout.CreateVisitCategoryResponse, error) {
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required to create visit category")
	}

	visitCategory := &models.VisitCategory{
		Name: in.Name,
	}

	if _, err := s.dal.CreateVisitCategory(ctx, visitCategory); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.CreateVisitCategoryResponse{
		Category: transformCategoryToResponse(visitCategory),
	}, nil
}

func (s *server) GetVisitCategory(ctx context.Context, in *layout.GetVisitCategoryRequest) (*layout.GetVisitCategoryResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_category_id required")
	}

	visitCategoryID, err := models.ParseVisitCategoryID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to query visit category id '%s' : %s", in.ID, err.Error())
	}

	visitCategory, err := s.dal.VisitCategory(ctx, visitCategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.GetVisitCategoryResponse{
		VisitCategory: transformCategoryToResponse(visitCategory),
	}, nil
}

func (s *server) UpdateVisitCategory(ctx context.Context, in *layout.UpdateVisitCategoryRequest) (*layout.UpdateVisitCategoryResponse, error) {
	if in.VisitCategoryID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_category_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	}

	categoryID, err := models.ParseVisitCategoryID(in.VisitCategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit_category_id %s: %s", in.VisitCategoryID, err.Error())
	}

	// ensure that category exists
	_, err = s.dal.VisitCategory(ctx, categoryID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.InvalidArgument, "visit_category with id %s not found", categoryID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	rowsUpdated, err := s.dal.UpdateVisitCategory(ctx, categoryID, &dal.VisitCategoryUpdate{
		Name: ptr.String(in.Name),
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if rowsUpdated > 1 {
		return nil, grpcErrorf(codes.Internal, "expected 1 row to be updated for %s but got %s", categoryID, rowsUpdated)
	}

	visitCategory, err := s.dal.VisitCategory(ctx, categoryID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.UpdateVisitCategoryResponse{
		Category: transformCategoryToResponse(visitCategory),
	}, nil
}

func (s *server) DeleteVisitCategory(ctx context.Context, in *layout.DeleteVisitCategoryRequest) (*layout.DeleteVisitCategoryResponse, error) {
	if in.VisitCategoryID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_category_id required")
	}

	categoryID, err := models.ParseVisitCategoryID(in.VisitCategoryID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	rowsUpdated, err := s.dal.UpdateVisitCategory(ctx, categoryID, &dal.VisitCategoryUpdate{
		Deleted: ptr.Bool(true),
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if rowsUpdated > 1 {
		return nil, grpcErrorf(codes.Internal, "Expected 1 layout to be deleted for visit category %s but %d rows were deleted", categoryID, rowsUpdated)
	}

	return &layout.DeleteVisitCategoryResponse{}, nil
}

func (s *server) GetVisitLayoutByVersion(ctx context.Context, in *layout.GetVisitLayoutByVersionRequest) (*layout.GetVisitLayoutByVersionResponse, error) {
	if in.VisitLayoutVersionID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_layout_version_id required")
	}

	visitLayoutVersionID, err := models.ParseVisitLayoutVersionID(in.VisitLayoutVersionID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit layout version id: %s", err.Error())
	}

	layoutVersion, err := s.dal.VisitLayoutVersion(ctx, visitLayoutVersionID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit layout version %s not found", visitLayoutVersionID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitLayout, err := s.dal.VisitLayout(ctx, layoutVersion.VisitLayoutID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit layout %s not found", layoutVersion.VisitLayoutID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &layout.GetVisitLayoutByVersionResponse{
		VisitLayout: transformVisitLayoutToResponse(visitLayout, layoutVersion),
	}, nil
}

func (s *server) processSAML(saml string) (*models.VisitLayoutVersion, error) {
	// validate SAML
	intakeSAML, err := samlparser.Parse(strings.NewReader(saml))
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid SAML: %s", err.Error())
	}

	// generate intakeLayout and reviewLayout
	intakeLayout, err := client.GenerateIntakeLayout(intakeSAML)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to generate intake layout from SAML: %s", err.Error())
	}

	reviewLayout, err := client.GenerateReviewLayout(intakeSAML)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to generate review layout from SAML: %s", err.Error())
	}

	mediaID, err := media.NewID()
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	samlLocation, err := s.store.PutSAML(mediaID+"-saml", saml)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	intakeLocation, err := s.store.PutIntake(mediaID+"-intake", intakeLayout)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	reviewLocation, err := s.store.PutReview(mediaID+"-review", reviewLayout)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &models.VisitLayoutVersion{
		SAMLLocation:         samlLocation,
		IntakeLayoutLocation: intakeLocation,
		ReviewLayoutLocation: reviewLocation,
	}, nil
}
