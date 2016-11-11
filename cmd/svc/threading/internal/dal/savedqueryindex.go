package dal

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const sqItemBatchSize = 200

func (d *dal) AddItemsToSavedQueryIndex(ctx context.Context, items []*SavedQueryThread) error {
	vals := dbutil.MySQLMultiInsert(sqItemBatchSize)
	for len(items) > 0 {
		vals.Reset()
		n := len(items)
		if n > sqItemBatchSize {
			n = sqItemBatchSize
		}
		for _, it := range items[:n] {
			vals.Append(it.SavedQueryID, it.ThreadID, it.Unread, it.Timestamp)
		}
		items = items[n:]
		for retry := 3; retry > 0; retry-- {
			_, err := d.db.Exec(`
				INSERT INTO saved_query_thread (saved_query_id, thread_id, unread, timestamp)
				VALUES `+vals.Query()+`
				ON DUPLICATE KEY UPDATE unread = VALUES(unread), timestamp = VALUES(timestamp)`,
				vals.Values()...)
			if dbutil.IsMySQLWarning(err, dbutil.MySQLNoRangeOptimization) {
				golog.Errorf("When adding items to saved query got MySQL warning: %s", err)
			} else if dbutil.IsMySQLError(err, dbutil.MySQLDeadlock) {
				if retry == 0 {
					return errors.Trace(err)
				}
				golog.Infof("Deadlock when add items to saved query, retry %d: %s", retry, err)
				time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(20)))
				continue
			} else if err != nil {
				return errors.Trace(err)
			}
			break
		}
	}
	return nil
}

func (d *dal) RemoveItemsFromSavedQueryIndex(ctx context.Context, items []*SavedQueryThread) error {
	vals := dbutil.MySQLMultiInsert(sqItemBatchSize)
	for len(items) > 0 {
		vals.Reset()
		n := len(items)
		if n > sqItemBatchSize {
			n = sqItemBatchSize
		}
		for _, it := range items[:n] {
			vals.Append(it.SavedQueryID, it.ThreadID)
		}
		items = items[n:]
		for retry := 3; retry > 0; retry-- {
			_, err := d.db.Exec(`DELETE FROM saved_query_thread WHERE (saved_query_id, thread_id) IN (`+vals.Query()+`)`, vals.Values()...)
			if dbutil.IsMySQLWarning(err, dbutil.MySQLNoRangeOptimization) {
				golog.Errorf("When removing items from saved query got warning: %s", err)
			} else if dbutil.IsMySQLError(err, dbutil.MySQLDeadlock) {
				if retry == 0 {
					return errors.Trace(err)
				}
				golog.Infof("Deadlock when removing items from saved query, retry %d: %s", retry, err)
				time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(20)))
				continue
			} else if err != nil {
				return errors.Trace(err)
			}
			break
		}
	}
	return nil
}

func (d *dal) RemoveAllItemsFromSavedQueryIndex(ctx context.Context, sqID models.SavedQueryID) error {
	_, err := d.db.Exec(`DELETE FROM saved_query_thread WHERE saved_query_id = ?`, sqID)
	return errors.Trace(err)
}

func (d *dal) RemoveThreadFromAllSavedQueryIndexes(ctx context.Context, threadID models.ThreadID) error {
	_, err := d.db.Exec(`DELETE FROM saved_query_thread WHERE thread_id = ?`, threadID)
	return errors.Trace(err)
}

func (d *dal) IterateThreadsInSavedQuery(ctx context.Context, sqID models.SavedQueryID, viewerEntityID string, it *Iterator) (*ThreadConnection, error) {
	if it.Count > maxThreadCount {
		it.Count = maxThreadCount
	}
	if it.Count <= 0 {
		it.Count = defaultThreadCount
	}

	cond := []string{"saved_query_id = ?"}
	vals := []interface{}{viewerEntityID, sqID}

	// Build query based on iterator in descending order so start = later and end = earlier.
	if it.StartCursor != "" {
		cond = append(cond, `timestamp < ?`)
		v, err := parseTimeCursor(it.StartCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad start cursor: " + it.StartCursor))
		}
		vals = append(vals, v)
	}
	if it.EndCursor != "" {
		cond = append(cond, `timestamp > ?`)
		v, err := parseTimeCursor(it.EndCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad end cursor: " + it.EndCursor))
		}
		vals = append(vals, v)
	}
	where := strings.Join(cond, " AND ")
	order := ` ORDER BY timestamp`
	if it.Direction == FromStart {
		order += " DESC"
	}
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to see if there's more threads than we need to set the "HasMore" flag
	queryStr := `
		SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), t.last_message_timestamp, t.last_external_message_timestamp, t.last_message_summary,
			t.last_external_message_summary, t.last_primary_entity_endpoints, t.created, t.message_count, t.type, COALESCE(t.system_title, ''), COALESCE(t.user_title, ''), t.origin, t.deleted,
			viewer.thread_id, viewer.entity_id, viewer.member, viewer.following, viewer.joined, viewer.last_viewed, viewer.last_unread_notify, viewer.last_referenced,
			(SELECT GROUP_CONCAT(tag SEPARATOR ' ') FROM thread_tags INNER JOIN tags ON tags.id = tag_id WHERE thread_tags.thread_id = t.id)
		FROM saved_query_thread sqt
		INNER JOIN threads t ON t.id = sqt.thread_id
		LEFT OUTER JOIN thread_entities viewer ON viewer.thread_id = t.id AND viewer.entity_id = ?
		WHERE ` + where + order + limit
	rows, err := d.db.Query(queryStr, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadConnection
	for rows.Next() {
		t, te, err := scanThreadAndEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		// TODO: technically the cursor should be the saved_query_thead.timestamp but need to update the scan function to fetch that.. functionally the same for now though
		cursor := formatTimeCursor(t.LastMessageTimestamp)
		tc.Edges = append(tc.Edges, ThreadEdge{
			Thread:       t,
			ThreadEntity: te,
			Cursor:       cursor,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(rows.Err())
	}

	// If we got more than was asked then we know there's more to be had
	if len(tc.Edges) > it.Count {
		tc.HasMore = true
		tc.Edges = tc.Edges[:it.Count]
	}

	// Always return in descending order so reverse if we were asked to query FromEnd
	if it.Direction == FromEnd {
		for i := 0; i < len(tc.Edges)/2; i++ {
			j := len(tc.Edges) - i - 1
			tc.Edges[i], tc.Edges[j] = tc.Edges[j], tc.Edges[i]
		}
	}

	return &tc, nil
}

func (d *dal) RebuildNotificationsSavedQuery(ctx context.Context, entityID string) error {
	sqs, err := d.SavedQueries(ctx, entityID)
	if err != nil {
		return errors.Trace(err)
	}

	var nsq *models.SavedQuery
	notifySQIDs := make([]interface{}, 0, len(sqs))
	for _, sq := range sqs {
		if sq.Type == models.SavedQueryTypeNotifications {
			nsq = sq
		} else if sq.NotificationsEnabled {
			notifySQIDs = append(notifySQIDs, sq.ID)
		}
	}

	if nsq == nil {
		return nil
	}

	if err := d.RemoveAllItemsFromSavedQueryIndex(ctx, nsq.ID); err != nil {
		return errors.Trace(err)
	}
	if len(notifySQIDs) == 0 {
		return nil
	}
	_, err = d.db.Exec(`
		INSERT INTO saved_query_thread (saved_query_id, thread_id, unread, timestamp)
		SELECT ?, thread_id, unread, timestamp
		FROM saved_query_thread sqt
		WHERE saved_query_id IN (`+dbutil.MySQLArgs(len(notifySQIDs))+`)
		ON DUPLICATE KEY UPDATE saved_query_thread.unread = sqt.unread, saved_query_thread.timestamp = sqt.timestamp`,
		append([]interface{}{nsq.ID}, notifySQIDs...)...)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) UnreadNotificationsCounts(ctx context.Context, entityIDs []string) (map[string]int, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}
	vals := make([]interface{}, len(entityIDs)+1)
	vals[0] = models.SavedQueryTypeNotifications
	for i, id := range entityIDs {
		vals[i+1] = id
	}
	rows, err := d.db.Query(`SELECT entity_id, unread FROM saved_queries WHERE type = ? AND entity_id IN (`+dbutil.MySQLArgs(len(entityIDs))+`)`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	counts := make(map[string]int, len(entityIDs))
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, errors.Trace(err)
		}
		counts[id] = count
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	// Entities that don't have notification saved queries are likely patients so look for threads with matching primary entity
	eids := make([]interface{}, 0, len(entityIDs)-len(counts))
	for _, id := range entityIDs {
		if _, ok := counts[id]; !ok {
			eids = append(eids, id)
		}
	}
	if len(eids) != 0 {
		rows, err := d.db.Query(`SELECT id, primary_entity_id FROM threads WHERE primary_entity_id IN (`+dbutil.MySQLArgs(len(eids))+`)`, eids...)
		if err != nil {
			return nil, errors.Trace(err)
		}
		defer rows.Close()
		for rows.Next() {
			threadID := models.EmptyThreadID()
			var entityID string
			if err := rows.Scan(&threadID, &entityID); err != nil {
				return nil, errors.Trace(err)
			}
			// TODO: assuming anyone who's a primary entity is external
			n, err := d.UnreadMessagesInThread(ctx, threadID, entityID, true)
			if err != nil {
				return nil, errors.Trace(err)
			}
			counts[entityID] += n
		}
		if err := rows.Err(); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return counts, nil
}
