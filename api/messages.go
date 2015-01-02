package api

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) GetPeople(id []int64) (map[int64]*common.Person, error) {
	if len(id) == 0 {
		return map[int64]*common.Person{}, nil
	}

	rows, err := d.db.Query(fmt.Sprintf(`SELECT person.id, role_type_tag, role_id FROM person INNER JOIN role_type on role_type_id = role_type.id WHERE person.id IN (%s)`, dbutil.MySQLArgs(len(id))), dbutil.AppendInt64sToInterfaceSlice(nil, id)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people := map[int64]*common.Person{}
	for rows.Next() {
		p := &common.Person{}
		if err := rows.Scan(&p.ID, &p.RoleType, &p.RoleID); err != nil {
			return nil, err
		}
		people[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := d.populateDoctorOrPatientForPeople(people); err != nil {
		return nil, err
	}
	return people, nil
}

func (d *DataService) GetPersonIDByRole(roleType string, roleID int64) (int64, error) {
	var id int64
	err := d.db.QueryRow(
		`SELECT person.id FROM person WHERE role_type_id = ? AND role_id = ?`,
		d.roleTypeMapping[roleType], roleID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return id, err
}

func (d *DataService) CaseMessageForAttachment(itemType string, itemID, senderPersonID, patientCaseID int64) (*common.CaseMessage, error) {
	var message common.CaseMessage
	err := d.db.QueryRow(`
		SELECT patient_case_message.id, patient_case_message.patient_case_id, tstamp, person_id, body, private, event_text
		FROM patient_case_message
		INNER JOIN patient_case_message_attachment on patient_case_message_attachment.message_id = patient_case_message.id
		WHERE patient_case_id = ? AND item_type = ? AND item_id = ? AND person_id = ?`, patientCaseID, itemType, itemID, senderPersonID).Scan(
		&message.ID,
		&message.CaseID,
		&message.Time,
		&message.PersonID,
		&message.Body,
		&message.IsPrivate,
		&message.EventText)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	// attachment
	var attachmentID int64
	err = d.db.QueryRow(`SELECT id from patient_case_message_attachment where message_id = ?`, message.ID).Scan(&attachmentID)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	message.Attachments = []*common.CaseMessageAttachment{
		&common.CaseMessageAttachment{
			ID:       attachmentID,
			ItemType: itemType,
			ItemID:   itemID,
		},
	}
	return &message, nil
}

func (d *DataService) ListCaseMessages(caseID int64, role string) ([]*common.CaseMessage, error) {
	var clause string
	// private messages should only be returned to the doctor or ma
	if role != DOCTOR_ROLE && role != MA_ROLE {
		clause = `AND private = 0`
	}

	rows, err := d.db.Query(`
		SELECT id, tstamp, person_id, body, private, event_text
		FROM patient_case_message
		WHERE patient_case_id = ? `+clause+` ORDER BY tstamp`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*common.CaseMessage
	var messageIDs []interface{}
	messageMap := map[int64]*common.CaseMessage{}
	for rows.Next() {
		m := &common.CaseMessage{
			CaseID: caseID,
		}
		if err := rows.Scan(&m.ID, &m.Time, &m.PersonID, &m.Body, &m.IsPrivate, &m.EventText); err != nil {
			return nil, err
		}
		messageMap[m.ID] = m
		messageIDs = append(messageIDs, m.ID)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attachments

	if len(messageIDs) > 0 {
		rows, err := d.db.Query(fmt.Sprintf(`
			SELECT id, item_type, item_id, message_id, title
			FROM patient_case_message_attachment
			WHERE message_id IN (%s)`, dbutil.MySQLArgs(len(messageIDs))),
			messageIDs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var mid int64
			a := &common.CaseMessageAttachment{}
			if err := rows.Scan(&a.ID, &a.ItemType, &a.ItemID, &mid, &a.Title); err != nil {
				return nil, err
			}
			switch a.ItemType {
			case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
				// If it's a media item, find the mimetype
				if err := d.db.QueryRow(`
					SELECT mimetype
					FROM media
					WHERE id = ?`, a.ItemID,
				).Scan(&a.MimeType); err == sql.ErrNoRows {
					return nil, NoRowsError
				} else if err != nil {
					return nil, err
				}

			}

			messageMap[mid].Attachments = append(messageMap[mid].Attachments, a)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return messages, nil
}

func (d *DataService) GetCaseIDFromMessageID(messageID int64) (int64, error) {
	var caseID int64
	err := d.db.QueryRow(`select patient_case_id from patient_case_message where id = ?`, messageID).Scan(&caseID)

	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return caseID, err
}

func (d *DataService) CreateCaseMessage(msg *common.CaseMessage) (int64, error) {
	if msg.CaseID <= 0 {
		return 0, errors.New("api.CreateCaseMessage: missing CaseID")
	}
	if msg.PersonID <= 0 {
		return 0, errors.New("api.CreateCaseMessage: missing PersonID")
	}
	if msg.Body == "" {
		return 0, errors.New("api.CreateCaseMessage: empty body")
	}
	if msg.Time.IsZero() {
		msg.Time = time.Now()
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		INSERT INTO patient_case_message (tstamp, person_id, body, patient_case_id, private, event_text)
		VALUES (?, ?, ?, ?, ?, ?)`, msg.Time, msg.PersonID, msg.Body, msg.CaseID, msg.IsPrivate, msg.EventText)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	msg.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, a := range msg.Attachments {
		_, err := tx.Exec(`
			INSERT INTO patient_case_message_attachment (message_id, item_type, item_id, title)
			VALUES (?, ?, ?, ?)`, msg.ID, a.ItemType, a.ItemID, a.Title)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		switch a.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			if err := d.claimMedia(tx, a.ItemID, common.ClaimerTypeConversationMessage, msg.ID); err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	_, err = tx.Exec(`
		REPLACE INTO patient_case_message_participant (patient_case_id, person_id, unread, last_read_tstamp)
		VALUES (?, ?, ?, ?)`,
		msg.CaseID, msg.PersonID, false, time.Now())
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Mark the conversation as unread for all participants except the one that just posted
	_, err = tx.Exec(`UPDATE patient_case_message_participant SET unread = true WHERE person_id != ?`, msg.PersonID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return msg.ID, tx.Commit()
}

func (d *DataService) CaseMessageParticipants(caseID int64, withRoleObjects bool) (map[int64]*common.CaseMessageParticipant, error) {
	rows, err := d.db.Query(`
		SELECT person_id, unread, last_read_tstamp, role_type_tag, role_id
		FROM patient_case_message_participant
		INNER JOIN person ON person.id = person_id
		INNER JOIN role_type ON role_type.id = role_type_id
		WHERE patient_case_id = ?`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make(map[int64]*common.CaseMessageParticipant)
	for rows.Next() {
		p := &common.CaseMessageParticipant{
			CaseID: caseID,
			Person: &common.Person{},
		}
		if err := rows.Scan(&p.Person.ID, &p.Unread, &p.LastRead, &p.Person.RoleType, &p.Person.RoleID); err != nil {
			return nil, err
		}
		participants[p.Person.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if withRoleObjects {
		for _, p := range participants {
			var err error
			switch p.Person.RoleType {
			case PATIENT_ROLE:
				p.Person.Patient, err = d.GetPatientFromID(p.Person.RoleID)
			case DOCTOR_ROLE, MA_ROLE:
				p.Person.Doctor, err = d.GetDoctorFromID(p.Person.RoleID)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	return participants, nil
}

func (d *DataService) MarkCaseMessagesAsRead(caseID, personID int64) error {
	_, err := d.db.Exec(`
		REPLACE INTO patient_case_message_participant (patient_case_id, person_id, unread, last_read_tstamp)
		VALUES (?, ?, ?, ?)`,
		caseID, personID, false, time.Now())
	return err
}

func (d *DataService) populateDoctorOrPatientForPeople(people map[int64]*common.Person) error {
	for _, p := range people {
		var err error
		switch p.RoleType {
		case PATIENT_ROLE:
			p.Patient, err = d.GetPatientFromID(p.RoleID)
		case DOCTOR_ROLE, MA_ROLE:
			p.Doctor, err = d.GetDoctorFromID(p.RoleID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
