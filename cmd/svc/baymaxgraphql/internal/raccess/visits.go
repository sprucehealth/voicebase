package raccess

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (m *resourceAccessor) VisitLayout(ctx context.Context, req *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error) {
	res, err := m.layout.GetVisitLayout(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, req.ID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return res, nil
}

func (m *resourceAccessor) VisitLayoutByVersion(ctx context.Context, req *layout.GetVisitLayoutByVersionRequest) (*layout.GetVisitLayoutByVersionResponse, error) {

	res, err := m.layout.GetVisitLayoutByVersion(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, req.VisitLayoutVersionID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return res, nil
}

func (m *resourceAccessor) CreateVisit(ctx context.Context, req *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.LayoutVersionID)
	}

	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, errors.Trace(err)
	}

	res, err := m.care.CreateVisit(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res, nil
}

func (m *resourceAccessor) DeleteVisit(ctx context.Context, req *care.DeleteVisitRequest) (*care.DeleteVisitResponse, error) {

	if err := m.canAccessResource(ctx, req.ActorEntityID, m.orgsForEntity); err != nil {
		return nil, errors.Trace(err)
	}

	res, err := m.care.DeleteVisit(ctx, req)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return res, nil
}

func (m *resourceAccessor) Visit(ctx context.Context, req *care.GetVisitRequest) (*care.GetVisitResponse, error) {
	// first get the visit then check whether or not caller can access resource
	res, err := m.care.GetVisit(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, req.ID)
		}

		return nil, errors.InternalError(ctx, err)
	}

	if err := m.canAccessResource(ctx, res.Visit.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	return res, nil
}

func (m *resourceAccessor) Visits(ctx context.Context, req *care.GetVisitsRequest) (*care.GetVisitsResponse, error) {
	res, err := m.care.GetVisits(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, visit := range res.Visits {
		if err := m.canAccessResource(ctx, visit.EntityID, m.orgsForEntity); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return res, nil
}

func (m *resourceAccessor) SubmitVisit(ctx context.Context, req *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error) {
	_, err := m.Visit(ctx, &care.GetVisitRequest{
		ID: req.VisitID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.VisitID)
	}

	return m.care.SubmitVisit(ctx, req)
}

func (m *resourceAccessor) TriageVisit(ctx context.Context, req *care.TriageVisitRequest) (*care.TriageVisitResponse, error) {
	// helper method does the auth check
	_, err := m.Visit(ctx, &care.GetVisitRequest{
		ID: req.VisitID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.VisitID)
	}

	return m.care.TriageVisit(ctx, req)
}

func (m *resourceAccessor) VisitLayoutVersion(ctx context.Context, req *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error) {

	res, err := m.layout.GetVisitLayoutVersion(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, req.ID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return res, nil
}

func (m *resourceAccessor) CreateVisitAnswers(ctx context.Context, req *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error) {
	// only the patient can submit answers
	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.VisitID)
	}

	if err := m.canAccessResource(ctx, req.ActorEntityID, m.orgsForEntity); err != nil {
		return nil, errors.Trace(err)
	}

	res, err := m.care.CreateVisitAnswers(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, req.VisitID)
		}
		return nil, errors.Trace(err)
	}
	return res, nil
}

func (m *resourceAccessor) GetAnswersForVisit(ctx context.Context, req *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error) {
	_, err := m.Visit(ctx, &care.GetVisitRequest{
		ID: req.VisitID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	res, err := m.care.GetAnswersForVisit(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res, nil
}
