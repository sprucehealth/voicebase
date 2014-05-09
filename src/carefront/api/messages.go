package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"time"
)

func (d *DataService) GetPeople(id []int64) (map[int64]*common.Person, error) {
	rows, err := d.DB.Query(fmt.Sprintf(`SELECT id, role_type, role_id FROM person WHERE ID IN (%s)`, nReplacements(len(id))), appendInt64sToInterfaceSlice(nil, id)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people := map[int64]*common.Person{}
	for rows.Next() {
		p := &common.Person{}
		if err := rows.Scan(&p.Id, &p.RoleType, &p.RoleId); err != nil {
			return nil, err
		}
		people[p.Id] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := d.populateDoctorOrPatientForPeople(people); err != nil {
		return nil, err
	}
	return people, nil
}

func (d *DataService) GetPersonIdByRole(roleType string, roleId int64) (int64, error) {
	var id int64
	err := d.DB.QueryRow(
		`SELECT id FROM person WHERE role_type = ? AND role_id = ?`,
		roleType, roleId).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return id, err
}

func (d *DataService) GetConversationsWithParticipants(ids []int64) ([]*common.Conversation, map[int64]*common.Person, error) {
	// Find the intersection of the sets of conversation_id for all of the participants.
	// This gives us the conversations that include ALL of the participants. The options
	// for doing this include using an intersection of queries (not available in MySQL),
	// a join for each participant, group by on a sub-query, or doing the group by in code.
	// The last option was chosen here. The query pulls down all conversation_id for each
	// participant, and if a conversation includes all of the participants then the number
	// of times the conversation_id appears in the results (# of rows) should equal the
	// number of participants.
	idvals := appendInt64sToInterfaceSlice(nil, ids)
	rows, err := d.DB.Query(
		fmt.Sprintf(`SELECT conversation_id FROM conversation_participant WHERE person_id IN (%s)`, nReplacements(len(ids))),
		idvals...)
	if err != nil {
		return nil, nil, err
	}
	cidCount := map[int64]int{}
	idvals = idvals[:0]
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		cidCount[id]++
		if cidCount[id] == len(ids) {
			idvals = append(idvals, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(idvals) == 0 {
		return []*common.Conversation{}, nil, nil
	}

	rows, err = d.DB.Query(fmt.Sprintf(`
		SELECT id, tstamp, topic_id, (SELECT title FROM conversation_topic WHERE id = topic_id),
			message_count, creator_id, owner_id, last_participant_id, last_message_tstamp, unread
		FROM conversation
		WHERE id IN (%s)
		ORDER BY tstamp`, nReplacements(len(idvals))), idvals...)
	if err != nil {
		return nil, nil, err
	}
	var convos []*common.Conversation
	personIdSet := map[int64]bool{}
	for rows.Next() {
		c := &common.Conversation{}
		if err := rows.Scan(
			&c.Id, &c.Time, &c.TopicId, &c.Title, &c.MessageCount, &c.CreatorId,
			&c.OwnerId, &c.LastParticipantId, &c.LastMessageTime, &c.Unread,
		); err != nil {
			return nil, nil, err
		}
		personIdSet[c.LastParticipantId] = true
		personIdSet[c.CreatorId] = true
		personIdSet[c.OwnerId] = true
		convos = append(convos, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	personIds := make([]int64, 0, len(personIdSet))
	for id := range personIdSet {
		personIds = append(personIds, id)
	}
	participants, err := d.GetPeople(personIds)
	if err != nil {
		return nil, nil, err
	}

	return convos, participants, nil
}

func (d *DataService) GetConversation(id int64) (*common.Conversation, error) {
	c := &common.Conversation{
		Id: id,
	}
	row := d.DB.QueryRow(`
		SELECT tstamp, topic_id, (SELECT title FROM conversation_topic WHERE id = topic_id),
			message_count, creator_id, owner_id, last_participant_id, last_message_tstamp, unread
		FROM conversation
		WHERE id = ?`, id)
	if err := row.Scan(
		&c.Time, &c.TopicId, &c.Title, &c.MessageCount, &c.CreatorId,
		&c.OwnerId, &c.LastParticipantId, &c.LastMessageTime, &c.Unread,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	// Messages

	rows, err := d.DB.Query(`SELECT id, tstamp, person_id, body FROM conversation_message WHERE conversation_id = ? ORDER BY tstamp`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []*common.ConversationMessage
	var messageIds []interface{}
	messageMap := map[int64]*common.ConversationMessage{}
	for rows.Next() {
		m := &common.ConversationMessage{
			ConversationId: id,
		}
		if err := rows.Scan(&m.Id, &m.Time, &m.FromId, &m.Body); err != nil {
			return nil, err
		}
		messageMap[m.Id] = m
		messageIds = append(messageIds, m.Id)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	c.Messages = messages

	// Attachments

	if len(messageIds) > 0 {
		rows, err := d.DB.Query(fmt.Sprintf(`
			SELECT id, item_type, item_id, message_id
			FROM conversation_message_attachment
			WHERE message_id IN (%s)`, nReplacements(len(messageIds))),
			messageIds...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var mid int64
			a := &common.ConversationAttachment{}
			if err := rows.Scan(&a.Id, &a.ItemType, &a.ItemId, &mid); err != nil {
				return nil, err
			}
			messageMap[mid].Attachments = append(messageMap[mid].Attachments, a)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Participants

	c.Participants, err = d.getConversationParticipants(d.DB, id)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (d *DataService) GetConversationParticipantIds(conversationId int64) ([]int64, error) {
	rows, err := d.DB.Query(`SELECT person_id FROM conversation_participant WHERE conversation_id = ?`, conversationId)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (d *DataService) GetConversationTopics() ([]*common.ConversationTopic, error) {
	rows, err := d.DB.Query(`SELECT id, title, ordinal, active FROM conversation_topic ORDER BY ordinal`)
	if err != nil {
		return nil, err
	}
	var topics []*common.ConversationTopic
	for rows.Next() {
		t := &common.ConversationTopic{}
		if err := rows.Scan(&t.Id, &t.Title, &t.Ordinal, &t.Active); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

func (d *DataService) AddConversationTopic(title string, ordinal int, active bool) (int64, error) {
	res, err := d.DB.Exec(`INSERT INTO conversation_topic (title, ordinal, active) VALUES (?, ?, ?)`, title, ordinal, active)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) MarkConversationAsRead(id int64) error {
	_, err := d.DB.Exec("UPDATE conversation SET unread = false WHERE id = ?", id)
	return err
}

func (d *DataService) CreateConversation(fromId, toId, topicId int64, message string, attachments []*common.ConversationAttachment) (int64, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	res, err := tx.Exec(`
		INSERT INTO conversation (tstamp, topic_id, message_count, creator_id, owner_id, last_participant_id, last_message_tstamp, unread)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, now, topicId, 1, fromId, toId, fromId, now, true)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	conId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if _, err := d.createMessage(tx, now, conId, fromId, message, attachments); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := d.addConversationParticipants(tx, conId, []int64{fromId, toId}); err != nil {
		tx.Rollback()
		return 0, err
	}

	return conId, tx.Commit()
}

func (d *DataService) ReplyToConversation(conversationId, fromId int64, message string, attachments []*common.ConversationAttachment) (int64, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return 0, err
	}

	now := time.Now()

	// Get new owner by looking up the first participant that
	// is not the sender of the reply.
	// TODO: This only works when there's 2 participants which
	// is the case at the moment (patient, doctor) but will
	// not work when a nurse or other participant is added.
	var ownerId int64
	err = tx.QueryRow(`
		SELECT person_id
		FROM conversation_participant
		WHERE conversation_id = ? AND person_id != ?
		LIMIT 1`,
		conversationId, fromId).Scan(&ownerId)
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(`
		UPDATE conversation
		SET
		  message_count = message_count + 1,
		  last_participant_id = ?,
		  last_message_tstamp = ?,
		  owner_id = ?,
		  unread = true
		WHERE id = ?`,
		fromId, now, ownerId, conversationId,
	); err != nil {
		tx.Rollback()
		return 0, err
	}

	msgId, err := d.createMessage(tx, now, conversationId, fromId, message, attachments)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return msgId, tx.Commit()
}

func (d *DataService) createMessage(tx *sql.Tx, now time.Time, conversationId, fromId int64, message string, attachments []*common.ConversationAttachment) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO conversation_message (tstamp, conversation_id, person_id, body) VALUES (?, ?, ?, ?)`,
		now, conversationId, fromId, message)
	if err != nil {
		return 0, err
	}
	msgId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, a := range attachments {
		_, err := tx.Exec(`
			INSERT INTO conversation_message_attachment (message_id, item_type, item_id)
			VALUES (?, ?, ?)`, msgId, a.ItemType, a.ItemId)
		if err != nil {
			return 0, err
		}
		switch a.ItemType {
		case common.AttachmentTypePhoto:
			if err := d.claimPhoto(tx, a.ItemId, common.ClaimerTypeConversationMessage, msgId); err != nil {
				return 0, err
			}
		}
	}

	return msgId, nil
}

func (d *DataService) addConversationParticipants(tx *sql.Tx, conversationId int64, participants []int64) error {
	for _, id := range participants {
		_, err := tx.Exec(`
			REPLACE INTO conversation_participant (conversation_id, person_id) VALUES (?, ?)`,
			conversationId, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataService) getConversationParticipants(db db, conversationId int64) (map[int64]*common.Person, error) {
	rows, err := db.Query(`
		SELECT person_id, role_type, role_id
		FROM conversation_participant
		INNER JOIN person ON person.id = person_id
		WHERE conversation_id = ?`, conversationId)
	if err != nil {
		return nil, err
	}
	participants := map[int64]*common.Person{}
	for rows.Next() {
		p := &common.Person{}
		if err := rows.Scan(&p.Id, &p.RoleType, &p.RoleId); err != nil {
			return nil, err
		}
		participants[p.Id] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := d.populateDoctorOrPatientForPeople(participants); err != nil {
		return nil, err
	}
	return participants, nil
}

func (d *DataService) populateDoctorOrPatientForPeople(people map[int64]*common.Person) error {
	for _, p := range people {
		var err error
		switch p.RoleType {
		case PATIENT_ROLE:
			p.Patient, err = d.GetPatientFromId(p.RoleId)
		case DOCTOR_ROLE:
			p.Doctor, err = d.GetDoctorFromId(p.RoleId)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
