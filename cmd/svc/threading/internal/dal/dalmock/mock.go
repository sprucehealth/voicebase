package dalmock

import (
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
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

func (dl *DAL) AddThreadFollowers(ctx context.Context, threadID models.ThreadID, followers []string) error {
	rets := dl.Expector.Record(threadID, followers)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) AddThreadMembers(ctx context.Context, threadID models.ThreadID, members []string) error {
	rets := dl.Expector.Record(threadID, members)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
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

func (dl *DAL) CreateThreadItem(ctx context.Context, item *models.ThreadItem) error {
	rets := dl.Expector.Record(item)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
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

func (dl *DAL) DeleteMessage(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, bool, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return nil, false, nil
	}
	return rets[0].(*models.ThreadItem), rets[1].(bool), mock.SafeError(rets[2])
}

func (dl *DAL) IterateThreads(ctx context.Context, query *models.Query, memberEntityIDs []string, viewerID string, forExternal bool, it *dal.Iterator) (*dal.ThreadConnection, error) {
	rets := dl.Expector.Record(query, memberEntityIDs, viewerID, forExternal, it)
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

func (dl *DAL) RemoveThreadFollowers(ctx context.Context, threadID models.ThreadID, followers []string) error {
	rets := dl.Expector.Record(threadID, followers)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) RemoveThreadMembers(ctx context.Context, threadID models.ThreadID, members []string) error {
	rets := dl.Expector.Record(threadID, members)
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

func (dl *DAL) DeleteSavedQueries(ctx context.Context, ids []models.SavedQueryID) error {
	rets := dl.Expector.Record(ids)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) SavedQueryTemplates(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
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

func (dl *DAL) Threads(ctx context.Context, ids []models.ThreadID, opts ...dal.QueryOption) ([]*models.Thread, error) {
	rets := dl.Expector.Record(append([]interface{}{ids}, optsToInterfaces(opts)...)...)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.Thread), mock.SafeError(rets[1])
}

func (dl *DAL) ThreadItem(ctx context.Context, id models.ThreadItemID, opts ...dal.QueryOption) (*models.ThreadItem, error) {
	rets := dl.Expector.Record(append([]interface{}{id}, optsToInterfaces(opts)...)...)
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

func (dl *DAL) EntitiesForThread(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadEntity, error) {
	rets := dl.Expector.Record(threadID)
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

func (dl *DAL) ThreadsWithEntity(ctx context.Context, entityID string, ids []models.ThreadID) ([]*models.Thread, []*models.ThreadEntity, error) {
	rets := dl.Expector.Record(entityID, ids)
	if len(rets) == 0 {
		return nil, nil, nil
	}
	return rets[0].([]*models.Thread), rets[1].([]*models.ThreadEntity), mock.SafeError(rets[2])
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

func (dl *DAL) UpdateSavedQuery(ctx context.Context, id models.SavedQueryID, update *dal.SavedQueryUpdate) error {
	rets := dl.Expector.Record(id, update)
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

func (dl *DAL) AddItemsToSavedQueryIndex(ctx context.Context, items []*dal.SavedQueryThread) error {
	rets := dl.Expector.Record(items)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) IterateThreadsInSavedQuery(ctx context.Context, sqID models.SavedQueryID, viewerEntityID string, it *dal.Iterator) (*dal.ThreadConnection, error) {
	rets := dl.Expector.Record(sqID, viewerEntityID, it)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.ThreadConnection), mock.SafeError(rets[1])
}

func (dl *DAL) RemoveAllItemsFromSavedQueryIndex(ctx context.Context, sqID models.SavedQueryID) error {
	rets := dl.Expector.Record(sqID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) RemoveItemsFromSavedQueryIndex(ctx context.Context, items []*dal.SavedQueryThread) error {
	rets := dl.Expector.Record(items)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) RemoveThreadFromAllSavedQueryIndexes(ctx context.Context, threadID models.ThreadID) error {
	rets := dl.Expector.Record(threadID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) RebuildNotificationsSavedQuery(ctx context.Context, entityID string) error {
	rets := dl.Expector.Record(entityID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UnreadNotificationsCounts(ctx context.Context, entityIDs []string) (map[string]int, error) {
	rets := dl.Expector.Record(entityIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(map[string]int), mock.SafeError(rets[1])
}

func (dl *DAL) CreateSavedMessage(ctx context.Context, sm *models.SavedMessage) (models.SavedMessageID, error) {
	rets := dl.Expector.Record(sm)
	if len(rets) == 0 {
		return models.SavedMessageID{}, nil
	}
	return rets[0].(models.SavedMessageID), mock.SafeError(rets[1])
}

func (dl *DAL) DeleteSavedMessages(ctx context.Context, ids []models.SavedMessageID) (int, error) {
	rets := dl.Expector.Record(ids)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int), mock.SafeError(rets[1])
}

func (dl *DAL) SavedMessages(ctx context.Context, ids []models.SavedMessageID) ([]*models.SavedMessage, error) {
	rets := dl.Expector.Record(ids)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.SavedMessage), mock.SafeError(rets[1])
}

func (dl *DAL) SavedMessagesForEntities(ctx context.Context, ownerEntityIDs []string) ([]*models.SavedMessage, error) {
	rets := dl.Expector.Record(ownerEntityIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.SavedMessage), mock.SafeError(rets[1])
}

func (dl *DAL) UnreadMessagesInThread(ctx context.Context, threadID models.ThreadID, entityID string, external bool) (int, error) {
	rets := dl.Expector.Record(threadID, entityID)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int), mock.SafeError(rets[1])
}

func (dl *DAL) UpdateSavedMessage(ctx context.Context, id models.SavedMessageID, update *dal.SavedMessageUpdate) error {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[1])
}

func (dl *DAL) CreateScheduledMessage(ctx context.Context, model *models.ScheduledMessage) (models.ScheduledMessageID, error) {
	rets := dl.Expector.Record(model)
	if len(rets) == 0 {
		return models.EmptyScheduledMessageID(), nil
	}
	return rets[0].(models.ScheduledMessageID), mock.SafeError(rets[1])
}

func (dl *DAL) DeleteScheduledMessage(ctx context.Context, id models.ScheduledMessageID) (int64, error) {
	rets := dl.Expector.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *DAL) ScheduledMessage(ctx context.Context, id models.ScheduledMessageID, opts ...dal.QueryOption) (*models.ScheduledMessage, error) {
	rets := dl.Expector.Record(id, optsToInterfaces(opts))
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ScheduledMessage), mock.SafeError(rets[1])
}

func (dl *DAL) ScheduledMessages(ctx context.Context, status []models.ScheduledMessageStatus, scheduledForBefore time.Time, opts ...dal.QueryOption) ([]*models.ScheduledMessage, error) {
	rets := dl.Expector.Record(status, scheduledForBefore, optsToInterfaces(opts))
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ScheduledMessage), mock.SafeError(rets[1])
}

func (dl *DAL) ScheduledMessagesForThread(ctx context.Context, threadID models.ThreadID, status []models.ScheduledMessageStatus, opts ...dal.QueryOption) ([]*models.ScheduledMessage, error) {
	rets := dl.Expector.Record(threadID, status, optsToInterfaces(opts))
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*models.ScheduledMessage), mock.SafeError(rets[1])
}

func (dl *DAL) AddThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error {
	rets := dl.Expector.Record(orgID, threadID, tags)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) RemoveThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error {
	rets := dl.Expector.Record(orgID, threadID, tags)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) TagsForOrg(ctx context.Context, orgID, prefix string) ([]models.Tag, error) {
	rets := dl.Expector.Record(orgID, prefix)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]models.Tag), mock.SafeError(rets[1])
}

func (dl *DAL) UpdateMessage(ctx context.Context, threadID models.ThreadID, itemID models.ThreadItemID, req *dal.PostMessageRequest) error {
	rets := dl.Expector.Record(threadID, itemID, req)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *DAL) UpdateScheduledMessage(ctx context.Context, id models.ScheduledMessageID, update *models.ScheduledMessageUpdate) (int64, error) {
	rets := dl.Expector.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func optsToInterfaces(opts []dal.QueryOption) []interface{} {
	ifs := make([]interface{}, len(opts))
	for i, o := range opts {
		ifs[i] = o
	}
	return ifs
}
