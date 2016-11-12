package dal

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"github.com/sprucehealth/backend/svc/threading"
)

const (
	threadColumns = `
        t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), t.last_message_timestamp, t.last_external_message_timestamp,
        t.last_message_summary, t.last_external_message_summary, t.last_primary_entity_endpoints, t.created, t.message_count,
        t.type, COALESCE(t.system_title, ''), COALESCE(t.user_title, ''), t.origin, t.deleted,
        (SELECT GROUP_CONCAT(tag SEPARATOR ' ') FROM thread_tags INNER JOIN tags ON tags.id = tag_id WHERE thread_tags.thread_id = t.id)
	`
	threadItemColumns = `ti.id, ti.thread_id, ti.created, ti.modified, ti.actor_entity_id, ti.internal, ti.type, ti.data, ti.deleted`
)

type QueryOption int

const (
	ForUpdate QueryOption = iota + 1
)

type queryOptions []QueryOption

func (qos queryOptions) Has(opt QueryOption) bool {
	for _, o := range qos {
		if o == opt {
			return true
		}
	}
	return false
}

var ErrNotFound = errors.New("threading/dal: object not found")

type ErrInvalidIterator string

func (e ErrInvalidIterator) Error() string {
	return fmt.Sprintf("threading/dal: invalid iterator: %s", string(e))
}

const (
	maxThreadCount         = 5000
	defaultThreadCount     = 20
	maxThreadItemCount     = 1000
	defaultThreadItemCount = 20
)

type Direction int

const (
	FromStart Direction = iota
	FromEnd
)

type Iterator struct {
	StartCursor string
	EndCursor   string
	Direction   Direction
	Count       int
}

type ThreadEdge struct {
	Thread       *models.Thread
	ThreadEntity *models.ThreadEntity
	Cursor       string
}

type ThreadConnection struct {
	Edges   []ThreadEdge
	HasMore bool
}

type ThreadItemEdge struct {
	Item   *models.ThreadItem
	Cursor string
}

type ThreadItemConnection struct {
	Edges   []ThreadItemEdge
	HasMore bool
}

type ThreadLink struct {
	ThreadID      models.ThreadID
	PrependSender bool
}

type ThreadUpdate struct {
	SystemTitle *string
	UserTitle   *string
}

type PostMessageRequest struct {
	ThreadID     models.ThreadID
	FromEntityID string
	Internal     bool
	Title        string
	Text         string
	TextRefs     []*models.Reference
	Attachments  []*models.Attachment
	Source       *models.Endpoint
	Destinations []*models.Endpoint
	Summary      string
}

type ThreadEntityUpdate struct {
	LastViewed       *time.Time
	LastUnreadNotify *time.Time
	LastReferenced   *time.Time
}

type SetupThreadStateUpdate struct {
	Step *int
}

type SavedQueryThread struct {
	ThreadID     models.ThreadID
	SavedQueryID models.SavedQueryID
	Timestamp    time.Time
	Unread       bool
}

type SavedQueryUpdate struct {
	Query                *models.Query
	Title                *string
	Ordinal              *int
	NotificationsEnabled *bool
}

type DAL interface {
	Transact(context.Context, func(context.Context, DAL) error) error

	AddThreadFollowers(ctx context.Context, threadID models.ThreadID, followerEntityIDs []string) error
	AddThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error
	AddThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error
	CreateSavedQuery(context.Context, *models.SavedQuery) (models.SavedQueryID, error)
	CreateSetupThreadState(ctx context.Context, threadID models.ThreadID, entityID string) error
	CreateThread(ctx context.Context, t *models.Thread) (models.ThreadID, error)
	CreateThreadItem(ctx context.Context, item *models.ThreadItem) error
	CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error
	CreateThreadLink(ctx context.Context, thread1Link, thread2Link *ThreadLink) error
	DeleteSavedQueries(ctx context.Context, ids []models.SavedQueryID) error
	DeleteThread(ctx context.Context, threadID models.ThreadID) error
	// DeleteMessage deletes a thread item that is a message and returns true iff the item wasn't already deleted
	DeleteMessage(ctx context.Context, threadItemID models.ThreadItemID) (*models.ThreadItem, bool, error)
	EntitiesForThread(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadEntity, error)
	IterateThreads(ctx context.Context, query *models.Query, memberEntityIDs []string, viewerEntityID string, forExternal bool, it *Iterator) (*ThreadConnection, error)
	IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *Iterator) (*ThreadItemConnection, error)
	LinkedThread(ctx context.Context, threadID models.ThreadID) (*models.Thread, bool, error)
	PostMessage(context.Context, *PostMessageRequest) (*models.ThreadItem, error)
	RecordThreadEvent(ctx context.Context, threadID models.ThreadID, actorEntityID string, event models.ThreadEvent) error
	RemoveThreadFollowers(ctx context.Context, threadID models.ThreadID, followerEntityIDs []string) error
	RemoveThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error
	RemoveThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error
	SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error)
	SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error)
	SavedQueryTemplates(ctx context.Context, entityID string) ([]*models.SavedQuery, error)
	SetupThreadState(ctx context.Context, threadID models.ThreadID, opts ...QueryOption) (*models.SetupThreadState, error)
	SetupThreadStateForEntity(ctx context.Context, entityID string, opts ...QueryOption) (*models.SetupThreadState, error)
	TagsForOrg(ctx context.Context, orgID, prefix string) ([]models.Tag, error)
	Threads(ctx context.Context, ids []models.ThreadID, opts ...QueryOption) ([]*models.Thread, error)
	ThreadItem(ctx context.Context, id models.ThreadItemID, opts ...QueryOption) (*models.ThreadItem, error)
	ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error)
	ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error)
	ThreadEntities(ctx context.Context, threadIDs []models.ThreadID, entityID string, opts ...QueryOption) (map[string]*models.ThreadEntity, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error)
	ThreadsForOrg(ctx context.Context, organizationID string, typ models.ThreadType, limit int) ([]*models.Thread, error)
	ThreadsWithEntity(ctx context.Context, entityID string, ids []models.ThreadID) ([]*models.Thread, []*models.ThreadEntity, error)
	UpdateMessage(ctx context.Context, threadID models.ThreadID, itemID models.ThreadItemID, req *PostMessageRequest) error
	// UnreadMessagesInThread returns the number of unread messages in a thread for an entity.
	UnreadMessagesInThread(ctx context.Context, threadID models.ThreadID, entityID string, external bool) (int, error)
	UpdateSavedQuery(context.Context, models.SavedQueryID, *SavedQueryUpdate) error
	UpdateSetupThreadState(context.Context, models.ThreadID, *SetupThreadStateUpdate) error
	UpdateThread(ctx context.Context, threadID models.ThreadID, update *ThreadUpdate) error
	// UpdateThreadEntity updates attributes about a thread entity. If the thread entity relationship doesn't exist then it is created.
	UpdateThreadEntity(ctx context.Context, threadID models.ThreadID, entityID string, update *ThreadEntityUpdate) error

	// Saved Messages

	// CreateSavedMessage creates a saved messages setting the ID in the provided model
	CreateSavedMessage(ctx context.Context, sm *models.SavedMessage) (models.SavedMessageID, error)
	// DeleteSavedMessages deletes a set of saved messages by ID
	DeleteSavedMessages(ctx context.Context, ids []models.SavedMessageID) (int, error)
	// SavedMessages returns a set of saved messages by ID
	SavedMessages(ctx context.Context, ids []models.SavedMessageID) ([]*models.SavedMessage, error)
	// SavedMessagesForEntities returns all saved messages owned by a set of entities
	SavedMessagesForEntities(ctx context.Context, ownerEntityIDs []string) ([]*models.SavedMessage, error)
	// UpdateSavedMessage updates an existing saved message
	UpdateSavedMessage(ctx context.Context, id models.SavedMessageID, update *SavedMessageUpdate) error

	// Saved Query Indexes

	AddItemsToSavedQueryIndex(ctx context.Context, items []*SavedQueryThread) error
	IterateThreadsInSavedQuery(ctx context.Context, sqID models.SavedQueryID, viewerEntityID string, it *Iterator) (*ThreadConnection, error)
	RemoveAllItemsFromSavedQueryIndex(ctx context.Context, sqID models.SavedQueryID) error
	RemoveItemsFromSavedQueryIndex(ctx context.Context, items []*SavedQueryThread) error
	// RemoveThreadFromAllSavedQueryIndexes clears a saved query index of all threads
	RemoveThreadFromAllSavedQueryIndexes(ctx context.Context, threadID models.ThreadID) error
	// RebuildNotificationsSavedQuery recreates the notifications saved query from all saved queries from the entity marked for notifications
	RebuildNotificationsSavedQuery(ctx context.Context, entityID string) error
	// UnreadNotificationsCounts returns the number of unread notifications for a set of entities
	UnreadNotificationsCounts(ctx context.Context, entityIDs []string) (map[string]int, error)

	// Scheduled Messages
	CreateScheduledMessage(ctx context.Context, model *models.ScheduledMessage) (models.ScheduledMessageID, error)
	DeleteScheduledMessage(ctx context.Context, id models.ScheduledMessageID) (int64, error)
	ScheduledMessage(ctx context.Context, id models.ScheduledMessageID, opts ...QueryOption) (*models.ScheduledMessage, error)
	ScheduledMessages(ctx context.Context, status []models.ScheduledMessageStatus, scheduledForBefore time.Time, opts ...QueryOption) ([]*models.ScheduledMessage, error)
	ScheduledMessagesForThread(ctx context.Context, threadID models.ThreadID, status []models.ScheduledMessageStatus, opts ...QueryOption) ([]*models.ScheduledMessage, error)
	UpdateScheduledMessage(ctx context.Context, id models.ScheduledMessageID, update *models.ScheduledMessageUpdate) (int64, error)

	// Triggered Messages
	CreateTriggeredMessage(ctx context.Context, model *models.TriggeredMessage) (models.TriggeredMessageID, error)
	CreateTriggeredMessages(ctx context.Context, models []*models.TriggeredMessage) error
	TriggeredMessage(ctx context.Context, id models.TriggeredMessageID, opts ...QueryOption) (*models.TriggeredMessage, error)
	TriggeredMessageForKeys(ctx context.Context, triggerKey string, triggerSubkey string, opts ...QueryOption) (*models.TriggeredMessage, error)
	DeleteTriggeredMessage(ctx context.Context, id models.TriggeredMessageID) (int64, error)
	UpdateTriggeredMessage(ctx context.Context, id models.TriggeredMessageID, update *models.TriggeredMessageUpdate) (int64, error)

	// Triggered Message Items
	CreateTriggeredMessageItem(ctx context.Context, model *models.TriggeredMessageItem) (models.TriggeredMessageItemID, error)
	CreateTriggeredMessageItems(ctx context.Context, models []*models.TriggeredMessageItem) error
	TriggeredMessageItem(ctx context.Context, id models.TriggeredMessageItemID, opts ...QueryOption) (*models.TriggeredMessageItem, error)
	TriggeredMessageItemsForTriggeredMessageID(ctx context.Context, triggeredMessageID models.TriggeredMessageID, opts ...QueryOption) ([]*models.TriggeredMessageItem, error)
	DeleteTriggeredMessageItem(ctx context.Context, id models.TriggeredMessageItemID) (int64, error)
	DeleteTriggeredMessageItemsForTriggeredMessage(ctx context.Context, id models.TriggeredMessageID) (int64, error)
}

// New returns an initialized instance of dal
func New(db *sql.DB, clk clock.Clock) DAL {
	return &dal{
		db:  tsql.AsDB(db),
		clk: clk,
	}
}

type dal struct {
	db  tsql.DB
	clk clock.Clock
}

// Transact encapsulates the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(ctx context.Context, trans func(context.Context, DAL) error) (err error) {
	return d.transact(ctx, func(ctx context.Context, dl *dal) error {
		return trans(ctx, dl)
	})
}

func (d *dal) transact(ctx context.Context, trans func(context.Context, *dal) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db:  tsql.AsSafeTx(tx),
		clk: d.clk,
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()

			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]

			if err := tx.Rollback(); err != nil {
				golog.Errorf("Rollback failed: %s", err)
			}
			err = errors.Errorf("Encountered panic during transaction execution: %v", r)
			golog.Errorf("%s - Stack trace: %s", err, string(buf))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

func (d *dal) CreateSavedQuery(ctx context.Context, sq *models.SavedQuery) (models.SavedQueryID, error) {
	if err := sq.Type.Validate(); err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
	id, err := models.NewSavedQueryID()
	if err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
	queryBlob, err := sq.Query.Marshal()
	if err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
	_, err = d.db.Exec(`
		INSERT INTO saved_queries (id, ordinal, entity_id, query, title, unread, total, notifications_enabled, type, hidden, template)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, sq.Ordinal, sq.EntityID, queryBlob, sq.Title, sq.Unread, sq.Total, sq.NotificationsEnabled, sq.Type, sq.Hidden, sq.Template)
	if err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
	sq.ID = id
	return id, nil
}

func (d *dal) CreateThread(ctx context.Context, thread *models.Thread) (models.ThreadID, error) {
	if thread.Type == "" {
		return models.ThreadID{}, errors.Trace(errors.New("thread type required"))
	}
	id, err := models.NewThreadID()
	if err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}
	lastPrimaryEntityEndpointsData, err := thread.LastPrimaryEntityEndpoints.Marshal()
	if err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}

	now := d.clk.Now()
	if thread.Created.IsZero() {
		thread.Created = now
	}
	if thread.LastMessageTimestamp.IsZero() {
		thread.LastMessageTimestamp = now
	}
	if thread.LastExternalMessageTimestamp.IsZero() {
		thread.LastExternalMessageTimestamp = now
	}

	_, err = d.db.Exec(`
		INSERT INTO threads (
			id, organization_id, primary_entity_id, last_message_timestamp, last_external_message_timestamp, last_message_summary,
			last_external_message_summary, last_primary_entity_endpoints, type,
			system_title, user_title, origin, created)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, thread.OrganizationID, thread.PrimaryEntityID, thread.LastMessageTimestamp, thread.LastExternalMessageTimestamp,
		thread.LastMessageSummary, thread.LastExternalMessageSummary, lastPrimaryEntityEndpointsData, thread.Type,
		thread.SystemTitle, thread.UserTitle, thread.Origin, thread.Created)
	if err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}
	thread.ID = id
	return id, nil
}

func (d *dal) CreateThreadItem(ctx context.Context, item *models.ThreadItem) error {
	return errors.Trace(d.createThreadItem(ctx, d.db, item))
}

func (d *dal) createThreadItem(ctx context.Context, db tsql.DB, item *models.ThreadItem) error {
	if !item.ID.IsValid {
		id, err := models.NewThreadItemID()
		if err != nil {
			return errors.Trace(err)
		}
		item.ID = id
	}
	if item.Created.IsZero() {
		item.Created = time.Now()
	}
	data, err := item.Data.Marshal()
	if err != nil {
		return errors.Trace(err)
	}
	itemType, err := models.ItemTypeForValue(item.Data)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = db.Exec(`
		INSERT INTO thread_items (id, thread_id, created, actor_entity_id, internal, type, data, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.ThreadID, item.Created, item.ActorEntityID, item.Internal, itemType, data, item.Deleted)
	return errors.Trace(err)
}

func (d *dal) CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error {
	if len(tds) == 0 {
		return nil
	}
	ins := dbutil.MySQLMultiInsert(len(tds))
	for _, td := range tds {
		ins.Append(td.ThreadItemID, td.ActorEntityID, td.ViewTime)
	}
	_, err := d.db.Exec(`
        INSERT IGNORE INTO thread_item_view_details
	        (thread_item_id, actor_entity_id, view_time)
        VALUES `+ins.Query(), ins.Values()...)
	if dbutil.IsMySQLWarning(err, dbutil.MySQLDuplicateEntry) {
		return nil
	}
	return errors.Trace(err)
}

func (d *dal) CreateThreadLink(ctx context.Context, thread1ID, thread2ID *ThreadLink) error {
	// Sanity check since self-reference is too scary to imagine
	if thread1ID.ThreadID.Val == thread2ID.ThreadID.Val {
		return errors.Trace(errors.New("cannot link a thread to itself"))
	}
	_, err := d.db.Exec(`INSERT INTO thread_links (thread1_id, thread1_prepend_sender, thread2_id, thread2_prepend_sender) VALUES(?, ?, ?, ?)`,
		thread1ID.ThreadID, thread1ID.PrependSender, thread2ID.ThreadID, thread2ID.PrependSender)
	return errors.Trace(err)
}

func (d *dal) DeleteThread(ctx context.Context, threadID models.ThreadID) error {
	_, err := d.db.Exec(`UPDATE threads SET deleted = true WHERE id = ?`, threadID)
	return errors.Trace(err)
}

func (d *dal) DeleteMessage(ctx context.Context, threadItemID models.ThreadItemID) (*models.ThreadItem, bool, error) {
	var deleted bool
	var item *models.ThreadItem
	err := d.transact(ctx, func(ctx context.Context, d *dal) error {
		// Fetch the item to make sure it's a message and not already deleted.
		var err error
		item, err = d.ThreadItem(ctx, threadItemID, ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		if _, ok := item.Data.(*models.Message); !ok {
			return errors.Errorf("Can only delete messages, not items of type %T for id %s", item.Data, item.ID)
		}
		if item.Deleted {
			return nil
		}
		item.Deleted = true
		deleted = true

		// Flag the item as deleted but don't modify content to be safe
		if _, err := d.db.Exec(`UPDATE thread_items SET deleted = true WHERE id = ?`, threadItemID); err != nil {
			return errors.Trace(err)
		}

		return errors.Trace(updateThreadLastMessageInfo(ctx, d.db, item.ThreadID, true))
	})
	return item, deleted, errors.Trace(err)
}

// updateThreadLastMessageInfo updates the denormalized fields on a thread from the last message.
// it should be used inside a transaction.
func updateThreadLastMessageInfo(ctx context.Context, tx tsql.DB, threadID models.ThreadID, isDelete bool) error {
	// Lock thread for update
	var threadCreated time.Time
	if err := tx.QueryRow(`SELECT created FROM threads WHERE id = ? FOR UPDATE`, threadID).Scan(&threadCreated); err != nil {
		return errors.Trace(err)
	}

	// Update denormalized fields on thread. Need to fetch last message and last external message.
	lastMessage, err := scanThreadItem(tx.QueryRow(`
		SELECT `+threadItemColumns+`
		FROM thread_items ti
		WHERE thread_id = ? AND type = ? AND deleted = false
		ORDER BY created DESC
		LIMIT 1`, threadID, models.ItemTypeMessage))
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Trace(err)
	}
	lastExternalMessage, err := scanThreadItem(tx.QueryRow(`
		SELECT `+threadItemColumns+`
		FROM thread_items ti
		WHERE thread_id = ? AND type = ? AND deleted = false AND internal = false
		ORDER BY created DESC
		LIMIT 1`, threadID, models.ItemTypeMessage))
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Trace(err)
	}
	lastMessageTimestamp := threadCreated
	lastExternalMessageTimestamp := threadCreated
	var lastMessageSummary string
	var lastExternalMessageSummary string
	if lastMessage != nil {
		lastMessageTimestamp = lastMessage.Created
		lastMessageSummary = lastMessage.Data.(*models.Message).Summary
	}
	if lastExternalMessage != nil {
		lastExternalMessageTimestamp = lastExternalMessage.Created
		lastExternalMessageSummary = lastExternalMessage.Data.(*models.Message).Summary
	}
	var messageCountUpdate string
	if isDelete {
		messageCountUpdate = `message_count = message_count - 1,`
	}
	_, err = tx.Exec(`
		UPDATE threads
		SET `+messageCountUpdate+`
			last_message_summary = ?,
			last_message_timestamp = ?,
			last_external_message_summary = ?,
			last_external_message_timestamp = ?
		WHERE id = ?`,
		lastMessageSummary, lastMessageTimestamp,
		lastExternalMessageSummary, lastExternalMessageTimestamp,
		threadID,
	)
	return errors.Trace(err)
}

func (d *dal) IterateThreads(ctx context.Context, query *models.Query, memberEntityIDs []string, viewerEntityID string, forExternal bool, it *Iterator) (*ThreadConnection, error) {
	if len(memberEntityIDs) == 0 {
		return nil, errors.Errorf("memberEntityIDs missing")
	}
	if viewerEntityID == "" {
		return nil, errors.Errorf("viewerEntityID missing")
	}

	if it.Count > maxThreadCount {
		it.Count = maxThreadCount
	}
	if it.Count <= 0 {
		it.Count = defaultThreadCount
	}
	orderField := "last_message_timestamp"
	if forExternal {
		orderField = "last_external_message_timestamp"
	}
	var cond []string
	vals := dbutil.AppendStringsToInterfaceSlice(nil, memberEntityIDs)
	vals = append(vals, viewerEntityID)

	cond = append(cond, "t.deleted = ?")
	vals = append(vals, false)

	var tags []string

	if query != nil {
		for _, e := range query.Expressions {
			switch v := e.Value.(type) {
			case *models.Expr_Flag_:
				switch v.Flag {
				case models.EXPR_FLAG_UNREAD:
					col := "last_message_timestamp"
					if forExternal {
						col = "last_external_message_timestamp"
					}
					if e.Not {
						cond = append(cond, "(viewer.last_viewed IS NOT NULL AND viewer.last_viewed >= t."+col+")")
					} else {
						cond = append(cond, "(viewer.last_viewed IS NULL OR viewer.last_viewed < t."+col+")")
					}
				case models.EXPR_FLAG_UNREAD_REFERENCE:
					if e.Not {
						cond = append(cond, "(viewer.last_referenced IS NULL OR (viewer.last_viewed IS NOT NULL AND viewer.last_viewed >= viewer.last_referenced))")
					} else {
						cond = append(cond, "(viewer.last_referenced IS NOT NULL AND (viewer.last_viewed IS NULL OR viewer.last_viewed < viewer.last_referenced))")
					}
				case models.EXPR_FLAG_FOLLOWING:
					if e.Not {
						cond = append(cond, "(viewer.following IS NULL AND viewer.following = false)")
					} else {
						cond = append(cond, "(viewer.following IS NOT NULL AND viewer.following = true)")
					}
				default:
					return nil, errors.Errorf("unknown expression flag %s", v.Flag)
				}
			case *models.Expr_ThreadType_:
				switch v.ThreadType {
				case models.EXPR_THREAD_TYPE_PATIENT:
					if e.Not {
						cond = append(cond, "(t.type != ? AND t.type != ?)")
					} else {
						cond = append(cond, "(t.type = ? OR t.type = ?)")
					}
					vals = append(vals, models.ThreadTypeExternal, models.ThreadTypeSecureExternal)
				case models.EXPR_THREAD_TYPE_PATIENT_SECURE:
					if e.Not {
						cond = append(cond, "t.type != ?")
					} else {
						cond = append(cond, "t.type = ?")
					}
					vals = append(vals, models.ThreadTypeSecureExternal)
				case models.EXPR_THREAD_TYPE_PATIENT_STANDARD:
					if e.Not {
						cond = append(cond, "t.type != ?")
					} else {
						cond = append(cond, "t.type = ?")
					}
					vals = append(vals, models.ThreadTypeExternal)
				case models.EXPR_THREAD_TYPE_TEAM:
					if e.Not {
						cond = append(cond, "t.type != ?")
					} else {
						cond = append(cond, "t.type = ?")
					}
					vals = append(vals, models.ThreadTypeTeam)
				case models.EXPR_THREAD_TYPE_SUPPORT:
					if e.Not {
						cond = append(cond, "(t.type != ? AND t.type != ?)")
					} else {
						cond = append(cond, "(t.type = ? OR t.type = ?)")
					}
					vals = append(vals, models.ThreadTypeSupport, models.ThreadTypeSetup)

				default:
					return nil, errors.Errorf("unknown expression thread type %s", v.ThreadType)
				}
			case *models.Expr_Tag:
				tags = append(tags, v.Tag)
			case *models.Expr_Token:
				col := "t.last_message_summary"
				if forExternal {
					col = "t.last_external_message_summary"
				}
				if e.Not {
					cond = append(cond, `NOT (COALESCE(t.system_title, '') LIKE ? OR COALESCE(t.user_title, '') LIKE ? OR `+col+` LIKE ?)`)
				} else {
					cond = append(cond, `(COALESCE(t.system_title, '') LIKE ? OR COALESCE(t.user_title, '') LIKE ? OR `+col+` LIKE ?)`)
				}
				match := "%" + v.Token + "%"
				vals = append(vals, match, match, match)
			default:
				return nil, errors.Errorf("unknown expression type %T", e.Value)
			}
		}
	}
	if len(tags) != 0 {
		cond = append(cond, `(
			SELECT COUNT(1)
			FROM thread_tags tt
			INNER JOIN tags ON tags.id = tt.tag_id AND tags.tag IN (`+dbutil.MySQLArgs(len(tags))+`)
			WHERE tt.thread_id = t.id
		) = ?`)
		for _, t := range tags {
			vals = append(vals, t)
		}
		vals = append(vals, len(tags))
	}

	// Build query based on iterator in descending order so start = later and end = earlier.
	if it.StartCursor != "" {
		cond = append(cond, "("+dbutil.EscapeMySQLName(orderField)+" < ?)")
		v, err := parseTimeCursor(it.StartCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad start cursor: " + it.StartCursor))
		}
		vals = append(vals, v)
	}
	if it.EndCursor != "" {
		cond = append(cond, "("+dbutil.EscapeMySQLName(orderField)+" > ?)")
		v, err := parseTimeCursor(it.EndCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad end cursor: " + it.EndCursor))
		}
		vals = append(vals, v)
	}
	where := ""
	if len(cond) != 0 {
		where = strings.Join(cond, " AND ")
	}
	order := " ORDER BY " + dbutil.EscapeMySQLName(orderField)
	if it.Direction == FromStart {
		order += " DESC"
	}
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to see if there's more threads than we need to set the "HasMore" flag
	queryStr := `
		SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), t.last_message_timestamp, t.last_external_message_timestamp, t.last_message_summary,
			t.last_external_message_summary, t.last_primary_entity_endpoints, t.created, t.message_count, t.type, COALESCE(t.system_title, ''), COALESCE(t.user_title, ''), t.origin, t.deleted,
			viewer.thread_id, viewer.entity_id, viewer.member, viewer.following, viewer.joined, viewer.last_viewed, viewer.last_unread_notify, viewer.last_referenced,
			(SELECT GROUP_CONCAT(tag SEPARATOR ' ') FROM thread_tags INNER JOIN tags ON tags.id = tag_id WHERE thread_tags.thread_id = t.id)
		FROM threads t
		INNER JOIN thread_entities te ON te.thread_id = t.id AND te.member = true AND te.entity_id IN (` + dbutil.MySQLArgs(len(memberEntityIDs)) + `)
		LEFT OUTER JOIN thread_entities viewer ON viewer.thread_id = t.id AND viewer.entity_id = ?
		WHERE ` + where + order + limit
	rows, err := d.db.Query(queryStr, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadConnection
	seen := make(map[uint64]struct{}) // track which IDs have been seen to remove duplicates
	var nThreads int
	for rows.Next() {
		t, te, err := scanThreadAndEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		nThreads++
		if _, ok := seen[t.ID.Val]; !ok {
			seen[t.ID.Val] = struct{}{}
			var cursor string
			if forExternal {
				cursor = formatTimeCursor(t.LastExternalMessageTimestamp)
			} else {
				cursor = formatTimeCursor(t.LastMessageTimestamp)
			}
			tc.Edges = append(tc.Edges, ThreadEdge{
				Thread:       t,
				ThreadEntity: te,
				Cursor:       cursor,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(rows.Err())
	}

	// If we got more than was asked then we know there's more to be had
	if nThreads > it.Count {
		tc.HasMore = true
	}
	if len(tc.Edges) > it.Count {
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

func (d *dal) IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *Iterator) (*ThreadItemConnection, error) {
	if it.Count > maxThreadItemCount {
		it.Count = maxThreadItemCount
	}
	if it.Count <= 0 {
		it.Count = defaultThreadItemCount
	}
	cond := []string{"thread_id = ?"}
	vals := []interface{}{threadID}
	if it.StartCursor != "" {
		cond = append(cond, "(id > ?)")
		v, err := models.ParseThreadItemID(it.StartCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad start cursor: " + it.StartCursor))
		}
		vals = append(vals, v)
	}
	if it.EndCursor != "" {
		cond = append(cond, "(id < ?)")
		v, err := models.ParseThreadItemID(it.EndCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad end cursor: " + it.EndCursor))
		}
		vals = append(vals, v)
	}
	if forExternal {
		cond = append(cond, "internal = false")
	}
	where := strings.Join(cond, " AND ")
	order := " ORDER BY id"
	if it.Direction == FromEnd {
		order += " DESC"
	}
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to check if there's more than requested available.. will filter it out later
	query := `
		SELECT ` + threadItemColumns + `
		FROM thread_items ti
		WHERE ` + where + order + limit
	rows, err := d.db.Query(query, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadItemConnection
	for rows.Next() {
		it, err := scanThreadItem(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tc.Edges = append(tc.Edges, ThreadItemEdge{
			Item:   it,
			Cursor: it.ID.String(),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	// If we got more than was asked then we know there's more to be had
	if len(tc.Edges) > it.Count {
		tc.Edges = tc.Edges[:it.Count]
		tc.HasMore = true
	}

	// Always return in ascending order so reverse if we were asked to query FromEnd
	if it.Direction == FromEnd {
		for i := 0; i < len(tc.Edges)/2; i++ {
			j := len(tc.Edges) - i - 1
			tc.Edges[i], tc.Edges[j] = tc.Edges[j], tc.Edges[i]
		}
	}

	return &tc, nil
}

func (d *dal) LinkedThread(ctx context.Context, threadID models.ThreadID) (*models.Thread, bool, error) {
	var thread1 ThreadLink
	var thread2 ThreadLink
	err := d.db.QueryRow(`
		SELECT thread1_id, thread1_prepend_sender, thread2_id, thread2_prepend_sender
		FROM thread_links
		WHERE thread1_id = ? OR thread2_id = ?`, threadID, threadID).Scan(&thread1.ThreadID, &thread1.PrependSender, &thread2.ThreadID, &thread2.PrependSender)
	if err == sql.ErrNoRows {
		return nil, false, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, false, errors.Trace(err)
	}

	var linkedThread *ThreadLink
	if threadID.Val == thread1.ThreadID.Val {
		linkedThread = &thread2
	} else {
		linkedThread = &thread1
	}

	row := d.db.QueryRow(`
		SELECT `+threadColumns+`
		FROM threads t
		WHERE id = ? AND deleted = false`, linkedThread.ThreadID)
	t, err := scanThread(row)
	return t, linkedThread.PrependSender, errors.Trace(err)
}

// ThreadItemFromPostMessageRequest transforms a post request into it's ThreadItem representation
func ThreadItemFromPostMessageRequest(ctx context.Context, req *PostMessageRequest, clk clock.Clock) (*models.ThreadItem, error) {
	id, err := models.NewThreadItemID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	msg := &models.Message{
		Title:        req.Title,
		Text:         req.Text,
		Attachments:  req.Attachments,
		Source:       req.Source,
		Destinations: req.Destinations,
		TextRefs:     req.TextRefs,
		Summary:      req.Summary,
	}
	return &models.ThreadItem{
		ID:            id,
		ThreadID:      req.ThreadID,
		Created:       clk.Now(),
		ActorEntityID: req.FromEntityID,
		Internal:      req.Internal,
		Data:          msg,
		Deleted:       false,
	}, nil
}

func (d *dal) PostMessage(ctx context.Context, req *PostMessageRequest) (*models.ThreadItem, error) {
	// TODO: validate request
	item, err := ThreadItemFromPostMessageRequest(ctx, req, d.clk)
	if err != nil {
		return nil, errors.Trace(err)
	}
	msg := item.Data.(*models.Message)

	tx, err := d.db.Begin()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var deleted bool
	if err := tx.QueryRow(`SELECT deleted FROM threads WHERE id = ? FOR UPDATE`, req.ThreadID).Scan(&deleted); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return nil, errors.Trace(ErrNotFound)
		}
		return nil, errors.Trace(err)
	}
	if deleted {
		tx.Rollback()
		return nil, errors.Trace(ErrNotFound)
	}

	if err := d.createThreadItem(ctx, tx, item); err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}

	// Update the denormalized fields on the threads table
	if item.Internal {
		_, err = tx.Exec(`
			UPDATE threads
			SET
				last_message_timestamp = GREATEST(last_message_timestamp, ?),
				last_message_summary = ?,
				message_count = (message_count + 1)
			WHERE id = ?`, item.Created, msg.Summary, item.ThreadID)
	} else {
		endpointList := models.EndpointList{Endpoints: req.Destinations}
		endpointListData, err := endpointList.Marshal()
		if err != nil {
			tx.Rollback()
			return nil, errors.Trace(err)
		}
		values := []interface{}{item.Created, item.Created, msg.Summary, msg.Summary}
		// Only update the endpoint list on an external thread if it's non empty
		var endpointUpdate string
		if len(req.Destinations) > 0 {
			endpointUpdate = ", last_primary_entity_endpoints = ?"
			values = append(values, endpointListData)
		}
		values = append(values, item.ThreadID)
		_, err = tx.Exec(`
			UPDATE threads
			SET
				last_message_timestamp = GREATEST(last_message_timestamp, ?),
				last_external_message_timestamp = GREATEST(last_external_message_timestamp, ?),
				last_message_summary = ?,
				message_count = (message_count + 1),
				last_external_message_summary = ? `+endpointUpdate+`
			WHERE id = ?`, values...)
	}
	if err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}

	return item, nil
}

func (d *dal) RecordThreadEvent(ctx context.Context, threadID models.ThreadID, actorEntityID string, event models.ThreadEvent) error {
	id, err := idgen.NewID()
	if err != nil {
		return errors.Trace(err)
	}
	_, err = d.db.Exec(`INSERT INTO thread_events (id, thread_id, actor_entity_id, event) VALUES (?, ?, ?, ?)`,
		id, threadID, actorEntityID, string(event))
	return errors.Trace(err)
}

func (d *dal) SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error) {
	row := d.db.QueryRow(`
		SELECT id, ordinal, entity_id, query, title, unread, total, notifications_enabled, type, hidden, template
		FROM saved_queries
		WHERE id = ?`, id)
	sq, err := scanSavedQuery(row)
	return sq, errors.Trace(err)
}

func (d *dal) SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rows, err := d.db.Query(`
		SELECT id, ordinal, entity_id, query, title, unread, total, notifications_enabled, type, hidden, template
		FROM saved_queries
		WHERE entity_id = ?
		ORDER BY ordinal`, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var sqs []*models.SavedQuery
	for rows.Next() {
		sq, err := scanSavedQuery(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		sqs = append(sqs, sq)
	}
	return sqs, errors.Trace(rows.Err())
}

func (d *dal) DeleteSavedQueries(ctx context.Context, ids []models.SavedQueryID) error {
	if len(ids) == 0 {
		return nil
	}

	interfaceSlice := make([]interface{}, len(ids))
	for i, id := range ids {
		interfaceSlice[i] = id
	}

	_, err := d.db.Exec(`DELETE FROM saved_queries WHERE id in (`+dbutil.MySQLArgs(len(ids))+`)`, interfaceSlice...)
	return errors.Trace(err)
}

func (d *dal) SavedQueryTemplates(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rows, err := d.db.Query(`
		SELECT id, ordinal, entity_id, query, title, unread, total, notifications_enabled, type, hidden, template
		FROM saved_queries
		WHERE entity_id = ? AND template = 1
		ORDER BY ordinal`, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var sqs []*models.SavedQuery
	for rows.Next() {
		sq, err := scanSavedQuery(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		sqs = append(sqs, sq)
	}
	return sqs, errors.Trace(rows.Err())
}

func (d *dal) TagsForOrg(ctx context.Context, orgID, prefix string) ([]models.Tag, error) {
	var rows *sql.Rows
	var err error
	if prefix == "" {
		rows, err = d.db.Query(`SELECT hidden, tag FROM tags WHERE organization_id = ? ORDER BY tag`, orgID)
	} else {
		rows, err = d.db.Query(`SELECT hidden, tag FROM tags WHERE organization_id = ? AND tag LIKE ? ORDER BY tag`, orgID, prefix+"%")
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.Hidden, &t.Name); err != nil {
			return nil, errors.Trace(err)
		}
		tags = append(tags, t)
	}
	return tags, errors.Trace(rows.Err())
}

func (d *dal) Threads(ctx context.Context, ids []models.ThreadID, opts ...QueryOption) ([]*models.Thread, error) {
	return d.threads(ctx, d.db, ids, opts...)
}

func (d *dal) threads(ctx context.Context, db tsql.DB, ids []models.ThreadID, opts ...QueryOption) ([]*models.Thread, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var forUpdateQuery string
	if queryOptions(opts).Has(ForUpdate) {
		models.SortThreadID(ids)
		forUpdateQuery = " FOR UPDATE"
	}
	rows, err := db.Query(`
		SELECT `+threadColumns+`
		FROM threads t
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`) AND deleted = false`+forUpdateQuery, models.ThreadIDsToInterfaces(ids)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	threads := make([]*models.Thread, 0, len(ids))
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		threads = append(threads, t)
	}

	return threads, errors.Trace(rows.Err())
}

func (d *dal) ThreadsWithEntity(ctx context.Context, entityID string, ids []models.ThreadID) ([]*models.Thread, []*models.ThreadEntity, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	rows, err := d.db.Query(`
		SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), t.last_message_timestamp, t.last_external_message_timestamp, t.last_message_summary,
			t.last_external_message_summary, t.last_primary_entity_endpoints, t.created, t.message_count, t.type, COALESCE(t.system_title, ''), COALESCE(t.user_title, ''), t.origin, t.deleted,
			te.thread_id, te.entity_id, te.member, te.following, te.joined, te.last_viewed, te.last_unread_notify, te.last_referenced,
			(SELECT GROUP_CONCAT(tag SEPARATOR ' ') FROM thread_tags INNER JOIN tags ON tags.id = tag_id WHERE thread_tags.thread_id = t.id)
		FROM threads t
		LEFT OUTER JOIN thread_entities te ON te.thread_id = t.id AND te.entity_id = ?
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`) AND deleted = false`,
		append([]interface{}{entityID}, models.ThreadIDsToInterfaces(ids)...)...)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer rows.Close()
	threads := make([]*models.Thread, 0, len(ids))
	threadEntities := make([]*models.ThreadEntity, 0, len(ids))
	for rows.Next() {
		t, te, err := scanThreadAndEntity(rows)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		threads = append(threads, t)
		threadEntities = append(threadEntities, te)
	}
	return threads, threadEntities, errors.Trace(rows.Err())
}

func (d *dal) ThreadItem(ctx context.Context, id models.ThreadItemID, opts ...QueryOption) (*models.ThreadItem, error) {
	query := `
		SELECT ` + threadItemColumns + `
		FROM thread_items ti
		WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		query += " FOR UPDATE"
	}
	row := d.db.QueryRow(query, id)
	ti, err := scanThreadItem(row)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return ti, errors.Trace(err)
}

func (d *dal) ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error) {
	rows, err := d.db.Query(`
		SELECT id FROM thread_items
        WHERE thread_id = ?
        AND created > ?
        ORDER BY created ASC`, threadID, after)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ids []models.ThreadItemID
	for rows.Next() {
		id := models.EmptyThreadItemID()
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Trace(err)
		}
		ids = append(ids, id)
	}
	return ids, errors.Trace(rows.Err())
}

func (d *dal) ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error) {
	rows, err := d.db.Query(`
		SELECT thread_item_id, actor_entity_id, view_time
		FROM thread_item_view_details
		WHERE thread_item_id = ?
        ORDER BY view_time ASC`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tds []*models.ThreadItemViewDetails
	for rows.Next() {
		td, err := scanThreadItemViewDetails(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tds = append(tds, td)
	}
	return tds, errors.Trace(rows.Err())
}

func (d *dal) ThreadEntities(ctx context.Context, threadIDs []models.ThreadID, entityID string, opts ...QueryOption) (map[string]*models.ThreadEntity, error) {
	if len(threadIDs) == 0 || entityID == "" {
		return nil, nil
	}

	var sfu string
	if queryOptions(opts).Has(ForUpdate) {
		models.SortThreadID(threadIDs)
		sfu = "ORDER BY thread_id FOR UPDATE"
	}
	values := make([]interface{}, len(threadIDs)+1)
	for i, v := range threadIDs {
		values[i] = v
	}
	values[len(threadIDs)] = entityID
	rows, err := d.db.Query(`
		SELECT thread_id, entity_id, member, following, joined, last_viewed, last_unread_notify, last_referenced
		FROM thread_entities
		WHERE thread_id IN (`+dbutil.MySQLArgs(len(threadIDs))+`)
			AND entity_id = ? `+sfu, values...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	tes := make(map[string]*models.ThreadEntity)
	for rows.Next() {
		te, err := scanThreadEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tes[te.ThreadID.String()] = te
	}
	return tes, errors.Trace(rows.Err())
}

func (d *dal) EntitiesForThread(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadEntity, error) {
	rows, err := d.db.Query(`
		SELECT thread_id, entity_id, member, following, joined, last_viewed, last_unread_notify, last_referenced
		FROM thread_entities
        WHERE thread_id = ?`, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tms []*models.ThreadEntity
	for rows.Next() {
		tm, err := scanThreadEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tms = append(tms, tm)
	}
	return tms, errors.Trace(rows.Err())
}

func (d *dal) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	var rows *sql.Rows
	var err error
	if primaryOnly {
		rows, err = d.db.Query(`
			SELECT `+threadColumns+`
			FROM threads t
			WHERE primary_entity_id = ? AND deleted = false`, entityID)
	} else {
		rows, err = d.db.Query(`
			SELECT `+threadColumns+`
			FROM thread_entities tm
			INNER JOIN threads t ON t.id = tm.thread_id
			WHERE tm.entity_id = ? AND deleted = false`, entityID)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var threads []*models.Thread
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		threads = append(threads, t)
	}
	return threads, errors.Trace(rows.Err())
}

func (d *dal) ThreadsForOrg(ctx context.Context, organizationID string, typ models.ThreadType, limit int) ([]*models.Thread, error) {
	vals := []interface{}{organizationID}
	where := ""
	if typ != "" {
		where = "AND type = ?"
		vals = append(vals, typ)
	}
	rows, err := d.db.Query(`
		SELECT `+threadColumns+`
		FROM threads t
		WHERE organization_id = ? AND deleted = false `+where+`
		LIMIT ?`, append(vals, limit)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var threads []*models.Thread
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		threads = append(threads, t)
	}
	return threads, errors.Trace(rows.Err())
}

func (d *dal) UpdateSavedQuery(ctx context.Context, id models.SavedQueryID, update *SavedQueryUpdate) error {
	if update == nil {
		return nil
	}

	args := dbutil.MySQLVarArgs()
	if update.Query != nil {
		queryBlob, err := update.Query.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		args.Append("query", queryBlob)
	}
	if update.Title != nil {
		args.Append("title", *update.Title)
	}
	if update.Ordinal != nil {
		args.Append("ordinal", *update.Ordinal)
	}
	if update.NotificationsEnabled != nil {
		args.Append("notifications_enabled", *update.NotificationsEnabled)
	}
	if args.IsEmpty() {
		return nil
	}

	_, err := d.db.Exec(`UPDATE saved_queries SET `+args.ColumnsForUpdate()+` WHERE id = ?`,
		append(args.Values(), id)...)
	return errors.Trace(err)
}

func (d *dal) UpdateThread(ctx context.Context, threadID models.ThreadID, update *ThreadUpdate) error {
	if update == nil {
		return nil
	}

	args := dbutil.MySQLVarArgs()
	if update.SystemTitle != nil {
		args.Append("system_title", *update.SystemTitle)
	}
	if update.UserTitle != nil {
		args.Append("user_title", *update.UserTitle)
	}
	if args.IsEmpty() {
		return nil
	}

	_, err := d.db.Exec(`UPDATE threads SET `+args.ColumnsForUpdate()+` WHERE id = ?`,
		append(args.Values(), threadID)...)
	return errors.Trace(err)
}

func (d *dal) UpdateMessage(ctx context.Context, threadID models.ThreadID, itemID models.ThreadItemID, req *PostMessageRequest) error {
	item, err := ThreadItemFromPostMessageRequest(ctx, req, d.clk)
	if err != nil {
		return errors.Trace(err)
	}
	err = d.transact(ctx, func(ctx context.Context, d *dal) error {
		data, err := item.Data.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := d.db.Exec(`UPDATE thread_items SET data = ? WHERE id = ?`, data, itemID); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(updateThreadLastMessageInfo(ctx, d.db, threadID, false))
	})
	return errors.Trace(err)
}

func (d *dal) AddThreadFollowers(ctx context.Context, threadID models.ThreadID, followerEntityIDs []string) error {
	if len(followerEntityIDs) == 0 {
		return nil
	}
	ins := dbutil.MySQLMultiInsert(len(followerEntityIDs))
	for _, id := range followerEntityIDs {
		ins.Append(threadID, id, true)
	}
	_, err := d.db.Exec(`
		INSERT INTO thread_entities (thread_id, entity_id, following)
		VALUES `+ins.Query()+`
		ON DUPLICATE KEY UPDATE following = true`,
		ins.Values()...)
	return errors.Trace(err)
}

func (d *dal) AddThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error {
	if len(memberEntityIDs) == 0 {
		return nil
	}
	ins := dbutil.MySQLMultiInsert(len(memberEntityIDs))
	for _, id := range memberEntityIDs {
		ins.Append(threadID, id, true)
	}
	_, err := d.db.Exec(`
		INSERT INTO thread_entities (thread_id, entity_id, member)
		VALUES `+ins.Query()+`
		ON DUPLICATE KEY UPDATE member = true`,
		ins.Values()...)
	return errors.Trace(err)
}

func (d *dal) RemoveThreadFollowers(ctx context.Context, threadID models.ThreadID, followerEntityIDs []string) error {
	if len(followerEntityIDs) == 0 {
		return nil
	}
	_, err := d.db.Exec(`
		UPDATE thread_entities
		SET following = false
		WHERE entity_id IN (`+dbutil.MySQLArgs(len(followerEntityIDs))+`) AND thread_id = ?`,
		append(dbutil.AppendStringsToInterfaceSlice(nil, followerEntityIDs), threadID)...)
	return errors.Trace(err)
}

func (d *dal) RemoveThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error {
	if len(memberEntityIDs) == 0 {
		return nil
	}
	_, err := d.db.Exec(`
		UPDATE thread_entities
		SET member = false
		WHERE entity_id IN (`+dbutil.MySQLArgs(len(memberEntityIDs))+`) AND thread_id = ?`,
		append(dbutil.AppendStringsToInterfaceSlice(nil, memberEntityIDs), threadID)...)
	return errors.Trace(err)
}

func (d *dal) AddThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// Make sure all tags exist
	ins := dbutil.MySQLMultiInsert(len(tags))
	for _, t := range tags {
		ins.Append(orgID, false, t) // TODO: hidden
	}
	_, err := d.db.Exec(`
		INSERT IGNORE INTO tags (organization_id, hidden, tag)
		VALUES `+ins.Query(), ins.Values()...)
	if err != nil && !dbutil.IsMySQLWarning(err, dbutil.MySQLDuplicateEntry) {
		return errors.Trace(err)
	}

	// Fetch IDs for tags
	rows, err := d.db.Query(`SELECT id FROM tags WHERE tag IN (`+dbutil.MySQLArgs(len(tags))+`) AND organization_id = ?`,
		append(dbutil.AppendStringsToInterfaceSlice(nil, tags), orgID)...)
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()
	ins.Reset()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return errors.Trace(err)
		}
		ins.Append(threadID, id)
	}

	// Add tags to thread
	_, err = d.db.Exec(`INSERT IGNORE INTO thread_tags (thread_id, tag_id) VALUES `+ins.Query(), ins.Values()...)
	if err != nil && !dbutil.IsMySQLWarning(err, dbutil.MySQLDuplicateEntry) {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) RemoveThreadTags(ctx context.Context, orgID string, threadID models.ThreadID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	_, err := d.db.Exec(`
		DELETE tt FROM thread_tags AS tt
		INNER JOIN tags AS t ON t.id = tt.tag_id
		WHERE thread_id = ? AND t.tag IN (`+dbutil.MySQLArgs(len(tags))+`)`,
		dbutil.AppendStringsToInterfaceSlice([]interface{}{threadID}, tags)...)
	return errors.Trace(err)
}

func (d *dal) UnreadMessagesInThread(ctx context.Context, threadID models.ThreadID, entityID string, external bool) (int, error) {
	var lastViewed *time.Time
	row := d.db.QueryRow(`SELECT last_viewed FROM thread_entities WHERE thread_id = ? AND entity_id = ?`, threadID, entityID)
	if err := row.Scan(&lastViewed); err != nil && err != sql.ErrNoRows {
		return 0, errors.Trace(err)
	}
	var andInternal string
	if external {
		andInternal = " AND internal = false"
	}
	if lastViewed == nil {
		row = d.db.QueryRow(`SELECT COUNT(1) FROM thread_items WHERE thread_id = ? AND deleted = false AND type = ?`+andInternal,
			threadID, string(models.ItemTypeMessage))
	} else {
		row = d.db.QueryRow(`SELECT COUNT(1) FROM thread_items WHERE thread_id = ? AND deleted = false AND type = ? AND created > ?`+andInternal,
			threadID, string(models.ItemTypeMessage), *lastViewed)
	}
	var count int
	err := row.Scan(&count)
	return count, errors.Trace(err)
}

func (d *dal) UpdateThreadEntity(ctx context.Context, threadID models.ThreadID, entityID string, update *ThreadEntityUpdate) error {
	var args dbutil.VarArgs

	if update != nil {
		args = dbutil.MySQLVarArgs()
		if update.LastViewed != nil {
			args.Append("last_viewed", *update.LastViewed)
		}
		if update.LastUnreadNotify != nil {
			args.Append("last_unread_notify", *update.LastUnreadNotify)
		}
		if update.LastReferenced != nil {
			args.Append("last_referenced", *update.LastReferenced)
		}
	}

	if args == nil || args.IsEmpty() {
		_, err := d.db.Exec(`
			INSERT INTO thread_entities (thread_id, entity_id)
			VALUES (?, ?) ON DUPLICATE KEY UPDATE thread_id=thread_id`, threadID, entityID)
		return errors.Trace(err)
	}

	insertCols := append([]string{"thread_id", "entity_id"}, args.Columns()...)
	vals := append([]interface{}{threadID, entityID}, args.Values()...)
	vals = append(vals, args.Values()...)

	query := `
		INSERT INTO thread_entities (` + strings.Join(insertCols, ",") + `)
		VALUES (` + dbutil.MySQLArgs(len(insertCols)) + `)
		ON DUPLICATE KEY UPDATE ` + args.ColumnsForUpdate()
	_, err := d.db.Exec(query, vals...)
	return errors.Trace(err)
}

func scanSavedQuery(row dbutil.Scanner) (*models.SavedQuery, error) {
	var sq models.SavedQuery
	var queryBlob []byte
	sq.ID = models.EmptySavedQueryID()
	err := row.Scan(&sq.ID, &sq.Ordinal, &sq.EntityID, &queryBlob, &sq.Title, &sq.Unread, &sq.Total, &sq.NotificationsEnabled, &sq.Type, &sq.Hidden, &sq.Template)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	sq.Query = new(models.Query)
	if err := proto.Unmarshal(queryBlob, sq.Query); err != nil {
		return nil, errors.Trace(err)
	}
	return &sq, nil
}

func scanThread(row dbutil.Scanner) (*models.Thread, error) {
	var t models.Thread
	t.ID = models.EmptyThreadID()
	var lastPrimaryEntityEndpointsData []byte
	var tags sql.NullString
	err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID, &t.LastMessageTimestamp, &t.LastExternalMessageTimestamp,
		&t.LastMessageSummary, &t.LastExternalMessageSummary, &lastPrimaryEntityEndpointsData, &t.Created, &t.MessageCount,
		&t.Type, &t.SystemTitle, &t.UserTitle, &t.Origin, &t.Deleted, &tags)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if tags.Valid {
		t.Tags = parseGroupedTags(tags.String)
	}
	if len(lastPrimaryEntityEndpointsData) != 0 {
		if err := proto.Unmarshal(lastPrimaryEntityEndpointsData, &t.LastPrimaryEntityEndpoints); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return &t, nil
}

func parseGroupedTags(groupedTags string) []models.Tag {
	ts := strings.Split(groupedTags, " ")
	tags := make([]models.Tag, 0, len(ts))
	for _, tag := range ts {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, models.Tag{Name: tag, Hidden: strings.HasPrefix(tag, threading.HiddenTagPrefix)})
		}
	}
	sort.Sort(models.TagsByName(tags))
	return tags
}

func scanThreadEntity(row dbutil.Scanner) (*models.ThreadEntity, error) {
	var te models.ThreadEntity
	te.ThreadID = models.EmptyThreadID()
	if err := row.Scan(&te.ThreadID, &te.EntityID, &te.Member, &te.Following, &te.Joined, &te.LastViewed, &te.LastUnreadNotify, &te.LastReferenced); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &te, nil
}

func scanThreadAndEntity(row dbutil.Scanner) (*models.Thread, *models.ThreadEntity, error) {
	var t models.Thread
	var te models.ThreadEntity
	var teEntityID *string
	var teMember, teFollowing *bool
	var teJoined *time.Time
	te.ThreadID = models.EmptyThreadID()
	t.ID = models.EmptyThreadID()
	var lastPrimaryEntityEndpointsData []byte
	var tags sql.NullString
	err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID, &t.LastMessageTimestamp, &t.LastExternalMessageTimestamp,
		&t.LastMessageSummary, &t.LastExternalMessageSummary, &lastPrimaryEntityEndpointsData, &t.Created, &t.MessageCount, &t.Type,
		&t.SystemTitle, &t.UserTitle, &t.Origin, &t.Deleted,
		&te.ThreadID, &teEntityID, &teMember, &teFollowing, &teJoined, &te.LastViewed, &te.LastUnreadNotify, &te.LastReferenced, &tags)
	if err == sql.ErrNoRows {
		return nil, nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}
	if len(lastPrimaryEntityEndpointsData) != 0 {
		if err := proto.Unmarshal(lastPrimaryEntityEndpointsData, &t.LastPrimaryEntityEndpoints); err != nil {
			return nil, nil, errors.Trace(err)
		}
	}
	if tags.Valid {
		t.Tags = parseGroupedTags(tags.String)
	}
	// The thread entity isn't guaranted to exist
	if te.ThreadID.IsValid && teEntityID != nil {
		te.EntityID = *teEntityID
		te.Member = *teMember
		te.Following = *teFollowing
		te.Joined = *teJoined
		return &t, &te, nil
	}
	return &t, nil, nil
}

func scanThreadItem(row dbutil.Scanner) (*models.ThreadItem, error) {
	it := &models.ThreadItem{
		ID:       models.EmptyThreadItemID(),
		ThreadID: models.EmptyThreadID(),
	}
	var itemType string
	var data []byte
	if err := row.Scan(&it.ID, &it.ThreadID, &it.Created, &it.Modified, &it.ActorEntityID, &it.Internal, &itemType, &data, &it.Deleted); err != nil {
		return nil, errors.Trace(err)
	}
	switch itemType {
	default:
		return nil, errors.Errorf("unknown thread item type %s", itemType)
	case models.ItemTypeMessage:
		m := &models.Message{}
		if err := m.Unmarshal(data); err != nil {
			return nil, errors.Trace(err)
		}
		it.Data = m
	}
	return it, nil
}

func scanThreadItemViewDetails(row dbutil.Scanner) (*models.ThreadItemViewDetails, error) {
	var t models.ThreadItemViewDetails
	t.ThreadItemID = models.EmptyThreadItemID()
	if err := row.Scan(&t.ThreadItemID, &t.ActorEntityID, &t.ViewTime); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &t, nil
}

func parseTimeCursor(cur string) (time.Time, error) {
	ts, err := strconv.ParseInt(cur, 10, 64)
	if err != nil {
		return time.Time{}, errors.Trace(err)
	}
	return time.Unix(ts/1e6, (ts%1e6)*1e3), nil
}

func formatTimeCursor(t time.Time) string {
	return strconv.FormatInt(t.UnixNano()/1e3, 10)
}

// dedupeStrings returns a slice of strings with duplicates removed. The order is not guaranteed to remain the same.
func dedupeStrings(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}
	mp := make(map[string]struct{}, len(ss))
	for i := 0; i < len(ss); i++ {
		s := ss[i]
		if _, ok := mp[s]; !ok {
			mp[s] = struct{}{}
		} else {
			ss[i] = ss[len(ss)-1]
			ss = ss[:len(ss)-1]
			i--
		}
	}
	return ss
}
