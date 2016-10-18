package dal

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

func (d *dal) CreateSavedMessage(ctx context.Context, sm *models.SavedMessage) (models.SavedMessageID, error) {
	var err error
	sm.ID, err = models.NewSavedMessageID()
	if err != nil {
		return sm.ID, errors.Trace(err)
	}
	if sm.Modified.IsZero() {
		sm.Modified = d.clk.Now()
	}
	if sm.Created.IsZero() {
		sm.Created = d.clk.Now()
	}
	itemType, err := models.ItemTypeForValue(sm.Content)
	if err != nil {
		return sm.ID, errors.Trace(err)
	}
	data, err := sm.Content.Marshal()
	if err != nil {
		return sm.ID, errors.Trace(err)
	}
	_, err = d.db.Exec(`
		INSERT INTO saved_messages (id, title, organization_id, creator_entity_id, owner_entity_id, internal, type, data, created, modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sm.ID, sm.Title, sm.OrganizationID, sm.CreatorEntityID, sm.OwnerEntityID, sm.Internal, string(itemType), data, sm.Created, sm.Modified)
	return sm.ID, errors.Trace(err)
}

func (d *dal) DeleteSavedMessages(ctx context.Context, ids []models.SavedMessageID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ifids := make([]interface{}, len(ids))
	for i, id := range ids {
		ifids[i] = id
	}
	res, err := d.db.Exec(`DELETE FROM saved_messages WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`, ifids...)
	if err != nil {
		return 0, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	return int(n), errors.Trace(err)
}

func (d *dal) SavedMessages(ctx context.Context, ids []models.SavedMessageID) ([]*models.SavedMessage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idfs := make([]interface{}, len(ids))
	for i, id := range ids {
		idfs[i] = id
	}
	rows, err := d.db.Query(`
		SELECT id, title, organization_id, creator_entity_id, owner_entity_id, internal, type, data, created, modified
		FROM saved_messages
		WHERE id IN (`+dbutil.MySQLArgs(len(idfs))+`)`,
		idfs...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	var msgs []*models.SavedMessage
	for rows.Next() {
		sm, err := scanSavedMessage(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		msgs = append(msgs, sm)
	}
	return msgs, errors.Trace(rows.Err())
}

func (d *dal) SavedMessagesForEntities(ctx context.Context, ownerEntityIDs []string) ([]*models.SavedMessage, error) {
	if len(ownerEntityIDs) == 0 {
		return nil, nil
	}
	rows, err := d.db.Query(`
		SELECT id, title, organization_id, creator_entity_id, owner_entity_id, internal, type, data, created, modified
		FROM saved_messages
		WHERE owner_entity_id IN (`+dbutil.MySQLArgs(len(ownerEntityIDs))+`)`,
		dbutil.AppendStringsToInterfaceSlice(nil, ownerEntityIDs)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	var msgs []*models.SavedMessage
	for rows.Next() {
		sm, err := scanSavedMessage(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		msgs = append(msgs, sm)
	}
	return msgs, errors.Trace(rows.Err())
}

type SavedMessageUpdate struct {
	Title   *string
	Content models.ItemValue
}

func (d *dal) UpdateSavedMessage(ctx context.Context, id models.SavedMessageID, update *SavedMessageUpdate) error {
	if update == nil {
		return nil
	}

	args := dbutil.MySQLVarArgs()
	if update.Title != nil {
		args.Append("title", *update.Title)
	}
	if update.Content != nil {
		itemType, err := models.ItemTypeForValue(update.Content)
		if err != nil {
			return errors.Trace(err)
		}
		data, err := update.Content.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		args.Append("type", string(itemType))
		args.Append("data", data)
	}
	if args.IsEmpty() {
		return nil
	}

	_, err := d.db.Exec(`UPDATE saved_messages SET `+args.ColumnsForUpdate()+` WHERE id = ?`,
		append(args.Values(), id)...)
	return errors.Trace(err)
}

func scanSavedMessage(row dbutil.Scanner) (*models.SavedMessage, error) {
	sm := &models.SavedMessage{
		ID: models.EmptySavedMessageID(),
	}
	var itemType string
	var data []byte
	if err := row.Scan(&sm.ID, &sm.Title, &sm.OrganizationID, &sm.CreatorEntityID, &sm.OwnerEntityID, &sm.Internal, &itemType, &data, &sm.Created, &sm.Modified); err != nil {
		return nil, errors.Trace(err)
	}
	switch t := models.ItemType(itemType); t {
	default:
		return nil, errors.Errorf("unknown thread item type %s", t)
	case models.ItemTypeMessage:
		m := &models.Message{}
		if err := m.Unmarshal(data); err != nil {
			return nil, errors.Trace(err)
		}
		sm.Content = m
	}
	return sm, nil
}
