package dal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

// CreateTriggeredMessage inserts a triggered_messages record
func (d *dal) CreateTriggeredMessage(ctx context.Context, model *models.TriggeredMessage) (models.TriggeredMessageID, error) {
	if !model.ID.IsValid {
		id, err := models.NewTriggeredMessageID()
		if err != nil {
			return models.EmptyTriggeredMessageID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO triggered_messages
          (trigger_key, trigger_subkey, id, actor_entity_id, organization_entity_id, enabled)
          VALUES (?, ?, ?, ?, ?, ?)`, model.TriggerKey, model.TriggerSubkey, model.ID, model.ActorEntityID, model.OrganizationEntityID, model.Enabled)
	if err != nil {
		return models.EmptyTriggeredMessageID(), errors.Trace(err)
	}

	return model.ID, nil
}

// CreateTriggeredMessages inserts triggered_messages records
func (d *dal) CreateTriggeredMessages(ctx context.Context, ms []*models.TriggeredMessage) error {
	for i, model := range ms {
		if !model.ID.IsValid {
			id, err := models.NewTriggeredMessageID()
			if err != nil {
				return errors.Trace(err)
			}
			ms[i].ID = id
		}
	}

	ins := dbutil.MySQLMultiInsert(len(ms))
	for _, model := range ms {
		ins.Append(model.OrganizationEntityID, model.ActorEntityID, model.TriggerKey, model.TriggerSubkey, model.ID, model.Enabled)
	}

	_, err := d.db.Exec(
		`INSERT INTO triggered_messages
			(organization_entity_id, actor_entity_id, trigger_key, trigger_subkey, id, enabled)
			VALUES `+ins.Query(), ins.Values()...)
	return errors.Trace(err)
}

// TriggeredMessage retrieves a triggered_messages record
func (d *dal) TriggeredMessage(ctx context.Context, id models.TriggeredMessageID, opts ...QueryOption) (*models.TriggeredMessage, error) {
	q := selectTriggeredMessage + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanTriggeredMessage(ctx, row, "id = %v", id)
	return model, errors.Trace(err)
}

// TriggeredMessage retrieves a triggered_messages record
func (d *dal) TriggeredMessageForKeys(ctx context.Context, organizationEntityID, triggerKey, triggerSubkey string, opts ...QueryOption) (*models.TriggeredMessage, error) {
	q := selectTriggeredMessage + ` WHERE organization_entity_id = ? AND trigger_subkey = ? AND trigger_key = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, organizationEntityID, triggerSubkey, triggerKey)
	model, err := scanTriggeredMessage(ctx, row, "organization_entity_id = %v, trigger_key = %v, trigger_subkey = %v", organizationEntityID, triggerSubkey, triggerKey)
	return model, errors.Trace(err)
}

// DeleteTriggeredMessage deletes a triggered_messages record
func (d *dal) DeleteTriggeredMessage(ctx context.Context, id models.TriggeredMessageID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM triggered_messages
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// UpdateTriggeredMessage updates the mutable aspects of a triggered_messages record
func (d *dal) UpdateTriggeredMessage(ctx context.Context, id models.TriggeredMessageID, update *models.TriggeredMessageUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Enabled != nil {
		args.Append("enabled", *update.Enabled)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE triggered_messages
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// CreateTriggeredMessageItem inserts a triggered_message_items record
func (d *dal) CreateTriggeredMessageItem(ctx context.Context, model *models.TriggeredMessageItem) (models.TriggeredMessageItemID, error) {
	if !model.ID.IsValid {
		id, err := models.NewTriggeredMessageItemID()
		if err != nil {
			return models.EmptyTriggeredMessageItemID(), errors.Trace(err)
		}
		model.ID = id
	}

	serializedData, err := model.Data.Marshal()
	if err != nil {
		return models.EmptyTriggeredMessageItemID(), errors.Trace(err)
	}
	itemType, err := models.ItemTypeForValue(model.Data)
	if err != nil {
		return models.TriggeredMessageItemID{}, errors.Trace(err)
	}

	_, err = d.db.Exec(
		`INSERT INTO triggered_message_items
          (data, id, triggered_message_id, ordinal, internal, actor_entity_id, type)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, serializedData, model.ID, model.TriggeredMessageID, model.Ordinal, model.Internal, model.ActorEntityID, itemType)
	if err != nil {
		return models.EmptyTriggeredMessageItemID(), errors.Trace(err)
	}

	return model.ID, nil
}

// CreateTriggeredMessageItems inserts triggered_message_items records
func (d *dal) CreateTriggeredMessageItems(ctx context.Context, ms []*models.TriggeredMessageItem) error {
	for i, model := range ms {
		if !model.ID.IsValid {
			id, err := models.NewTriggeredMessageItemID()
			if err != nil {
				return errors.Trace(err)
			}
			ms[i].ID = id
		}
	}

	ins := dbutil.MySQLMultiInsert(len(ms))
	for _, model := range ms {
		serializedData, err := model.Data.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		itemType, err := models.ItemTypeForValue(model.Data)
		if err != nil {
			return errors.Trace(err)
		}
		ins.Append(model.Ordinal, model.Internal, model.ActorEntityID, itemType, serializedData, model.ID, model.TriggeredMessageID)
	}

	_, err := d.db.Exec(
		`INSERT INTO triggered_message_items
			(ordinal, internal, actor_entity_id, ordinal, type, data, id, triggered_message_id) 
			VALUES `+ins.Query(), ins.Values()...)
	return errors.Trace(err)
}

// TriggeredMessageItem retrieves a triggered_message_items record
func (d *dal) TriggeredMessageItem(ctx context.Context, id models.TriggeredMessageItemID, opts ...QueryOption) (*models.TriggeredMessageItem, error) {
	q := selectTriggeredMessageItem + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanTriggeredMessageItem(ctx, row, "id = %v", id)
	return model, errors.Trace(err)
}

// TriggeredMessageItem retrieves a triggered_message_items record
func (d *dal) TriggeredMessageItemsForTriggeredMessage(ctx context.Context, triggeredMessageID models.TriggeredMessageID, opts ...QueryOption) ([]*models.TriggeredMessageItem, error) {
	q := selectTriggeredMessageItem + ` WHERE triggered_message_id = ? ORDER BY ordinal ASC`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, triggeredMessageID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ms []*models.TriggeredMessageItem
	for rows.Next() {
		m, err := scanTriggeredMessageItem(ctx, rows, "triggered_message_id = %v", triggeredMessageID)

		if err != nil {
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, errors.Trace(rows.Err())
}

// DeleteTriggeredMessageItem deletes a triggered_message_items record
func (d *dal) DeleteTriggeredMessageItem(ctx context.Context, id models.TriggeredMessageItemID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM triggered_message_items
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteTriggeredMessageItemsForTriggeredMessage deletes triggered_message_items records that map to the provided TriggeredMessageID
func (d *dal) DeleteTriggeredMessageItemsForTriggeredMessage(ctx context.Context, id models.TriggeredMessageID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM triggered_message_items
          WHERE triggered_message_id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectTriggeredMessage = `
    SELECT triggered_messages.created, triggered_messages.modified, triggered_messages.id, triggered_messages.actor_entity_id, triggered_messages.organization_entity_id, triggered_messages.trigger_key, triggered_messages.trigger_subkey, triggered_messages.enabled
      FROM triggered_messages`

func scanTriggeredMessage(ctx context.Context, row dbutil.Scanner, contextFormat string, args ...interface{}) (*models.TriggeredMessage, error) {
	var m models.TriggeredMessage
	m.ID = models.EmptyTriggeredMessageID()

	err := row.Scan(&m.Created, &m.Modified, &m.ID, &m.ActorEntityID, &m.OrganizationEntityID, &m.TriggerKey, &m.TriggerSubkey, &m.Enabled)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(ErrNotFound, "No rows found - threading.TriggeredMessage - Context: "+fmt.Sprintf(contextFormat, args...))
	}
	return &m, errors.Trace(err)
}

const selectTriggeredMessageItem = `
    SELECT triggered_message_items.type, triggered_message_items.data, triggered_message_items.created, triggered_message_items.modified, triggered_message_items.id, triggered_message_items.triggered_message_id, triggered_message_items.ordinal, triggered_message_items.internal, triggered_message_items.actor_entity_id
      FROM triggered_message_items`

func scanTriggeredMessageItem(ctx context.Context, row dbutil.Scanner, contextFormat string, args ...interface{}) (*models.TriggeredMessageItem, error) {
	var m models.TriggeredMessageItem
	m.ID = models.EmptyTriggeredMessageItemID()
	m.TriggeredMessageID = models.EmptyTriggeredMessageID()

	var data []byte
	err := row.Scan(&m.Type, &data, &m.Created, &m.Modified, &m.ID, &m.TriggeredMessageID, &m.Ordinal, &m.Internal, &m.ActorEntityID)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(ErrNotFound, "No rows found - threading.TriggeredMessageItem - Context: "+fmt.Sprintf(contextFormat, args...))
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	switch m.Type {
	default:
		return nil, errors.Errorf("unknown thread item type %s", m.Type)
	case models.ItemTypeMessage:
		msg := &models.Message{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, errors.Trace(err)
		}
		m.Data = msg
	}
	return &m, errors.Trace(err)
}
