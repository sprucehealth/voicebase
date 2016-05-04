package mock

import (
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"golang.org/x/net/context"

	"testing"
)

func New(t *testing.T) *mockDAL {
	return &mockDAL{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

type mockDAL struct {
	*mock.Expector
}

func (m *mockDAL) Transact(ctx context.Context, trans func(context.Context, dal.DAL) error) error {
	return trans(ctx, m)
}

func (m *mockDAL) CreateVisitLayout(ctx context.Context, layout *models.VisitLayout) (models.VisitLayoutID, error) {
	rets := m.Record(layout)
	if len(rets) == 0 {
		return models.EmptyVisitLayoutID(), nil
	}

	return rets[0].(models.VisitLayoutID), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateVisitLayoutVersion(ctx context.Context, doc *models.VisitLayoutVersion) (models.VisitLayoutVersionID, error) {
	rets := m.Record(doc)
	if len(rets) == 0 {
		return models.EmptyVisitLayoutVersionID(), nil
	}

	return rets[0].(models.VisitLayoutVersionID), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateVisitCategory(ctx context.Context, category *models.VisitCategory) (models.VisitCategoryID, error) {
	rets := m.Record(category)
	if len(rets) == 0 {
		return models.EmptyVisitCategoryID(), nil
	}

	return rets[0].(models.VisitCategoryID), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitLayout(ctx context.Context, id models.VisitLayoutID) (*models.VisitLayout, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.VisitLayout), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitCategory(ctx context.Context, id models.VisitCategoryID) (*models.VisitCategory, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.VisitCategory), mock.SafeError(rets[1])
}

func (m *mockDAL) UpdateVisitCategory(ctx context.Context, id models.VisitCategoryID, update *dal.VisitCategoryUpdate) (int64, error) {
	rets := m.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}

	return rets[0].(int64), mock.SafeError(rets[1])
}

func (m *mockDAL) UpdateVisitLayout(ctx context.Context, id models.VisitLayoutID, update *dal.VisitLayoutUpdate) (int64, error) {
	rets := m.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}

	return rets[0].(int64), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitLayoutVersion(ctx context.Context, id models.VisitLayoutVersionID) (*models.VisitLayoutVersion, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.VisitLayoutVersion), mock.SafeError(rets[1])
}

func (m *mockDAL) ActiveVisitLayoutVersion(ctx context.Context, id models.VisitLayoutID) (*models.VisitLayoutVersion, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.VisitLayoutVersion), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitCategories() ([]*models.VisitCategory, error) {
	rets := m.Record()
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.VisitCategory), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitLayouts(visitCategoryID models.VisitCategoryID) ([]*models.VisitLayout, error) {
	rets := m.Record(visitCategoryID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.VisitLayout), mock.SafeError(rets[1])
}
