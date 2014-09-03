package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
