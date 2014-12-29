package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) CreateScheduledMessage(msg *common.ScheduledMessage) (int64, error) {
	return createScheduledMessage(d.db, msg)
}

func createScheduledMessage(db db, msg *common.ScheduledMessage) (int64, error) {
	if msg.Message == nil {
		return 0, errors.New("missing Message")
	}
	if msg.Status.String() == "" {
		return 0, errors.New("missing Status")
	}

	jsonData, err := json.Marshal(msg.Message)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec(`
		INSERT INTO scheduled_message
		(patient_id, message_type, message_json, event, scheduled, status)
		VALUES (?,?,?,?,?,?)`, msg.PatientID, msg.Message.TypeName(), jsonData, msg.Event, msg.Scheduled, msg.Status.String())
	if err != nil {
		return 0, err
	}
	msg.ID, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return msg.ID, nil
}

func deleteScheduledMessage(db db, id int64) error {
	_, err := db.Exec(`DELETE FROM scheduled_message WHERE id = ?`, id)
	return err
}

func (d *DataService) ScheduledMessage(id int64, messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error) {
	var scheduledMsg common.ScheduledMessage
	var msgType string
	var msgJSON []byte
	var errString sql.NullString
	if err := d.db.QueryRow(`
		SELECT id, patient_id, message_type, message_json, event, status,
			created, scheduled, completed, error
		FROM scheduled_message
		WHERE id = ?`, id).Scan(
		&scheduledMsg.ID,
		&scheduledMsg.PatientID,
		&msgType,
		&msgJSON,
		&scheduledMsg.Event,
		&scheduledMsg.Status,
		&scheduledMsg.Created,
		&scheduledMsg.Scheduled,
		&scheduledMsg.Completed,
		&errString); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	if msgJSON != nil {
		msgDataType, ok := messageTypes[msgType]
		if !ok {
			return nil, fmt.Errorf("Unable to find message type to render data: %s", msgType)
		}
		scheduledMsg.Message = reflect.New(msgDataType).Interface().(common.Typed)
		if err := json.Unmarshal(msgJSON, &scheduledMsg.Message); err != nil {
			return nil, err
		}
	}

	scheduledMsg.Error = errString.String
	return &scheduledMsg, nil
}

func (d *DataService) CreateScheduledMessageTemplate(template *common.ScheduledMessageTemplate) error {
	if template == nil {
		return errors.New("No scheduled message template specified")
	}

	_, err := d.db.Exec(`
		INSERT INTO scheduled_message_template
		(name, event, schedule_period, message)
		VALUES (?,?,?,?)`,
		template.Name, template.Event, template.SchedulePeriod, template.Message)
	return err
}

func (d *DataService) UpdateScheduledMessageTemplate(template *common.ScheduledMessageTemplate) error {
	if template == nil {
		return errors.New("No scheduled message template specified")
	}

	_, err := d.db.Exec(`
		REPLACE INTO scheduled_message_template
		(id, name, event, schedule_period, message)
		VALUES (?,?,?,?,?)`,
		template.ID, template.Name, template.Event, template.SchedulePeriod, template.Message)
	return err
}

func (d *DataService) ScheduledMessageTemplate(id int64) (*common.ScheduledMessageTemplate, error) {
	var scheduledMessageTemplate common.ScheduledMessageTemplate
	err := d.db.QueryRow(`
		SELECT id, name, event, message, schedule_period, created
		FROM scheduled_message_template
		WHERE id = ?`, id).Scan(
		&scheduledMessageTemplate.ID,
		&scheduledMessageTemplate.Name,
		&scheduledMessageTemplate.Event,
		&scheduledMessageTemplate.Message,
		&scheduledMessageTemplate.SchedulePeriod,
		&scheduledMessageTemplate.Created,
	)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	return &scheduledMessageTemplate, nil
}

func (d *DataService) ListScheduledMessageTemplates() ([]*common.ScheduledMessageTemplate, error) {
	rows, err := d.db.Query(`
		SELECT id, name, event, message, scheduled_period, created
		FROM scheduled_message_template`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scheduledMessageTemplates []*common.ScheduledMessageTemplate
	for rows.Next() {
		var sMessageTemplate common.ScheduledMessageTemplate
		if err := rows.Scan(
			&sMessageTemplate.ID,
			&sMessageTemplate.Name,
			&sMessageTemplate.Event,
			&sMessageTemplate.Message,
			&sMessageTemplate.SchedulePeriod,
			&sMessageTemplate.Created); err != nil {
			return nil, err
		}
		scheduledMessageTemplates = append(scheduledMessageTemplates, &sMessageTemplate)
	}

	return scheduledMessageTemplates, rows.Err()
}

func (d *DataService) DeleteScheduledMessageTemplate(id int64) error {
	_, err := d.db.Exec(`DELETE FROM scheduled_message_template WHERE id = ?`, id)
	return err
}

func (d *DataService) ScheduledMessageTemplates(eventType string) ([]*common.ScheduledMessageTemplate, error) {
	var scheduledMessageTemplates []*common.ScheduledMessageTemplate
	rows, err := d.db.Query(`
		SELECT id, name, event, schedule_period, message, created
		FROM scheduled_message_template
		WHERE event = ?`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sMessageTemplate common.ScheduledMessageTemplate
		if err := rows.Scan(
			&sMessageTemplate.ID,
			&sMessageTemplate.Name,
			&sMessageTemplate.Event,
			&sMessageTemplate.SchedulePeriod,
			&sMessageTemplate.Message,
			&sMessageTemplate.Created); err != nil {
			return nil, err
		}
		scheduledMessageTemplates = append(scheduledMessageTemplates, &sMessageTemplate)
	}

	return scheduledMessageTemplates, rows.Err()
}

func (d *DataService) RandomlyPickAndStartProcessingScheduledMessage(messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	limit := 10
	rows, err := tx.Query(`
		SELECT id
		FROM scheduled_message
		WHERE status = ? AND scheduled <= ?
		LIMIT ?
		FOR UPDATE`,
		common.SMScheduled.String(), time.Now().UTC(), limit)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer rows.Close()

	elligibleMessageIds := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return nil, err
		}
		elligibleMessageIds = append(elligibleMessageIds, id)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, err
	}

	// nothing to do if there are no elligibile messages
	if len(elligibleMessageIds) == 0 {
		tx.Rollback()
		return nil, NoRowsError
	}

	// pick a random id to work on
	msgId := elligibleMessageIds[rand.Intn(len(elligibleMessageIds))]

	// attempt to pick this message for processing by updating the status of the message
	// only if it currently exists in the scheduled state
	_, err = tx.Exec(`
		UPDATE scheduled_message SET status = ?
		WHERE status = ? AND id = ?`,
		common.SMProcessing.String(),
		common.SMScheduled.String(), msgId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.ScheduledMessage(msgId, messageTypes)
}

func (d *DataService) UpdateScheduledMessage(id int64, status common.ScheduledMessageStatus) error {
	_, err := d.db.Exec(`UPDATE scheduled_message SET status = ? WHERE id = ?`, status.String(), id)
	return err
}
