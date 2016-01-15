package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"golang.org/x/net/context"
)

// Build time check for matching against the interface
var d dal.DAL = &mockDAL{}

type mockDAL struct {
	*mock.Expector
}

func newMockDAL(t *testing.T) *mockDAL {
	return &mockDAL{&mock.Expector{T: t}}
}

func (dl *mockDAL) Transact(ctx context.Context, trans func(context.Context, dal.DAL) error) error {
	return trans(ctx, dl)
}

func (dl *mockDAL) CreateSavedQuery(ctx context.Context, sq *models.SavedQuery) (models.SavedQueryID, error) {
	rets := dl.Expector.Record(sq)
	if len(rets) == 0 {
		return models.SavedQueryID{}, nil
	}
	return rets[0].(models.SavedQueryID), mock.SafeError(rets[1])
}

func (dl *mockDAL) CreateThread(ctx context.Context, thread *models.Thread) (models.ThreadID, error) {
	rets := dl.Expector.Record(thread)
	if len(rets) == 0 {
		return models.ThreadID{}, nil
	}
	return rets[0].(models.ThreadID), mock.SafeError(rets[1])
}

func (dl *mockDAL) CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error {
	rets := dl.Expector.Record(tds)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) IterateThreads(ctx context.Context, orgID string, forExternal bool, it *dal.Iterator) (*dal.ThreadConnection, error) {
	rets := dl.Expector.Record(orgID, forExternal, it)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.ThreadConnection), mock.SafeError(rets[1])
}

func (dl *mockDAL) IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *dal.Iterator) (*dal.ThreadItemConnection, error) {
	rets := dl.Expector.Record(threadID, forExternal, it)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.ThreadItemConnection), mock.SafeError(rets[1])
}

func (dl *mockDAL) PostMessage(ctx context.Context, req *dal.PostMessageRequest) (*models.ThreadItem, error) {
	rets := dl.Expector.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ThreadItem), mock.SafeError(rets[1])
}

func (dl *mockDAL) SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.SavedQuery), mock.SafeError(rets[1])
}

func (dl *mockDAL) SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.SavedQuery), mock.SafeError(rets[1])
}

func (dl *mockDAL) Thread(ctx context.Context, id models.ThreadID) (*models.Thread, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.Thread), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadItem(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ThreadItem), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error) {
	rets := dl.Expector.Record(threadID, after)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]models.ThreadItemID), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadItemViewDetails), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadMemberships(ctx context.Context, threadIDs []models.ThreadID, entityID string, forUpdate bool) ([]*models.ThreadMember, error) {
	rets := dl.Expector.Record(threadIDs, threadIDs, entityID, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadMember), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadMembers(ctx context.Context, threadIDs models.ThreadID) ([]*models.ThreadMember, error) {
	rets := dl.Expector.Record(threadIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadMember), mock.SafeError(rets[1])
}

func (dl *mockDAL) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	rets := dl.Expector.Record(entityID, primaryOnly)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *dal.MemberUpdate) error {
	rets := dl.Expector.Record(threadID, entityID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}
