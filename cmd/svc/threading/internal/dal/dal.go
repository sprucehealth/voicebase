package dal

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"golang.org/x/net/context"
)

var ErrNotFound = errors.New("threading/dal: object not found")

type ErrInvalidIterator string

func (e ErrInvalidIterator) Error() string {
	return fmt.Sprintf("threading/dal: invalid iterator: %s", string(e))
}

const (
	maxThreadCount         = 1000
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
	Member           *bool
	LastViewed       *time.Time
	LastUnreadNotify *time.Time
	LastReferenced   *time.Time
}

type OnboardingStateUpdate struct {
	Step *int
}

type DAL interface {
	CreateSavedQuery(context.Context, *models.SavedQuery) (models.SavedQueryID, error)
	CreateOnboardingState(ctx context.Context, threadID models.ThreadID, entityID string) error
	CreateThread(context.Context, *models.Thread) (models.ThreadID, error)
	CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error
	CreateThreadLink(ctx context.Context, thread1Link, thread2Link *ThreadLink) error
	DeleteThread(ctx context.Context, threadID models.ThreadID) error
	EntitiesForThread(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadEntity, error)
	IterateThreads(ctx context.Context, orgEntityID, viewerEntityID string, forExternal bool, it *Iterator) (*ThreadConnection, error)
	IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *Iterator) (*ThreadItemConnection, error)
	LinkedThread(ctx context.Context, threadID models.ThreadID) (*models.Thread, bool, error)
	OnboardingState(ctx context.Context, threadID models.ThreadID, forUpdate bool) (*models.OnboardingState, error)
	OnboardingStateForEntity(ctx context.Context, entityID string, forUpdate bool) (*models.OnboardingState, error)
	PostMessage(context.Context, *PostMessageRequest) (*models.ThreadItem, error)
	RecordThreadEvent(ctx context.Context, threadID models.ThreadID, actorEntityID string, event models.ThreadEvent) error
	SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error)
	SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error)
	Thread(ctx context.Context, id models.ThreadID) (*models.Thread, error)
	ThreadItem(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, error)
	ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error)
	ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error)
	ThreadEntities(ctx context.Context, threadIDs []models.ThreadID, entityID string, forUpdate bool) (map[string]*models.ThreadEntity, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error)
	ThreadsForOrg(ctx context.Context, organizationID string) ([]*models.Thread, error)
	UpdateThread(ctx context.Context, threadID models.ThreadID, update *ThreadUpdate) error
	// UpdateThreadEntity updates attributes about a thread entity. If the thread entity relationship doesn't exist then it is created.
	UpdateThreadEntity(ctx context.Context, threadID models.ThreadID, entityID string, update *ThreadEntityUpdate) error
	UpdateThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error
	UpdateOnboardingState(context.Context, models.ThreadID, *OnboardingStateUpdate) error

	Transact(context.Context, func(context.Context, DAL) error) error
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

type dal struct {
	db tsql.DB
}

// Transact encapsulates the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(ctx context.Context, trans func(context.Context, DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			errString := fmt.Sprintf("Encountered panic during transaction execution: %v", r)
			golog.Errorf(errString)
			err = errors.Trace(errors.New(errString))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

func (d *dal) CreateOnboardingState(ctx context.Context, threadID models.ThreadID, entityID string) error {
	_, err := d.db.Exec(`INSERT INTO onboarding_threads (thread_id, entity_id, step) VALUES (?, ?, ?)`, threadID, entityID, 0)
	return errors.Trace(err)
}

func (d *dal) CreateSavedQuery(ctx context.Context, sq *models.SavedQuery) (models.SavedQueryID, error) {
	id, err := models.NewSavedQueryID()
	if err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
	queryBlob := []byte{} // TODO
	if _, err := d.db.Exec(`
		INSERT INTO saved_queries (id, organization_id, entity_id, query)
		VALUES (?, ?, ?, ?)
	`, id, sq.OrganizationID, sq.EntityID, queryBlob); err != nil {
		return models.SavedQueryID{}, errors.Trace(err)
	}
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
	now := time.Now()
	if _, err := d.db.Exec(`
		INSERT INTO threads (id, organization_id, primary_entity_id, last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, type, system_title, user_title)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, thread.OrganizationID, thread.PrimaryEntityID, now, now, thread.LastMessageSummary, thread.LastExternalMessageSummary, lastPrimaryEntityEndpointsData, thread.Type, thread.SystemTitle, thread.UserTitle); err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}
	thread.ID = id
	return id, nil
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

func (d *dal) IterateThreads(ctx context.Context, orgEntityID, viewerEntityID string, forExternal bool, it *Iterator) (*ThreadConnection, error) {
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
	cond := []string{"(t.type != ? OR te.member = true)", "organization_id = ?", "deleted = ?"}
	vals := []interface{}{viewerEntityID, models.ThreadTypeTeam, orgEntityID, false}
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
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to check if there's more than requested available.. will filter it out later
	rows, err := d.db.Query(`
		SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), t.last_message_timestamp, t.last_external_message_timestamp, t.last_message_summary,
			t.last_external_message_summary, t.last_primary_entity_endpoints, t.created, t.message_count, t.type, COALESCE(t.system_title, ''), COALESCE(t.user_title, ''),
			te.thread_id, te.entity_id, te.member, te.joined, te.last_viewed, te.last_unread_notify, te.last_referenced
		FROM threads t
		LEFT OUTER JOIN thread_entities te ON te.thread_id = t.id AND te.entity_id = ?
		WHERE `+where+order+limit, vals...)
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

	if err := rows.Err(); err != nil {
		return nil, errors.Trace(rows.Err())
	}

	// If we got more than was asked then we know there's more to be had
	if len(tc.Edges) > it.Count {
		tc.Edges = tc.Edges[:it.Count]
		tc.HasMore = true
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
		SELECT id, thread_id, created, actor_entity_id, internal, type, data
		FROM thread_items
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
		SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count, type, COALESCE(system_title, ''), COALESCE(user_title, '')
		FROM threads
		WHERE id = ? AND deleted = false`, linkedThread.ThreadID)
	t, err := scanThread(row)
	return t, linkedThread.PrependSender, errors.Trace(err)
}

func (d *dal) OnboardingState(ctx context.Context, threadID models.ThreadID, forUpdate bool) (*models.OnboardingState, error) {
	var forUpdateSQL string
	if forUpdate {
		forUpdateSQL = ` FOR UPDATE`
	}
	row := d.db.QueryRow(`SELECT thread_id, step FROM onboarding_threads WHERE thread_id = ?`+forUpdateSQL, threadID)
	var state models.OnboardingState
	if err := row.Scan(&state.ThreadID, &state.Step); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &state, nil
}

func (d *dal) OnboardingStateForEntity(ctx context.Context, entityID string, forUpdate bool) (*models.OnboardingState, error) {
	var forUpdateSQL string
	if forUpdate {
		forUpdateSQL = ` FOR UPDATE`
	}
	row := d.db.QueryRow(`SELECT thread_id, step FROM onboarding_threads WHERE entity_id = ?`+forUpdateSQL, entityID)
	var state models.OnboardingState
	if err := row.Scan(&state.ThreadID, &state.Step); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &state, nil
}

func (d *dal) PostMessage(ctx context.Context, req *PostMessageRequest) (*models.ThreadItem, error) {
	// TODO: validate request

	id, err := models.NewThreadItemID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	msg := &models.Message{
		Title:        req.Title,
		Text:         req.Text,
		Attachments:  req.Attachments,
		Status:       models.Message_NORMAL,
		Source:       req.Source,
		Destinations: req.Destinations,
		TextRefs:     req.TextRefs,
		Summary:      req.Summary,
	}
	item := &models.ThreadItem{
		ID:            id,
		ThreadID:      req.ThreadID,
		Created:       time.Now(),
		ActorEntityID: req.FromEntityID,
		Internal:      req.Internal,
		Type:          models.ItemTypeMessage,
		Data:          msg,
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, errors.Trace(err)
	}

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

	_, err = tx.Exec(`
		INSERT INTO thread_items (id, thread_id, created, actor_entity_id, internal, type, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.Created, item.ActorEntityID, item.Internal, string(item.Type), data)
	if err != nil {
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
		SELECT id, organization_id, entity_id
		FROM saved_queries
		WHERE id = ?`, id)
	sq, err := scanSavedQuery(row)
	return sq, errors.Trace(err)
}

func (d *dal) SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rows, err := d.db.Query(`
		SELECT id, organization_id, entity_id
		FROM saved_queries
		WHERE entity_id = ?`, entityID)
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

func (d *dal) Thread(ctx context.Context, id models.ThreadID) (*models.Thread, error) {
	row := d.db.QueryRow(`
		SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count, type, COALESCE(system_title, ''), COALESCE(user_title, '')
		FROM threads
		WHERE id = ? AND deleted = false`, id)
	t, err := scanThread(row)
	return t, errors.Trace(err)
}

func (d *dal) ThreadItem(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, error) {
	row := d.db.QueryRow(`
		SELECT id, thread_id, created, actor_entity_id, internal, type, data
		FROM thread_items
		WHERE id = ?`, id)
	ti, err := scanThreadItem(row)
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

func (d *dal) ThreadEntities(ctx context.Context, threadIDs []models.ThreadID, entityID string, forUpdate bool) (map[string]*models.ThreadEntity, error) {
	if len(threadIDs) == 0 || entityID == "" {
		return nil, nil
	}

	var sfu string
	if forUpdate {
		sfu = "ORDER BY thread_id FOR UPDATE"
	}
	values := make([]interface{}, len(threadIDs)+1)
	for i, v := range threadIDs {
		values[i] = v
	}
	values[len(threadIDs)] = entityID
	rows, err := d.db.Query(`
		SELECT thread_id, entity_id, member, joined, last_viewed, last_unread_notify, last_referenced
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
		SELECT thread_id, entity_id, member, joined, last_viewed, last_unread_notify, last_referenced
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
			SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count, type, COALESCE(system_title, ''), COALESCE(user_title, '')
			FROM threads
			WHERE primary_entity_id = ? AND deleted = false`, entityID)
	} else {
		rows, err = d.db.Query(`
			SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count, type, COALESCE(system_title, ''), COALESCE(user_title, '')
			FROM thread_entities tm
			INNER JOIN threads t ON t.id = tm.thread_id
			WHERE tm.entity_id = ? AND deleted = false`, entityID)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
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

func (d *dal) ThreadsForOrg(ctx context.Context, organizationID string) ([]*models.Thread, error) {
	var rows *sql.Rows
	var err error
	rows, err = d.db.Query(`
			SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count, type, COALESCE(system_title, ''), COALESCE(user_title, '')
			FROM threads
			WHERE organization_id = ? AND deleted = false`, organizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
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

func (d *dal) UpdateThreadEntity(ctx context.Context, threadID models.ThreadID, entityID string, update *ThreadEntityUpdate) error {
	var args dbutil.VarArgs

	if update != nil {
		args = dbutil.MySQLVarArgs()
		if update.Member != nil {
			args.Append("member", *update.Member)
		}
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
			INSERT IGNORE INTO thread_entities (thread_id, entity_id)
			VALUES (?, ?)`, threadID, entityID)
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

func (d *dal) UpdateThreadMembers(ctx context.Context, threadID models.ThreadID, memberEntityIDs []string) error {
	// Dedupe entity IDs as the queries below will fail otherwise
	memberEntityIDs = dedupeStrings(memberEntityIDs)

	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	// First remove all members so we can add back the ones we want
	_, err = tx.Exec(`UPDATE thread_entities SET member = ? WHERE thread_id = ?`, false, threadID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	if len(memberEntityIDs) == 0 {
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
		return nil
	}

	// Get a list of all existing thread entity records since we need to do a combination of update + insert
	rows, err := tx.Query(`SELECT entity_id FROM thread_entities WHERE entity_id IN (`+dbutil.MySQLArgs(len(memberEntityIDs))+`) AND thread_id = ?`,
		append(dbutil.AppendStringsToInterfaceSlice(nil, memberEntityIDs), threadID)...)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	defer rows.Close()
	teids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
		teids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	var teUpdates []interface{}
	ins := dbutil.MySQLMultiInsert(0)
	for _, id := range memberEntityIDs {
		if _, ok := teids[id]; ok {
			teUpdates = append(teUpdates, id)
		} else {
			ins.Append(threadID, id, true)
		}
	}
	if len(teUpdates) != 0 {
		_, err = tx.Exec(`UPDATE thread_entities SET member = true WHERE entity_id IN (`+dbutil.MySQLArgs(len(teUpdates))+`) AND thread_id = ?`, append(teUpdates, threadID)...)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}
	if !ins.IsEmpty() {
		_, err = tx.Exec(`
			INSERT INTO thread_entities (thread_id, entity_id, member)
			VALUES `+ins.Query(), ins.Values()...)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) UpdateOnboardingState(ctx context.Context, threadID models.ThreadID, update *OnboardingStateUpdate) error {
	args := dbutil.MySQLVarArgs()
	if update.Step != nil {
		args.Append("step", *update.Step)
	}
	if args.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE onboarding_threads SET `+args.ColumnsForUpdate()+` WHERE thread_id = ?`,
		append(args.Values(), threadID)...)
	return errors.Trace(err)
}

func scanSavedQuery(row dbutil.Scanner) (*models.SavedQuery, error) {
	var sq models.SavedQuery
	sq.ID = models.EmptySavedQueryID()
	if err := row.Scan(&sq.ID, &sq.OrganizationID, &sq.EntityID); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &sq, nil
}

func scanThread(row dbutil.Scanner) (*models.Thread, error) {
	var t models.Thread
	t.ID = models.EmptyThreadID()
	var lastPrimaryEntityEndpointsData []byte
	err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID, &t.LastMessageTimestamp, &t.LastExternalMessageTimestamp,
		&t.LastMessageSummary, &t.LastExternalMessageSummary, &lastPrimaryEntityEndpointsData, &t.Created, &t.MessageCount,
		&t.Type, &t.SystemTitle, &t.UserTitle)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if len(lastPrimaryEntityEndpointsData) != 0 {
		if err := proto.Unmarshal(lastPrimaryEntityEndpointsData, &t.LastPrimaryEntityEndpoints); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return &t, nil
}

func scanThreadEntity(row dbutil.Scanner) (*models.ThreadEntity, error) {
	var te models.ThreadEntity
	te.ThreadID = models.EmptyThreadID()
	if err := row.Scan(&te.ThreadID, &te.EntityID, &te.Member, &te.Joined, &te.LastViewed, &te.LastUnreadNotify, &te.LastReferenced); err == sql.ErrNoRows {
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
	var teMember *bool
	var teJoined *time.Time
	te.ThreadID = models.EmptyThreadID()
	t.ID = models.EmptyThreadID()
	var lastPrimaryEntityEndpointsData []byte
	err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID, &t.LastMessageTimestamp, &t.LastExternalMessageTimestamp,
		&t.LastMessageSummary, &t.LastExternalMessageSummary, &lastPrimaryEntityEndpointsData, &t.Created, &t.MessageCount, &t.Type,
		&t.SystemTitle, &t.UserTitle, &te.ThreadID, &teEntityID, &teMember, &teJoined, &te.LastViewed, &te.LastUnreadNotify, &te.LastReferenced)
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
	// The thread entity isn't guaranted to exist
	if te.ThreadID.IsValid && teEntityID != nil {
		te.EntityID = *teEntityID
		te.Member = *teMember
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
	if err := row.Scan(&it.ID, &it.ThreadID, &it.Created, &it.ActorEntityID, &it.Internal, &itemType, &data); err != nil {
		return nil, errors.Trace(err)
	}
	it.Type = models.ItemType(itemType)
	switch it.Type {
	default:
		return nil, errors.Trace(fmt.Errorf("unknown thread item type %s", itemType))
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
