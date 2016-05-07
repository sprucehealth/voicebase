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

func New(t *testing.T) *mockDAL {
	return &mockDAL{
		&mock.Expector{
			T: t,
		},
	}
}

func (m *mockDAL) CreateVisit(ctx context.Context, visit *models.Visit) (models.VisitID, error) {
	rets := m.Record(visit)
	if len(rets) == 0 {
		return models.EmptyVisitID(), nil
	}

	return rets[0].(models.VisitID), mock.SafeError(rets[1])
}

func (m *mockDAL) Visit(ctx context.Context, id models.VisitID) (*models.Visit, error) {
	rets := m.Record(id)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.Visit), mock.SafeError(rets[1])
}
