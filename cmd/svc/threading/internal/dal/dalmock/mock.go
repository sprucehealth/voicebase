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

func (dl *DAL) CreateSetupThreadState(ctx context.Context, threadID models.ThreadID, entityID string) error {
	rets := dl.Expector.Record(threadID, entityID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
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

func (dl *DAL) CreateThreadLink(ctx context.Context, thread1, thread2 *dal.ThreadLink) error {
	rets := dl.Expector.Record(thread1, thread2)
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

func (dl *DAL) IterateThreads(ctx context.Context, orgID, viewerID string, forExternal bool, it *dal.Iterator) (*dal.ThreadConnection, error) {
	rets := dl.Expector.Record(orgID, viewerID, forExternal, it)
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

func (dl *DAL) LinkedThread(ctx context.Context, threadID models.ThreadID) (*models.Thread, bool, error) {
	rets := dl.Expector.Record(threadID)
	if len(rets) == 0 {
		return nil, false, nil
	}
	return rets[0].(*models.Thread), rets[1].(bool), mock.SafeError(rets[2])
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

func (dl *DAL) SetupThreadState(ctx context.Context, threadID models.ThreadID, opts ...dal.QueryOption) (*models.SetupThreadState, error) {
	rets := dl.Expector.Record(threadID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.SetupThreadState), mock.SafeError(rets[1])
}

func (dl *DAL) SetupThreadStateForEntity(ctx context.Context, entityID string, opts ...dal.QueryOption) (*models.SetupThreadState, error) {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.SetupThreadState), mock.SafeError(rets[1])
}

func (dl *DAL) Thread(ctx context.Context, id models.ThreadID, opts ...dal.QueryOption) (*models.Thread, error) {
	rets := dl.Expector.Record(append([]interface{}{id}, optsToInterfaces(opts)...)...)
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

func (dl *DAL) ThreadEntities(ctx context.Context, threadIDs []models.ThreadID, entityID string, opts ...dal.QueryOption) (map[string]*models.ThreadEntity, error) {
	rets := dl.Expector.Record(append([]interface{}{threadIDs, entityID}, optsToInterfaces(opts)...)...)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(map[string]*models.ThreadEntity), mock.SafeError(rets[1])
}

func (dl *DAL) EntitiesForThread(ctx context.Context, threadIDs models.ThreadID) ([]*models.ThreadEntity, error) {
	rets := dl.Expector.Record(threadIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ThreadEntity), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	rets := dl.Expector.Record(entityID, primaryOnly)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadsForOrg(ctx context.Context, organizationID string, typ models.ThreadType, limit int) ([]*models.Thread, error) {
	rets := dl.Expector.Record(organizationID, typ, limit)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) UpdateSetupThreadState(ctx context.Context, threadID models.ThreadID, update *dal.SetupThreadStateUpdate) error {
	rets := dl.Expector.Record(threadID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UpdateThread(ctx context.Context, threadID models.ThreadID, update *dal.ThreadUpdate) error {
	rets := dl.Expector.Record(threadID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UpdateThreadEntity(ctx context.Context, threadID models.ThreadID, entityID string, update *dal.ThreadEntityUpdate) error {
	rets := dl.Expector.Record(threadID, entityID, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UpdateThreadMembers(ctx context.Context, threadID models.ThreadID, members []string) error {
	rets := dl.Expector.Record(threadID, members)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func optsToInterfaces(opts []dal.QueryOption) []interface{} {
	ifs := make([]interface{}, len(opts))
	for i, o := range opts {
		ifs[i] = o
	}
	return ifs
}
