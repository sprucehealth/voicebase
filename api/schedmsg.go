package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) CreateScheduledMessage(msg *common.ScheduledMessage) error {
	jsonData, err := json.Marshal(msg.MessageJSON)
	if err != nil {
		return err
	}
	res, err := d.db.Exec(`
		INSERT INTO scheduled_message
		(patient_id, message_type, message_json, type, scheduled, status)
		VALUES (?,?,?,?,?,?)`, msg.PatientID, msg.MessageType, jsonData, msg.Type, msg.Scheduled, msg.Status.String())
	msg.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) ScheduledMessage(id int64, messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error) {
	var scheduledMsg common.ScheduledMessage
	var msgJSON []byte
	var errString sql.NullString
	if err := d.db.QueryRow(`
		SELECT id, patient_id, message_type, message_json, type, status, 
			created, scheduled, completed, error
		FROM scheduled_message
		WHERE id = ?`, id).Scan(
		&scheduledMsg.ID,
		&scheduledMsg.PatientID,
		&scheduledMsg.MessageType,
		&msgJSON,
		&scheduledMsg.Type,
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
		msgDataType, ok := messageTypes[scheduledMsg.MessageType]
		if !ok {
			return nil, fmt.Errorf("Unable to find message type to render data: %s", scheduledMsg.MessageType)
		}
		scheduledMsg.MessageJSON = reflect.New(msgDataType).Interface().(common.Typed)
		if err := json.Unmarshal(msgJSON, &scheduledMsg.MessageJSON); err != nil {
			return nil, err
		}
	}

	scheduledMsg.Error = errString.String
	return &scheduledMsg, nil
}

func (d *DataService) ScheduledMessageTemplates(eventType string, messageTypes map[string]reflect.Type) ([]*common.ScheduledMessageTemplate, error) {
	var scheduledMessageTemplates []*common.ScheduledMessageTemplate
	rows, err := d.db.Query(`
		SELECT id, type, schedule_period, message_template_type, 
		message_template_json, creator_account_id, created
		FROM scheduled_message_template
		WHERE type = ?`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sMessageTemplate common.ScheduledMessageTemplate
		var messageJSON []byte
		if err := rows.Scan(
			&sMessageTemplate.ID,
			&sMessageTemplate.Type,
			&sMessageTemplate.SchedulePeriod,
			&sMessageTemplate.MessageType,
			&messageJSON,
			&sMessageTemplate.CreatorAccountID,
			&sMessageTemplate.Created); err != nil {
			return nil, err
		}
		if messageJSON != nil {
			msgDataType, ok := messageTypes[sMessageTemplate.MessageType]
			if !ok {
				return nil, fmt.Errorf("Unable to find message type to render data: %s", common.SMCaseMessageType)
			}
			sMessageTemplate.MessageJSON = reflect.New(msgDataType).Interface().(common.Typed)
			if err := json.Unmarshal(messageJSON, &sMessageTemplate.MessageJSON); err != nil {
				return nil, err
			}
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
	rows, err := tx.Query(`SELECT id FROM scheduled_message WHERE status = ? AND scheduled < now() LIMIT ?`, common.SMScheduled.String(), limit)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	elligibileMessageIds := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return nil, err
		}
		elligibileMessageIds = append(elligibileMessageIds, id)
	}

	// attempt to pick a random msg for processing with a maximum of 3 retries
	for i := 0; i < 3; i++ {
		// pick a random id to work on
		msgId := elligibileMessageIds[rand.Intn(len(elligibileMessageIds))]

		// attempt to pick this message for processing by updating the status of the message
		// only if it currently exists in the scheduled state
		res, err := tx.Exec(`
			UPDATE scheduled_message SET status = ? 
			WHERE status = ? AND id = ?`, common.SMProcessing.String(), common.SMScheduled.String(), msgId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		if rowsAffected > 0 {
			if err := tx.Commit(); err != nil {
				tx.Rollback()
				return nil, err
			}
			return d.ScheduledMessage(msgId, messageTypes)
		}
	}

	return nil, NoRowsError
}

func (d *DataService) UpdateScheduledMessage(id int64, status common.ScheduledMessageStatus) error {
	_, err := d.db.Exec(`update scheduled_message set status = ? where id = ?`, status.String(), id)
	return err
}
