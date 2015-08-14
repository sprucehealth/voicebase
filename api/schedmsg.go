package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) CreateScheduledMessage(msg *common.ScheduledMessage) (int64, error) {
	return createScheduledMessage(d.db, msg)
}

func createScheduledMessage(db db, msg *common.ScheduledMessage) (int64, error) {
	if msg.Message == nil {
		return 0, errors.Trace(errors.New("missing Message"))
	}
	if msg.Status.String() == "" {
		return 0, errors.Trace(errors.New("missing Status"))
	}

	jsonData, err := json.Marshal(msg.Message)
	if err != nil {
		return 0, errors.Trace(err)
	}
	res, err := db.Exec(`
		INSERT INTO scheduled_message
		(patient_id, message_type, message_json, event, scheduled, status)
		VALUES (?,?,?,?,?,?)`, msg.PatientID, msg.Message.TypeName(), jsonData, msg.Event, msg.Scheduled, msg.Status.String())
	if err != nil {
		return 0, errors.Trace(err)
	}
	msg.ID, err = res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}
	return msg.ID, nil
}

func deleteScheduledMessage(db db, id int64) error {
	_, err := db.Exec(`DELETE FROM scheduled_message WHERE id = ?`, id)
	return errors.Trace(err)
}

func (d *dataService) ScheduledMessage(id int64, messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error) {
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
		return nil, errors.Trace(ErrNotFound("scheduled_message"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	if msgJSON != nil {
		msgDataType, ok := messageTypes[msgType]
		if !ok {
			return nil, fmt.Errorf("Unable to find message type to render data: %s", msgType)
		}
		scheduledMsg.Message = reflect.New(msgDataType).Interface().(common.Typed)
		if err := json.Unmarshal(msgJSON, &scheduledMsg.Message); err != nil {
			return nil, errors.Trace(err)
		}
	}

	scheduledMsg.Error = errString.String
	return &scheduledMsg, nil
}

func (d *dataService) CreateScheduledMessageTemplate(template *common.ScheduledMessageTemplate) error {
	if template == nil {
		return errors.Trace(errors.New("No scheduled message template specified"))
	}

	_, err := d.db.Exec(`
		INSERT INTO scheduled_message_template
		(name, event, schedule_period, message)
		VALUES (?,?,?,?)`,
		template.Name, template.Event, template.SchedulePeriod, template.Message)
	return errors.Trace(err)
}

func (d *dataService) UpdateScheduledMessageTemplate(template *common.ScheduledMessageTemplate) error {
	if template == nil {
		return errors.New("No scheduled message template specified")
	}

	_, err := d.db.Exec(`
		REPLACE INTO scheduled_message_template
		(id, name, event, schedule_period, message)
		VALUES (?,?,?,?,?)`,
		template.ID, template.Name, template.Event, template.SchedulePeriod, template.Message)
	return errors.Trace(err)
}

func (d *dataService) ScheduledMessageTemplate(id int64) (*common.ScheduledMessageTemplate, error) {
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
		return nil, errors.Trace(ErrNotFound("scheduled_message_template"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &scheduledMessageTemplate, nil
}

func (d *dataService) ListScheduledMessageTemplates() ([]*common.ScheduledMessageTemplate, error) {
	rows, err := d.db.Query(`
		SELECT id, name, event, message, schedule_period, created
		FROM scheduled_message_template`)
	if err != nil {
		return nil, errors.Trace(err)
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
			return nil, errors.Trace(err)
		}
		scheduledMessageTemplates = append(scheduledMessageTemplates, &sMessageTemplate)
	}

	return scheduledMessageTemplates, rows.Err()
}

func (d *dataService) DeleteScheduledMessageTemplate(id int64) error {
	_, err := d.db.Exec(`DELETE FROM scheduled_message_template WHERE id = ?`, id)
	return errors.Trace(err)
}

func (d *dataService) ScheduledMessageTemplates(eventType string) ([]*common.ScheduledMessageTemplate, error) {
	var scheduledMessageTemplates []*common.ScheduledMessageTemplate
	rows, err := d.db.Query(`
		SELECT id, name, event, schedule_period, message, created
		FROM scheduled_message_template
		WHERE event = ?`, eventType)
	if err != nil {
		return nil, errors.Trace(err)
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
			return nil, errors.Trace(err)
		}
		scheduledMessageTemplates = append(scheduledMessageTemplates, &sMessageTemplate)
	}

	return scheduledMessageTemplates, rows.Err()
}

func (d *dataService) RandomlyPickAndStartProcessingScheduledMessage(messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, errors.Trace(err)
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
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	elligibleMessageIds := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return nil, errors.Trace(err)
		}
		elligibleMessageIds = append(elligibleMessageIds, id)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}

	// nothing to do if there are no elligibile messages
	if len(elligibleMessageIds) == 0 {
		tx.Rollback()
		return nil, errors.Trace(ErrNotFound("scheduled_message"))
	}

	// pick a random id to work on
	msgID := elligibleMessageIds[rand.Intn(len(elligibleMessageIds))]

	// attempt to pick this message for processing by updating the status of the message
	// only if it currently exists in the scheduled state
	_, err = tx.Exec(`
		UPDATE scheduled_message SET status = ?
		WHERE status = ? AND id = ?`,
		common.SMProcessing.String(),
		common.SMScheduled.String(), msgID)
	if err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Trace(err)
	}

	return d.ScheduledMessage(msgID, messageTypes)
}

func (d *dataService) UpdateScheduledMessage(id int64, status common.ScheduledMessageStatus) error {
	_, err := d.db.Exec(`UPDATE scheduled_message SET status = ? WHERE id = ?`, status.String(), id)
	return errors.Trace(err)
}

// DeactivateScheduledMessagesForPatient moves all scheduled messages that map to the provided patient id to the DEACTIVATED state and returns the number of rows affected
func (d *dataService) DeactivateScheduledMessagesForPatient(patientID common.PatientID) (int64, error) {
	args := dbutil.MySQLVarArgs()
	args.Append(`status`, common.SMDeactivated.String())
	res, err := d.db.Exec(`
		UPDATE scheduled_message SET `+args.Columns()+`
		WHERE patient_id = ?
		AND status = (?)`, append(args.Values(), patientID, common.SMScheduled.String())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}
