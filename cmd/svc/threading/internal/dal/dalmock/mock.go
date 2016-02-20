package dalmock

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"golang.org/x/net/context"
)

// Build time check for matching against the interface
var d dal.DAL = &DAL{}

type DAL struct {
	*mock.Expector
}

func New(t *testing.T) *DAL {
	return &DAL{&mock.Expector{T: t}}
}

func (dl *DAL) Transact(ctx context.Context, trans func(context.Context, dal.DAL) error) error {
	return trans(ctx, dl)
}

func (dl *DAL) CreateOnboardingState(ctx context.Context, threadID models.ThreadID, entityID string) error {
	rets := dl.Expector.Record(threadID, entityID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) CreateSavedQuery(ctx context.Context, sq *models.SavedQuery) (models.SavedQueryID, error) {
	rets := dl.Expector.Record(sq)
	if len(rets) == 0 {
		return models.SavedQueryID{}, nil
	}
	return rets[0].(models.SavedQueryID), mock.SafeError(rets[1])
}

func (dl *DAL) CreateThread(ctx context.Context, thread *models.Thread) (models.ThreadID, error) {
	rets := dl.Expector.Record(thread)
	if len(rets) == 0 {
		return models.ThreadID{}, nil
	}
	return rets[0].(models.ThreadID), mock.SafeError(rets[1])
}

func (dl *DAL) CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error {
	rets := dl.Expector.Record(tds)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) CreateThreadLink(ctx context.Context, thread1ID, thread2ID models.ThreadID) error {
	rets := dl.Expector.Record(thread1ID, thread2ID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) DeleteThread(ctx context.Context, threadID models.ThreadID) error {
	rets := dl.Expector.Record(threadID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) IterateThreads(ctx context.Context, orgID string, forExternal bool, it *dal.Iterator) (*dal.ThreadConnection, error) {
	rets := dl.Expector.Record(orgID, forExternal, it)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.ThreadConnection), mock.SafeError(rets[1])
}

func (dl *DAL) IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *dal.Iterator) (*dal.ThreadItemConnection, error) {
	rets := dl.Expector.Record(threadID, forExternal, it)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.ThreadItemConnection), mock.SafeError(rets[1])
}

func (dl *DAL) LinkedThread(ctx context.Context, threadID models.ThreadID) (*models.Thread, error) {
	rets := dl.Expector.Record(threadID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) OnboardingState(ctx context.Context, threadID models.ThreadID, forUpdate bool) (*models.OnboardingState, error) {
	rets := dl.Expector.Record(threadID, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.OnboardingState), mock.SafeError(rets[1])
}

func (dl *DAL) OnboardingStateForEntity(ctx context.Context, entityID string, forUpdate bool) (*models.OnboardingState, error) {
	rets := dl.Expector.Record(entityID, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.OnboardingState), mock.SafeError(rets[1])
}

func (dl *DAL) PostMessage(ctx context.Context, req *dal.PostMessageRequest) (*models.ThreadItem, error) {
	rets := dl.Expector.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ThreadItem), mock.SafeError(rets[1])
}

func (dl *DAL) RecordThreadEvent(ctx context.Context, threadID models.ThreadID, actorEntityID string, event models.ThreadEvent) error {
	rets := dl.Expector.Record(threadID, actorEntityID, event)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.SavedQuery), mock.SafeError(rets[1])
}

func (dl *DAL) SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.SavedQuery), mock.SafeError(rets[1])
}

func (dl *DAL) Thread(ctx context.Context, id models.ThreadID) (*models.Thread, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadItem(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ThreadItem), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error) {
	rets := dl.Expector.Record(threadID, after)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]models.ThreadItemID), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadItemViewDetails), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadMemberships(ctx context.Context, threadIDs []models.ThreadID, entityIDs []string, forUpdate bool) (map[string][]*models.ThreadMember, error) {
	rets := dl.Expector.Record(threadIDs, entityIDs, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(map[string][]*models.ThreadMember), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadMembers(ctx context.Context, threadIDs models.ThreadID) ([]*models.ThreadMember, error) {
	rets := dl.Expector.Record(threadIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadMember), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	rets := dl.Expector.Record(entityID, primaryOnly)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadsForOrg(ctx context.Context, organizationID string) ([]*models.Thread, error) {
	rets := dl.Expector.Record(organizationID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *dal.MemberUpdate) error {
	rets := dl.Expector.Record(threadID, entityID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UpdateOnboardingState(ctx context.Context, threadID models.ThreadID, update *dal.OnboardingStateUpdate) error {
	rets := dl.Expector.Record(threadID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}
