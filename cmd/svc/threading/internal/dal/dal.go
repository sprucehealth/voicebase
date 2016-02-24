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
	Thread *models.Thread
	Cursor string
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

type MemberUpdate struct {
	Following        *bool
	LastViewed       *time.Time
	LastUnreadNotify *time.Time
}

type DAL interface {
	CreateSavedQuery(context.Context, *models.SavedQuery) (models.SavedQueryID, error)
	CreateThread(context.Context, *models.Thread) (models.ThreadID, error)
	CreateThreadItemViewDetails(ctx context.Context, tds []*models.ThreadItemViewDetails) error
	DeleteThread(ctx context.Context, threadID models.ThreadID) error
	IterateThreads(ctx context.Context, orgID string, forExternal bool, it *Iterator) (*ThreadConnection, error)
	IterateThreadItems(ctx context.Context, threadID models.ThreadID, forExternal bool, it *Iterator) (*ThreadItemConnection, error)
	PostMessage(context.Context, *PostMessageRequest) (*models.ThreadItem, error)
	RecordThreadEvent(ctx context.Context, threadID models.ThreadID, actorEntityID string, event models.ThreadEvent) error
	SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error)
	SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error)
	Thread(ctx context.Context, id models.ThreadID) (*models.Thread, error)
	ThreadItem(ctx context.Context, id models.ThreadItemID) (*models.ThreadItem, error)
	ThreadItemIDsCreatedAfter(ctx context.Context, threadID models.ThreadID, after time.Time) ([]models.ThreadItemID, error)
	ThreadItemViewDetails(ctx context.Context, id models.ThreadItemID) ([]*models.ThreadItemViewDetails, error)
	ThreadMemberships(ctx context.Context, threadIDs []models.ThreadID, entityIDs []string, forUpdate bool) (map[string][]*models.ThreadMember, error)
	ThreadMembers(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadMember, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error)
	ThreadsForOrg(ctx context.Context, organizationID string) ([]*models.Thread, error)
	// UpdateMember updates attributes about a thread member. If the membership doesn't exist then it is created.
	UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *MemberUpdate) error

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
		INSERT INTO threads (id, organization_id, primary_entity_id, last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, thread.OrganizationID, thread.PrimaryEntityID, now, now, thread.LastMessageSummary, thread.LastExternalMessageSummary, lastPrimaryEntityEndpointsData); err != nil {
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

func (d *dal) DeleteThread(ctx context.Context, threadID models.ThreadID) error {
	_, err := d.db.Exec(`UPDATE threads SET deleted = true WHERE id = ?`, threadID)
	return errors.Trace(err)
}

func (d *dal) IterateThreads(ctx context.Context, orgID string, forExternal bool, it *Iterator) (*ThreadConnection, error) {
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
	cond := []string{"organization_id = ?", "deleted = ?"}
	vals := []interface{}{orgID, false}
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
		SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count
		FROM threads
		WHERE `+where+order+limit, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadConnection
	for rows.Next() {
		t, err := scanThread(rows)
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
			Thread: t,
			Cursor: cursor,
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

	return &tc, errors.Trace(rows.Err())
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
		SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count
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

func (d *dal) ThreadMemberships(ctx context.Context, threadIDs []models.ThreadID, entityIDs []string, forUpdate bool) (map[string][]*models.ThreadMember, error) {
	if len(threadIDs) == 0 || len(entityIDs) == 0 {
		return nil, nil
	}

	var sfu string
	if forUpdate {
		sfu = "FOR UPDATE"
	}
	values := make([]interface{}, len(threadIDs)+len(entityIDs))
	for i, e := range entityIDs {
		values[i] = e
	}
	for i, v := range threadIDs {
		values[i+len(entityIDs)] = v
	}
	rows, err := d.db.Query(fmt.Sprintf(`
		SELECT thread_id, entity_id, following, joined, last_viewed, last_unread_notify
		FROM thread_members
		WHERE entity_id IN (`+dbutil.MySQLArgs(len(entityIDs))+`)
        AND thread_id IN (`+dbutil.MySQLArgs(len(threadIDs))+`) %s`, sfu), values...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	tms := make(map[string][]*models.ThreadMember)
	for rows.Next() {
		tm, err := scanThreadMember(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tms[tm.EntityID] = append(tms[tm.EntityID], tm)
	}
	return tms, errors.Trace(err)
}

func (d *dal) ThreadMembers(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadMember, error) {
	rows, err := d.db.Query(`
		SELECT thread_id, entity_id, following, joined, last_viewed, last_unread_notify
		FROM thread_members
        WHERE thread_id = ?`, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tms []*models.ThreadMember
	for rows.Next() {
		tm, err := scanThreadMember(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tms = append(tms, tm)
	}
	return tms, errors.Trace(err)
}

func (d *dal) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	var rows *sql.Rows
	var err error
	if primaryOnly {
		rows, err = d.db.Query(`
			SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count
			FROM threads
			WHERE primary_entity_id = ? AND deleted = false`, entityID)
	} else {
		rows, err = d.db.Query(`
			SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count
			FROM thread_members tm
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
			SELECT id, organization_id, COALESCE(primary_entity_id, ''), last_message_timestamp, last_external_message_timestamp, last_message_summary, last_external_message_summary, last_primary_entity_endpoints, created, message_count
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

func (d *dal) UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *MemberUpdate) error {
	var args dbutil.VarArgs

	if update != nil {
		args = dbutil.MySQLVarArgs()
		if update.Following != nil {
			args.Append("following", *update.Following)
		}
		if update.LastViewed != nil {
			args.Append("last_viewed", *update.LastViewed)
		}
		if update.LastUnreadNotify != nil {
			args.Append("last_unread_notify", *update.LastUnreadNotify)
		}
	}

	if args == nil || args.IsEmpty() {
		_, err := d.db.Exec(`
			INSERT IGNORE INTO thread_members (thread_id, entity_id)
			VALUES (?, ?)`, threadID, entityID)
		return errors.Trace(err)
	}

	insertCols := append([]string{"thread_id", "entity_id"}, args.Columns()...)
	vals := append([]interface{}{threadID, entityID}, args.Values()...)
	vals = append(vals, args.Values()...)

	query := `
		INSERT INTO thread_members (` + strings.Join(insertCols, ",") + `)
		VALUES (` + dbutil.MySQLArgs(len(insertCols)) + `)
		ON DUPLICATE KEY UPDATE ` + args.ColumnsForUpdate()
	_, err := d.db.Exec(query, vals...)
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
	err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID, &t.LastMessageTimestamp, &t.LastExternalMessageTimestamp, &t.LastMessageSummary, &t.LastExternalMessageSummary, &lastPrimaryEntityEndpointsData, &t.Created, &t.MessageCount)
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

func scanThreadMember(row dbutil.Scanner) (*models.ThreadMember, error) {
	var tm models.ThreadMember
	tm.ThreadID = models.EmptyThreadID()
	if err := row.Scan(&tm.ThreadID, &tm.EntityID, &tm.Following, &tm.Joined, &tm.LastViewed, &tm.LastUnreadNotify); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &tm, nil
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
