package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"golang.org/x/net/context"
)

var _ dal.DAL = &mockDAL{}

type mockDAL struct {
	*mock.Expector
}

func New(t testing.TB) *mockDAL {
	return &mockDAL{
		&mock.Expector{
			T: t,
		},
	}
}

func (m *mockDAL) Transact(ctx context.Context, trans func(ctx context.Context, dl dal.DAL) error) error {
	return trans(ctx, m)
}

func (m *mockDAL) CarePlan(ctx context.Context, id models.CarePlanID) (*models.CarePlan, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.CarePlan), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateCarePlan(ctx context.Context, cp *models.CarePlan) (models.CarePlanID, error) {
	rets := m.Record(cp)
	if len(rets) == 0 {
		return models.EmptyCarePlanID(), nil
	}
	return rets[0].(models.CarePlanID), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateVisit(ctx context.Context, visit *models.Visit) (models.VisitID, error) {
	rets := m.Record(visit)
	if len(rets) == 0 {
		return models.EmptyVisitID(), nil
	}

	return rets[0].(models.VisitID), mock.SafeError(rets[1])
}

func (m *mockDAL) SubmitCarePlan(ctx context.Context, id models.CarePlanID, parentID string) error {
	rets := m.Record(id, parentID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *mockDAL) Visit(ctx context.Context, id models.VisitID, opts ...dal.QueryOption) (*models.Visit, error) {
	rets := m.Record(id)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.Visit), mock.SafeError(rets[1])
}

func (m *mockDAL) UpdateVisit(ctx context.Context, id models.VisitID, update *dal.VisitUpdate) (int64, error) {
	rets := m.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}

	return rets[0].(int64), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateVisitAnswer(ctx context.Context, visitID models.VisitID, actoryEntityID string, answer *models.Answer) error {
	rets := m.Record(visitID, actoryEntityID, answer)

	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *mockDAL) DeleteVisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (int64, error) {
	rets := m.Record(visitID, questionIDs)

	if len(rets) == 0 {
		return 0, nil
	}

	return rets[1].(int64), mock.SafeError(rets[1])
}

func (m *mockDAL) VisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (map[string]*models.Answer, error) {
	rets := m.Record(visitID, questionIDs)

	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(map[string]*models.Answer), mock.SafeError(rets[1])
}
