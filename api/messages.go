package api

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) GetPeople(id []int64) (map[int64]*common.Person, error) {
	if len(id) == 0 {
		return map[int64]*common.Person{}, nil
	}
	rows, err := d.db.Query(fmt.Sprintf(`SELECT person.id, role_type_tag, role_id FROM person INNER JOIN role_type on role_type_id = role_type.id WHERE person.id IN (%s)`, dbutil.MySQLArgs(len(id))), dbutil.AppendInt64sToInterfaceSlice(nil, id)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	people := map[int64]*common.Person{}
	for rows.Next() {
		p := &common.Person{}
		if err := rows.Scan(&p.ID, &p.RoleType, &p.RoleID); err != nil {
			return nil, errors.Trace(err)
		}
		people[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}
	if err := d.populateDoctorOrPatientForPeople(people); err != nil {
		return nil, errors.Trace(err)
	}
	return people, nil
}

func (d *dataService) GetPersonIDByRole(roleType string, roleID int64) (int64, error) {
	var id int64
	err := d.db.QueryRow(
		`SELECT person.id FROM person WHERE role_type_id = ? AND role_id = ?`,
		d.roleTypeMapping[roleType], roleID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("person")
	}
	return id, errors.Trace(err)
}

func (d *dataService) CaseMessageForAttachment(itemType string, itemID, senderPersonID, patientCaseID int64) (*common.CaseMessage, error) {
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
		return nil, ErrNotFound("patient_case_message")
	} else if err != nil {
		return nil, err
	}

	// attachment
	var attachmentID int64
	err = d.db.QueryRow(`SELECT id from patient_case_message_attachment where message_id = ?`, message.ID).Scan(&attachmentID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_case_message_attachment")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	message.Attachments = []*common.CaseMessageAttachment{
		{
			ID:       attachmentID,
			ItemType: itemType,
			ItemID:   itemID,
		},
	}
	return &message, nil
}

func (d *dataService) CaseMessagesRead(messageIDs []int64, personID int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	reps := make([]byte, 0, 6*len(messageIDs)) // len("(?,?),") == 6
	vals := make([]interface{}, 0, 2*len(messageIDs))
	for _, mid := range messageIDs {
		reps = append(reps, "(?,?),"...)
		vals = append(vals, mid, personID)
	}
	reps = reps[:len(reps)-1] // -1 to remove trailing comma
	_, err := d.db.Exec(`
		INSERT IGNORE INTO patient_case_message_read (message_id, person_id)
		VALUES `+string(reps), vals...)
	return errors.Trace(err)
}

func (d *dataService) ListCaseMessages(caseID int64, opts ListCaseMessagesOption) ([]*common.CaseMessage, error) {
	var clause string

	if !opts.has(LCMOIncludePrivate) {
		clause = `AND private = 0`
	}

	rows, err := d.db.Query(`
		SELECT id, tstamp, person_id, body, private, event_text
		FROM patient_case_message
		WHERE patient_case_id = ? `+clause+` ORDER BY tstamp`, caseID)
	if err != nil {
		return nil, errors.Trace(err)
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
			return nil, errors.Trace(err)
		}
		messageMap[m.ID] = m
		messageIDs = append(messageIDs, m.ID)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	if len(messages) == 0 {
		return messages, nil
	}

	// Attachments

	rows, err = d.db.Query(fmt.Sprintf(`
			SELECT a.id, item_type, item_id, message_id, title, mimetype
			FROM patient_case_message_attachment a
			LEFT OUTER JOIN media m ON m.id = a.item_id AND a.item_type IN ('photo', 'audio')
			WHERE message_id IN (%s)`, dbutil.MySQLArgs(len(messageIDs))),
		messageIDs...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid int64
		var mimetype sql.NullString
		a := &common.CaseMessageAttachment{}
		if err := rows.Scan(&a.ID, &a.ItemType, &a.ItemID, &mid, &a.Title, &mimetype); err != nil {
			return nil, errors.Trace(err)
		}
		a.MimeType = mimetype.String
		messageMap[mid].Attachments = append(messageMap[mid].Attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	// Read receipts

	if opts.has(LCMOIncludeReadReceipts) {
		receipts, err := d.caseMessageReadReceipts(messageIDs)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for mid, rr := range receipts {
			messageMap[mid].ReadReceipts = rr
		}
	}
	return messages, nil
}

func (d *dataService) caseMessageReadReceipts(msgIDs []interface{}) (map[int64][]*common.ReadReceipt, error) {
	rows, err := d.db.Query(`
		SELECT "message_id", "person_id", "timestamp"
		FROM "patient_case_message_read"
		WHERE "message_id" IN (`+dbutil.MySQLArgs(len(msgIDs))+`)`,
		msgIDs...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	receipts := make(map[int64][]*common.ReadReceipt, len(msgIDs))
	for rows.Next() {
		var mid int64
		rr := &common.ReadReceipt{}
		if err := rows.Scan(&mid, &rr.PersonID, &rr.Time); err != nil {
			return nil, errors.Trace(err)
		}
		receipts[mid] = append(receipts[mid], rr)
	}
	return receipts, errors.Trace(rows.Err())
}

func (d *dataService) GetCaseIDFromMessageID(messageID int64) (int64, error) {
	var caseID int64
	err := d.db.QueryRow(`SELECT patient_case_id FROM patient_case_message WHERE id = ?`, messageID).Scan(&caseID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("patient_case_message")
	}
	return caseID, errors.Trace(err)
}

func (d *dataService) CreateCaseMessage(msg *common.CaseMessage) (int64, error) {
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
		return 0, errors.Trace(err)
	}

	res, err := tx.Exec(`
		INSERT INTO patient_case_message (tstamp, person_id, body, patient_case_id, private, event_text)
		VALUES (?, ?, ?, ?, ?, ?)`, msg.Time, msg.PersonID, msg.Body, msg.CaseID, msg.IsPrivate, msg.EventText)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}
	msg.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	for _, a := range msg.Attachments {
		_, err := tx.Exec(`
			INSERT INTO patient_case_message_attachment (message_id, item_type, item_id, title)
			VALUES (?, ?, ?, ?)`, msg.ID, a.ItemType, a.ItemID, a.Title)
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
		switch a.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			if err := d.claimMedia(tx, a.ItemID, common.ClaimerTypeConversationMessage, msg.ID); err != nil {
				tx.Rollback()
				return 0, errors.Trace(err)
			}
		}
	}

	_, err = tx.Exec(`
		REPLACE INTO patient_case_message_participant (patient_case_id, person_id)
		VALUES (?, ?)`,
		msg.CaseID, msg.PersonID)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	// Mark the posted message as read by the person that posted it as hopefully they did.
	_, err = d.db.Exec(`
		INSERT INTO patient_case_message_read (message_id, person_id)
		VALUES (?, ?)`, msg.ID, msg.PersonID)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	return msg.ID, errors.Trace(tx.Commit())
}

func (d *dataService) CaseMessageParticipants(caseID int64, withRoleObjects bool) (map[int64]*common.CaseMessageParticipant, error) {
	rows, err := d.db.Query(`
		SELECT person_id, role_type_tag, role_id
		FROM patient_case_message_participant
		INNER JOIN person ON person.id = person_id
		INNER JOIN role_type ON role_type.id = role_type_id
		WHERE patient_case_id = ?`, caseID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	participants := make(map[int64]*common.CaseMessageParticipant)
	for rows.Next() {
		p := &common.CaseMessageParticipant{
			CaseID: caseID,
			Person: &common.Person{},
		}
		if err := rows.Scan(&p.Person.ID, &p.Person.RoleType, &p.Person.RoleID); err != nil {
			return nil, errors.Trace(err)
		}
		participants[p.Person.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	if withRoleObjects {
		for _, p := range participants {
			var err error
			switch p.Person.RoleType {
			case RolePatient:
				p.Person.Patient, err = d.GetPatientFromID(p.Person.RoleID)
			case RoleDoctor, RoleCC:
				p.Person.Doctor, err = d.GetDoctorFromID(p.Person.RoleID)
			}
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return participants, nil
}

func (d *dataService) UnreadMessageCount(caseID, personID int64) (int, error) {
	row := d.db.QueryRow(`
		SELECT count(1)
		FROM patient_case_message cm
		LEFT JOIN patient_case_message_read cmr ON cmr.message_id = cm.id AND cmr.person_id = ?
		WHERE cm.patient_case_id = ?
			AND cmr.message_id IS NULL
			AND cm.person_id != ?`, personID, caseID, personID)
	var count int
	err := row.Scan(&count)
	return count, err
}

func (d *dataService) populateDoctorOrPatientForPeople(people map[int64]*common.Person) error {
	for _, p := range people {
		var err error
		switch p.RoleType {
		case RolePatient:
			p.Patient, err = d.GetPatientFromID(p.RoleID)
		case RoleDoctor, RoleCC:
			p.Doctor, err = d.GetDoctorFromID(p.RoleID)
		}
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
