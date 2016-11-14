package dal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

// CreateScheduledMessage inserts a scheduled_message record
func (d *dal) CreateScheduledMessage(ctx context.Context, model *models.ScheduledMessage) (models.ScheduledMessageID, error) {
	if !model.ID.IsValid {
		id, err := models.NewScheduledMessageID()
		if err != nil {
			return models.EmptyScheduledMessageID(), errors.Trace(err)
		}
		model.ID = id
	}

	serializedData, err := model.Data.Marshal()
	if err != nil {
		return models.EmptyScheduledMessageID(), errors.Trace(err)
	}
	itemType, err := models.ItemTypeForValue(model.Data)
	if err != nil {
		return models.ScheduledMessageID{}, errors.Trace(err)
	}
	_, err = d.db.Exec(
		`INSERT INTO scheduled_messages
          (actor_entity_id, type, scheduled_for, sent_at, id, thread_id, internal, data, status, sent_thread_item_id)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.ActorEntityID, itemType, model.ScheduledFor, model.SentAt, model.ID,
		model.ThreadID, model.Internal, serializedData, model.Status, model.SentThreadItemID)
	if err != nil {
		return models.EmptyScheduledMessageID(), errors.Trace(err)
	}

	return model.ID, nil
}

// ScheduledMessage retrieves a scheduled_message record
func (d *dal) ScheduledMessage(ctx context.Context, id models.ScheduledMessageID, opts ...QueryOption) (*models.ScheduledMessage, error) {
	q := selectScheduledMessage + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanScheduledMessage(ctx, row, "%s", id)
	return model, errors.Trace(err)
}

// ScheduledMessagesForThread retrieves the scheduled messages for the indicated thread
func (d *dal) ScheduledMessagesForThread(ctx context.Context, threadID models.ThreadID, status []models.ScheduledMessageStatus, opts ...QueryOption) ([]*models.ScheduledMessage, error) {
	args := []interface{}{threadID}
	q := selectScheduledMessage + ` WHERE thread_id = ?`
	if len(status) > 0 {
		q += ` AND status IN (` + dbutil.MySQLArgs(len(status)) + `)`
		for _, s := range status {
			args = append(args, s)
		}
	}
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var scheduledMessages []*models.ScheduledMessage
	for rows.Next() {
		sm, err := scanScheduledMessage(ctx, rows, fmt.Sprintf("thread_id: %s - status: %v", threadID, status))
		if err != nil {
			return nil, errors.Trace(err)
		}
		scheduledMessages = append(scheduledMessages, sm)
	}
	return scheduledMessages, errors.Trace(rows.Err())
}

// ScheduledMessage retrieves a scheduled_message record
func (d *dal) ScheduledMessages(ctx context.Context, status []models.ScheduledMessageStatus, scheduledForBefore time.Time, opts ...QueryOption) ([]*models.ScheduledMessage, error) {
	args := []interface{}{scheduledForBefore}
	q := selectScheduledMessage + ` WHERE scheduled_for < ?`
	if len(status) > 0 {
		q += ` AND status IN (` + dbutil.MySQLArgs(len(status)) + `)`
		for _, s := range status {
			args = append(args, s)
		}
	}
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var scheduledMessages []*models.ScheduledMessage
	for rows.Next() {
		sm, err := scanScheduledMessage(ctx, rows, fmt.Sprintf("scheduled_for < %s - status: %v", scheduledForBefore, status))
		if err != nil {
			return nil, errors.Trace(err)
		}
		scheduledMessages = append(scheduledMessages, sm)
	}
	return scheduledMessages, errors.Trace(rows.Err())
}

// UpdateScheduledMessage updates the mutable aspects of a scheduled_message record
func (d *dal) UpdateScheduledMessage(ctx context.Context, id models.ScheduledMessageID, update *models.ScheduledMessageUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.SentAt != nil {
		args.Append("sent_at", *update.SentAt)
	}
	if update.SentAt != nil {
		args.Append("sent_thread_item_id", *update.SentThreadItemID)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE scheduled_messages
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteScheduledMessage deletes a scheduled_message record
func (d *dal) DeleteScheduledMessage(ctx context.Context, id models.ScheduledMessageID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM scheduled_messages
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectScheduledMessage = `
    SELECT sm.actor_entity_id, sm.type, sm.scheduled_for, sm.sent_at, sm.created, sm.modified, sm.id,
        sm.thread_id, sm.internal, sm.status, sm.sent_thread_item_id, sm.data
    FROM scheduled_messages sm`

func scanScheduledMessage(ctx context.Context, row dbutil.Scanner, contextFormat string, args ...interface{}) (*models.ScheduledMessage, error) {
	var m models.ScheduledMessage
	m.ID = models.EmptyScheduledMessageID()
	m.ThreadID = models.EmptyThreadID()
	m.SentThreadItemID = models.EmptyThreadItemID()

	var itemType string
	var data []byte
	err := row.Scan(&m.ActorEntityID, &itemType, &m.ScheduledFor, &m.SentAt, &m.Created, &m.Modified, &m.ID,
		&m.ThreadID, &m.Internal, &m.Status, &m.SentThreadItemID, &data)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(ErrNotFound, "No rows found - threading.scheduled_messages - Context: "+fmt.Sprintf(contextFormat, args...))
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	switch itemType {
	default:
		return nil, errors.Errorf("unknown thread item type %s", itemType)
	case models.ItemTypeMessage:
		msg := &models.Message{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, errors.Trace(err)
		}
		m.Data = msg
	}
	return &m, errors.Trace(err)
}
