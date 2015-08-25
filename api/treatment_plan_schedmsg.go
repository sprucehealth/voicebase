package api

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) TreatmentPlanScheduledMessage(id int64) (*common.TreatmentPlanScheduledMessage, error) {
	return d.treatmentPlanScheduledMessage(id, "treatment_plan")
}

func (d *dataService) treatmentPlanScheduledMessage(id int64, tbl string) (*common.TreatmentPlanScheduledMessage, error) {
	row := d.db.QueryRow(`
			SELECT id, scheduled_days, message, scheduled_message_id, treatment_plan_id
			FROM `+tbl+`_scheduled_message
			WHERE id = ?`, id)

	m := &common.TreatmentPlanScheduledMessage{}
	if err := row.Scan(
		&m.ID, &m.ScheduledDays, &m.Message,
		&m.ScheduledMessageID, &m.TreatmentPlanID,
	); err == sql.ErrNoRows {
		return nil, ErrNotFound(tbl + "_scheduled_message")
	} else if err != nil {
		return nil, err
	}

	// Attachments

	rows, err := d.db.Query(`
		SELECT a.id, item_type, item_id, title, mimetype
		FROM `+tbl+`_scheduled_message_attachment a
		LEFT OUTER JOIN media m ON m.id = item_id
		WHERE `+tbl+`_scheduled_message_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		a := &common.CaseMessageAttachment{}
		var mimetype sql.NullString
		if err := rows.Scan(&a.ID, &a.ItemType, &a.ItemID, &a.Title, &mimetype); err != nil {
			return nil, err
		}
		switch a.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			a.MimeType = mimetype.String
		}
		m.Attachments = append(m.Attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return m, nil
}

func (d *dataService) CreateTreatmentPlanScheduledMessage(msg *common.TreatmentPlanScheduledMessage) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	id, err := d.createTreatmentPlanScheduledMessage(tx, "treatment_plan", common.ClaimerTypeTreatmentPlanScheduledMessage, 0, msg)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit()
}

func (d *dataService) createTreatmentPlanScheduledMessage(tx *sql.Tx, tbl, claimerType string, id int64, msg *common.TreatmentPlanScheduledMessage) (int64, error) {
	if msg.TreatmentPlanID <= 0 {
		return 0, errors.New("missing TreatmentPlanID")
	}
	if msg.Message == "" {
		return 0, errors.New("missing Message")
	}
	if msg.ScheduledDays <= 0 {
		return 0, errors.New("missing ScheduledDays")
	}

	if id == 0 {
		res, err := tx.Exec(`
			INSERT INTO `+tbl+`_scheduled_message (
				`+tbl+`_id, scheduled_days, message)
			VALUES (?, ?, ?)`,
			msg.TreatmentPlanID, msg.ScheduledDays, msg.Message)
		if err != nil {
			return 0, err
		}
		msg.ID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		_, err := tx.Exec(`
			INSERT INTO `+tbl+`_scheduled_message (
				id, `+tbl+`_id, scheduled_days, message)
			VALUES (?, ?, ?, ?)`,
			id, msg.TreatmentPlanID, msg.ScheduledDays, msg.Message)
		if err != nil {
			return 0, err
		}
		msg.ID = id
	}

	for _, a := range msg.Attachments {
		_, err := tx.Exec(`
			INSERT INTO `+tbl+`_scheduled_message_attachment (
				`+tbl+`_scheduled_message_id, item_type, item_id, title)
			VALUES (?, ?, ?, ?)`, msg.ID, a.ItemType, a.ItemID, a.Title)
		if err != nil {
			return 0, err
		}
		switch a.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			if err := d.claimMedia(tx, a.ItemID, claimerType, msg.ID); err != nil {
				return 0, err
			}
		}
	}

	return msg.ID, nil
}

func (d *dataService) ListTreatmentPlanScheduledMessages(treatmentPlanID int64) ([]*common.TreatmentPlanScheduledMessage, error) {
	return d.listTreatmentPlanScheduledMessages("treatment_plan", treatmentPlanID)
}

func (d *dataService) listTreatmentPlanScheduledMessages(tbl string, treatmentPlanID int64) ([]*common.TreatmentPlanScheduledMessage, error) {
	var rows *sql.Rows
	var err error
	if tbl == "treatment_plan" {
		rows, err = d.db.Query(`
			SELECT id, scheduled_days, message, scheduled_message_id
			FROM treatment_plan_scheduled_message
			WHERE treatment_plan_id = ?
			ORDER BY scheduled_days`, treatmentPlanID)
	} else {
		rows, err = d.db.Query(`
			SELECT id, scheduled_days, message, NULL
			FROM dr_favorite_treatment_plan_scheduled_message
			WHERE dr_favorite_treatment_plan_id = ?
			ORDER BY scheduled_days`, treatmentPlanID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*common.TreatmentPlanScheduledMessage
	var msgIDs []interface{}
	msgMap := map[int64]*common.TreatmentPlanScheduledMessage{}
	for rows.Next() {
		m := &common.TreatmentPlanScheduledMessage{
			TreatmentPlanID: treatmentPlanID,
		}
		if err := rows.Scan(&m.ID, &m.ScheduledDays, &m.Message, &m.ScheduledMessageID); err != nil {
			return nil, err
		}
		msgMap[m.ID] = m
		msgIDs = append(msgIDs, m.ID)
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attachments

	if len(msgIDs) > 0 {
		rows, err := d.db.Query(fmt.Sprintf(`
			SELECT a.id, item_type, item_id, title, `+tbl+`_scheduled_message_id, mimetype
			FROM `+tbl+`_scheduled_message_attachment a
			LEFT OUTER JOIN media m ON m.id = item_id
			WHERE `+tbl+`_scheduled_message_id IN (%s)`, dbutil.MySQLArgs(len(msgIDs))),
			msgIDs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var mid int64
			a := &common.CaseMessageAttachment{}
			var mimetype sql.NullString
			if err := rows.Scan(&a.ID, &a.ItemType, &a.ItemID, &a.Title, &mid, &mimetype); err != nil {
				return nil, err
			}
			a.MimeType = mimetype.String
			msgMap[mid].Attachments = append(msgMap[mid].Attachments, a)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return msgs, nil
}

func (d *dataService) DeleteTreatmentPlanScheduledMessage(treatmentPlanID, messageID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.deleteTreatmentPlanScheduledMessage(tx, "treatment_plan",
		common.ClaimerTypeTreatmentPlanScheduledMessage, treatmentPlanID, messageID,
	); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *dataService) deleteTreatmentPlanScheduledMessage(tx *sql.Tx, tbl, claimerType string, treatmentPlanID, messageID int64) error {
	var smID *int64
	if tbl != "dr_favorite_treatment_plan" {
		if err := tx.QueryRow(
			`SELECT scheduled_message_id FROM `+tbl+`_scheduled_message WHERE id = ? AND `+tbl+`_id = ?`,
			messageID, treatmentPlanID,
		).Scan(&smID); err == sql.ErrNoRows {
			return ErrNotFound(tbl + "_scheduled_message")
		} else if err != nil {
			return err
		}
	}

	// Unclaim all attached media
	_, err := tx.Exec(`
		DELETE FROM media_claim
		WHERE claimer_type = ? AND claimer_id = ?`,
		claimerType, messageID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		DELETE FROM `+tbl+`_scheduled_message_attachment
		WHERE `+tbl+`_scheduled_message_id = ?`, messageID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		DELETE FROM `+tbl+`_scheduled_message
		WHERE id = ?`, messageID,
	); err != nil {
		return err
	}
	if smID != nil {
		return deleteScheduledMessage(tx, *smID)
	}
	return nil
}

func (d *dataService) ReplaceTreatmentPlanScheduledMessage(id int64, msg *common.TreatmentPlanScheduledMessage) error {
	if id <= 0 {
		return errors.New("message id required")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.deleteTreatmentPlanScheduledMessage(tx, "treatment_plan", common.ClaimerTypeTreatmentPlanScheduledMessage, msg.TreatmentPlanID, id); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := d.createTreatmentPlanScheduledMessage(tx, "treatment_plan", common.ClaimerTypeTreatmentPlanScheduledMessage, id, msg); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *dataService) UpdateTreatmentPlanScheduledMessage(id int64, update *TreatmentPlanScheduledMessageUpdate) error {
	args := dbutil.MySQLVarArgs()

	if update.ScheduledMessageID != nil {
		args.Append("scheduled_message_id", *update.ScheduledMessageID)
	}

	if update.Message != nil {
		args.Append("message", *update.Message)
	}

	_, err := d.db.Exec(`
		UPDATE treatment_plan_scheduled_message
		SET `+args.Columns()+` WHERE id = ?`,
		append(args.Values(), id)...)
	return err
}

func (d *dataService) listFavoriteTreatmentPlanScheduledMessages(ftpID int64) ([]*common.TreatmentPlanScheduledMessage, error) {
	return d.listTreatmentPlanScheduledMessages("dr_favorite_treatment_plan", ftpID)
}

func (d *dataService) SetFavoriteTreatmentPlanScheduledMessages(ftpID int64, msgs []*common.TreatmentPlanScheduledMessage) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := deleteFavoriteTreatmentPlanScheduledMessages(tx, ftpID); err != nil {
		tx.Rollback()
		return err
	}

	for _, m := range msgs {
		m.TreatmentPlanID = ftpID
		m.ID, err = d.createTreatmentPlanScheduledMessage(tx, "dr_favorite_treatment_plan",
			common.ClaimerTypeFavoriteTreatmentPlanScheduledMessage, 0, m)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *dataService) DeleteFavoriteTreatmentPlanScheduledMessages(ftpID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := deleteFavoriteTreatmentPlanScheduledMessages(tx, ftpID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func deleteFavoriteTreatmentPlanScheduledMessages(tx *sql.Tx, ftpID int64) error {
	rows, err := tx.Query(`
		SELECT id
		FROM dr_favorite_treatment_plan_scheduled_message
		WHERE dr_favorite_treatment_plan_id = ?`, ftpID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Leave room at the beginning for other params
	vals := []interface{}{nil}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		vals = append(vals, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(vals) <= 1 {
		return nil
	}
	replacements := dbutil.MySQLArgs(len(vals) - 1)

	// Unclaim all attached media
	vals[0] = common.ClaimerTypeFavoriteTreatmentPlanScheduledMessage
	if _, err := tx.Exec(`
		DELETE FROM media_claim
		WHERE claimer_type = ? AND claimer_id IN (`+replacements+`)`, vals...,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		DELETE FROM dr_favorite_treatment_plan_scheduled_message_attachment
		WHERE dr_favorite_treatment_plan_scheduled_message_id IN (`+replacements+`)`, vals[1:]...,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		DELETE FROM dr_favorite_treatment_plan_scheduled_message
		WHERE dr_favorite_treatment_plan_id = ?`, ftpID,
	); err != nil {
		return err
	}

	return nil
}
