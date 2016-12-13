package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
)

// rebuildSavedQuery reindexes the threads in a saved query
func (s *threadsServer) rebuildSavedQuery(ctx context.Context, sq *models.SavedQuery) error {
	// we need all the memberships for the entity to be able to get a full list of threads
	ents, err := s.entityAndMemberships(ctx, sq.EntityID, []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT})
	if err != nil {
		errors.Trace(err)
	}
	if len(ents) == 0 {
		// Inactive entities don't need saved queries so clear them
		return errors.Trace(s.dal.RemoveAllItemsFromSavedQueryIndex(ctx, sq.ID))
	}

	// The notifications saved query is handled specifically so ignore them here test
	if sq.Type != models.SavedQueryTypeNotifications {
		forExternal := true
		entIDs := make([]string, len(ents))
		for i, e := range ents {
			entIDs[i] = e.ID
			if e.ID == sq.EntityID {
				forExternal = isExternalEntity(e)
			}
		}

		if err := s.dal.RemoveAllItemsFromSavedQueryIndex(ctx, sq.ID); err != nil {
			return errors.Trace(err)
		}

		it := &dal.Iterator{
			Direction: dal.FromStart,
			Count:     5000,
		}
		var newItems []*dal.SavedQueryThread
		for {
			tc, err := s.dal.IterateThreads(ctx, sq.Query, entIDs, sq.EntityID, forExternal, it)
			if err != nil {
				return errors.Trace(err)
			}
			newItems = newItems[:0]
			for _, e := range tc.Edges {
				// Sanity check to make sure the thread should really be included
				if ok, err := threadMatchesQuery(sq.Query, e.Thread, e.ThreadEntity, forExternal); err != nil {
					golog.ContextLogger(ctx).Errorf("Failed to match thread %s against query %s: %s", e.Thread.ID, sq.ID, err)
				} else if !ok {
					golog.ContextLogger(ctx).Errorf("Query %s returned non-matching thread %s from database", sq.ID, e.Thread.ID)
					continue
				}
				timestamp := e.Thread.LastMessageTimestamp
				if forExternal {
					timestamp = e.Thread.LastExternalMessageTimestamp
				}
				newItems = append(newItems, &dal.SavedQueryThread{
					ThreadID:     e.Thread.ID,
					SavedQueryID: sq.ID,
					Timestamp:    timestamp,
					Unread:       isUnread(e.Thread, e.ThreadEntity, forExternal),
				})
			}
			if len(newItems) != 0 {
				if err := s.dal.AddItemsToSavedQueryIndex(ctx, newItems); err != nil {
					return errors.Trace(err)
				}
			}
			if !tc.HasMore {
				break
			}
			it.StartCursor = tc.Edges[len(tc.Edges)-1].Cursor
		}
	}

	// Rebuild the notifications saved query when necessary
	if sq.Type == models.SavedQueryTypeNotifications || sq.NotificationsEnabled {
		if err := s.dal.RebuildNotificationsSavedQuery(ctx, sq.EntityID); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

type savedQueryUpdateResult struct {
	entityShouldBeNotified map[string]bool
}

// updateSavedQueriesAddThread updates all matching saved queries for a new thread
func (s *threadsServer) updateSavedQueriesAddThread(ctx context.Context, thread *models.Thread, memberEntityIDs []string) (*savedQueryUpdateResult, error) {
	if len(memberEntityIDs) == 0 {
		return &savedQueryUpdateResult{entityShouldBeNotified: make(map[string]bool)}, nil
	}
	// Resolve the root entities to be able to query for all possible saved queries
	entities, err := s.resolveInternalEntities(ctx, memberEntityIDs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var result savedQueryUpdateResult
	var newItems []*dal.SavedQueryThread
	newItems, result.entityShouldBeNotified = s.determineUpdatesToIndexForEntities(ctx, thread, entities, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(newItems) != 0 {
		return &result, errors.Trace(s.dal.AddItemsToSavedQueryIndex(ctx, newItems))
	}

	return &result, nil
}

// updateSavedQueriesForThread updates all relevant saved queries when a thread is updated (e.g. new post, membership change)
func (s *threadsServer) updateSavedQueriesForThread(ctx context.Context, thread *models.Thread) (*savedQueryUpdateResult, error) {
	// Get the list of members for the thread and follow memberships to get the root internal entities.
	tes, err := s.dal.EntitiesForThread(ctx, thread.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	memberEntityIDs := make([]string, 0, len(tes))
	teMap := make(map[string]*models.ThreadEntity, len(tes))
	for _, te := range tes {
		teMap[te.EntityID] = te
		if te.Member {
			memberEntityIDs = append(memberEntityIDs, te.EntityID)
		}
	}
	if len(memberEntityIDs) == 0 {
		return &savedQueryUpdateResult{
			entityShouldBeNotified: map[string]bool{},
		}, errors.Trace(s.dal.RemoveThreadFromAllSavedQueryIndexes(ctx, thread.ID))
	}
	entities, err := s.resolveInternalEntities(ctx, memberEntityIDs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var result savedQueryUpdateResult
	var addItems []*dal.SavedQueryThread
	addItems, result.entityShouldBeNotified = s.determineUpdatesToIndexForEntities(ctx, thread, entities, teMap)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := dl.RemoveThreadFromAllSavedQueryIndexes(ctx, thread.ID); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.AddItemsToSavedQueryIndex(ctx, addItems))
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &result, nil
}

// determineUpdatesToIndexForEntities determines the updates to make to the index
// for changes made to a thread. It returns a list of items to add to the index
// along with entities that should be notified about the update.
func (s *threadsServer) determineUpdatesToIndexForEntities(
	ctx context.Context,
	thread *models.Thread,
	entities []*directory.Entity,
	teMap map[string]*models.ThreadEntity,
) ([]*dal.SavedQueryThread, map[string]bool) {
	// Currently only supporting internal entities. The entity resolution guarantees that.
	externalEntity := false

	var newItems []*dal.SavedQueryThread
	entityShouldBeNotified := make(map[string]bool)
	for _, e := range entities {
		te := teMap[e.ID]
		sqs, err := s.dal.SavedQueries(ctx, e.ID)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to get saved queries for entity '%s': %s", e.ID, err)
			continue
		}
		// Find the notifications saved query
		var nsq *models.SavedQuery
		for _, sq := range sqs {
			if sq.Type == models.SavedQueryTypeNotifications {
				nsq = sq
				break
			}
		}
		// Match against saved queries to see which ones the thread falls into
		for _, sq := range sqs {
			if sq.Type == models.SavedQueryTypeNotifications {
				continue
			}
			matched, err := threadMatchesQuery(sq.Query, thread, te, externalEntity)
			if err != nil {
				golog.ContextLogger(ctx).Errorf("Failed to matched thread %s against saved query %s: %s", thread.ID, sq.ID, err)
				continue
			}
			if matched {
				timestamp := thread.LastMessageTimestamp
				if externalEntity {
					timestamp = thread.LastExternalMessageTimestamp
				}
				entityShouldBeNotified[e.ID] = sq.NotificationsEnabled || entityShouldBeNotified[e.ID]
				unread := isUnread(thread, te, externalEntity)
				newItems = append(newItems, &dal.SavedQueryThread{
					ThreadID:     thread.ID,
					SavedQueryID: sq.ID,
					Timestamp:    timestamp,
					Unread:       unread,
				})
				if sq.NotificationsEnabled && nsq != nil {
					newItems = append(newItems, &dal.SavedQueryThread{
						ThreadID:     thread.ID,
						SavedQueryID: nsq.ID,
						Timestamp:    timestamp,
						Unread:       unread,
					})
				}
			}
		}
	}
	return newItems, entityShouldBeNotified
}

// entityAndMemberships looks up an entity and returns the entity itself and all its memberships
func (s *threadsServer) entityAndMemberships(ctx context.Context, entityID string, rootTypes []directory.EntityType) ([]*directory.Entity, error) {
	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: rootTypes,
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(res.Entities) != 1 {
		return nil, errors.Errorf("Expected 1 entity for '%s' got %d", entityID, len(res.Entities))
	}
	// For non-internal (patient_ entities we don't want to include anything but the entity itself
	if res.Entities[0].Type != directory.EntityType_INTERNAL {
		return res.Entities, nil
	}
	ents := make([]*directory.Entity, 0, 1+len(res.Entities[0].Memberships))
	ents = append(ents, res.Entities[0])
	for _, e := range res.Entities[0].Memberships {
		ents = append(ents, e)
	}
	return ents, nil
}

// resolveInternalEntities looks up a list of entities and resolves memberships to get a list of all internal entities.
// Given a set of entities that can be internal or organizations, it fetches the members of the orgs to flatten to a list
// of only and all internal entities.
func (s *threadsServer) resolveInternalEntities(ctx context.Context, entityIDs []string) ([]*directory.Entity, error) {
	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: entityIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	es, err := internalEntities(nil, res.Entities)
	return es, errors.Trace(err)
}

// internalEntities recursively descends a list of entities to find all internal entities.
func internalEntities(out, in []*directory.Entity) ([]*directory.Entity, error) {
	for _, e := range in {
		switch e.Type {
		case directory.EntityType_INTERNAL:
			out = append(out, e)
		case directory.EntityType_ORGANIZATION:
			var err error
			out, err = internalEntities(out, e.Members)
			if err != nil {
				return nil, errors.Trace(err)
			}
		default:
			return nil, errors.Errorf("unexpected entity type %s", e.Type)
		}
	}
	return out, nil
}
