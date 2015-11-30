package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
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
	Text         string
	Attachments  []*models.Attachment
	Source       *models.Endpoint
	Destinations []*models.Endpoint
}

type MemberUpdate struct {
	Following *bool
}

type DAL interface {
	CreateSavedQuery(context.Context, *models.SavedQuery) (models.SavedQueryID, error)
	CreateThread(context.Context, *models.Thread) (models.ThreadID, error)
	IterateThreads(ctx context.Context, orgID string, it *Iterator) (*ThreadConnection, error)
	IterateThreadItems(ctx context.Context, threadID models.ThreadID, it *Iterator) (*ThreadItemConnection, error)
	PostMessage(context.Context, *PostMessageRequest) (*models.ThreadItem, error)
	SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error)
	SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error)
	Thread(ctx context.Context, threadID models.ThreadID) (*models.Thread, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error)
	// UpdateMember updates attributes about a thread member. If the membership doesn't exist then it is created.
	UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *MemberUpdate) error

	Transact(context.Context, func(context.Context, DAL) error) error
}

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
	sq.ID = id
	return id, nil
}

func (d *dal) CreateThread(ctx context.Context, thread *models.Thread) (models.ThreadID, error) {
	id, err := models.NewThreadID()
	if err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}
	if _, err := d.db.Exec(`
		INSERT INTO threads (id, organization_id, primary_entity_id)
		VALUES (?, ?, ?)
	`, id, thread.OrganizationID, thread.PrimaryEntityID); err != nil {
		return models.ThreadID{}, errors.Trace(err)
	}
	thread.ID = id
	return id, nil
}

func (d *dal) IterateThreads(ctx context.Context, orgID string, it *Iterator) (*ThreadConnection, error) {
	if it.Count > maxThreadCount {
		it.Count = maxThreadCount
	}
	if it.Count <= 0 {
		it.Count = defaultThreadCount
	}
	cond := []string{"organization_id = ?"}
	vals := []interface{}{orgID}
	if it.StartCursor != "" {
		cond = append(cond, "(id > ?)")
		v, err := models.ParseThreadID(it.StartCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad start cursor"))
		}
		vals = append(vals, v)
	}
	if it.EndCursor != "" {
		cond = append(cond, "(id < ?)")
		v, err := models.ParseThreadID(it.EndCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad end cursor"))
		}
		vals = append(vals, v)
	}
	where := ""
	if len(cond) != 0 {
		where = strings.Join(cond, " AND ")
	}
	order := " ORDER BY id"
	if it.Direction == FromEnd {
		order += " DESC"
	}
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to check if there's more than requested available.. will filter it out later
	rows, err := d.db.Query(`
		SELECT id, COALESCE(primary_entity_id, '')
		FROM threads
		WHERE `+where+order+limit, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadConnection
	for rows.Next() {
		t := &models.Thread{
			ID:             models.EmptyThreadID(),
			OrganizationID: orgID,
		}
		if err := rows.Scan(&t.ID, &t.PrimaryEntityID); err != nil {
			return nil, errors.Trace(err)
		}
		tc.Edges = append(tc.Edges, ThreadEdge{
			Thread: t,
			Cursor: t.ID.String(),
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

func (d *dal) IterateThreadItems(ctx context.Context, threadID models.ThreadID, it *Iterator) (*ThreadItemConnection, error) {
	golog.Debugf("threading.dal.IterateThreadItems: iterator=%+v", it)
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
			return nil, errors.Trace(ErrInvalidIterator("bad start cursor"))
		}
		vals = append(vals, v)
	}
	if it.EndCursor != "" {
		cond = append(cond, "(id < ?)")
		v, err := models.ParseThreadItemID(it.EndCursor)
		if err != nil {
			return nil, errors.Trace(ErrInvalidIterator("bad end cursor"))
		}
		vals = append(vals, v)
	}
	where := ""
	if len(cond) != 0 {
		where = strings.Join(cond, " AND ")
	}
	order := " ORDER BY id"
	if it.Direction == FromEnd {
		order += " DESC"
	}
	limit := fmt.Sprintf(" LIMIT %d", it.Count+1) // +1 to check if there's more than requested available.. will filter it out later
	query := `
		SELECT id, created, actor_entity_id, internal, type, data
		FROM thread_items
		WHERE ` + where + order + limit
	golog.Debugf("threading.dal.IterateThreadItems: query='%s' vals=%s", query, vals)
	rows, err := d.db.Query(query, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tc ThreadItemConnection
	for rows.Next() {
		it := &models.ThreadItem{
			ID:       models.EmptyThreadItemID(),
			ThreadID: threadID,
		}
		var itemType string
		var data []byte
		if err := rows.Scan(&it.ID, &it.Created, &it.ActorEntityID, &it.Internal, &itemType, &data); err != nil {
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
		tc.Edges = append(tc.Edges, ThreadItemEdge{
			Item:   it,
			Cursor: it.ID.String(),
		})
	}
	golog.Debugf("threading.dal.IterateThreadItems: results=%d", len(tc.Edges))

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
		Text:         req.Text,
		Attachments:  req.Attachments,
		Status:       models.Message_NORMAL,
		Source:       req.Source,
		Destinations: req.Destinations,
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
	_, err = d.db.Exec(`
		INSERT INTO thread_items (id, thread_id, created, actor_entity_id, internal, type, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.Created, item.ActorEntityID, item.Internal, string(item.Type), data)
	return item, errors.Trace(err)
}

func (d *dal) SavedQuery(ctx context.Context, id models.SavedQueryID) (*models.SavedQuery, error) {
	row := d.db.QueryRow(`
		SELECT id, organization_id, entity_id
		FROM saved_queries
		WHERE id = ?`, id)
	return scanSavedQuery(row)
}

func (d *dal) SavedQueries(ctx context.Context, entityID string) ([]*models.SavedQuery, error) {
	rows, err := d.db.Query(`
		SELECT id, organization_id, entity_id
		FROM saved_queries
		WHERE entity_id = ?`, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
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
		SELECT id, organization_id, COALESCE(primary_entity_id, '')
		FROM threads
		WHERE id = ?`, id)
	return scanThread(row)
}

func (d *dal) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*models.Thread, error) {
	var rows *sql.Rows
	var err error
	if primaryOnly {
		rows, err = d.db.Query(`
			SELECT id, organization_id, COALESCE(primary_entity_id, '')
			FROM threads
			WHERE primary_entity_id = ?`, entityID)
	} else {
		rows, err = d.db.Query(`
			SELECT t.id, t.organization_id, COALESCE(t.primary_entity_id, '')
			FROM thread_members tm
			INNER JOIN threads t ON t.id = tm.thread_id
			WHERE tm.entity_id = ?`, entityID)
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

func (d *dal) UpdateMember(ctx context.Context, threadID models.ThreadID, entityID string, update *MemberUpdate) error {
	var args dbutil.VarArgs

	if update != nil {
		args = dbutil.MySQLVarArgs()
		if update.Following != nil {
			args.Append("following", *update.Following)
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
	if err := row.Scan(&t.ID, &t.OrganizationID, &t.PrimaryEntityID); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &t, nil
}
