package api

import (
	"carefront/common"
	"encoding/json"
)

func (d *DataService) DeleteHomeNotification(id int64) error {
	_, err := d.DB.Exec("DELETE FROM home_feed WHERE id = ?", id)
	return err
}

func (d *DataService) DeleteHomeNotificationByUID(patientId int64, uid string) error {
	_, err := d.DB.Exec("DELETE FROM home_feed WHERE patient_id = ? AND uid = ?", patientId, uid)
	return err
}

func (d *DataService) GetHomeNotificationsForPatient(patientId int64) ([]*common.HomeNotification, error) {
	rows, err := d.DB.Query(`
		SELECT id, uid, tstamp, expires, dismissible, dismiss_on_action, priority, type, data
		FROM home_feed
		WHERE patient_id = ?`, patientId)
	if err != nil {
		return nil, err
	}
	var notes []*common.HomeNotification
	for rows.Next() {
		note := &common.HomeNotification{
			PatientId: patientId,
		}
		err := rows.Scan(
			&note.Id, &note.UID, &note.Timestamp, &note.Expires, &note.Dismissible,
			&note.DismissOnAction, &note.Priority, &note.Type, &note.Data,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (d *DataService) InsertHomeNotification(note *common.HomeNotification) (int64, error) {
	data, ok := note.Data.([]byte)
	if !ok {
		var err error
		data, err = json.Marshal(note.Data)
		if err != nil {
			return 0, err
		}
	}
	res, err := d.DB.Exec(`
		INSERT INTO home_feed (patient_id, uid, expires, dismissible, dismiss_on_action, priority, type, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.PatientId, note.UID, note.Expires, note.Dismissible,
		note.DismissOnAction, note.Priority, note.Type, data)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
