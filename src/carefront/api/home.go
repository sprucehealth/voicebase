package api

import (
	"carefront/common"
	"carefront/libs/golog"
	"encoding/json"
	"reflect"
	"time"
)

func (d *DataService) DeleteHomeNotifications(ids []int64) error {
	switch len(ids) {
	case 0:
		return nil
	case 1:
		_, err := d.DB.Exec("DELETE FROM home_feed WHERE id = ?", ids[0])
		return err
	}
	_, err := d.DB.Exec("DELETE FROM home_feed WHERE id IN "+nReplacements(len(ids)), appendInt64sToInterfaceSlice(nil, ids)...)
	return err
}

func (d *DataService) DeleteHomeNotificationByUID(patientId int64, uid string) error {
	_, err := d.DB.Exec("DELETE FROM home_feed WHERE patient_id = ? AND uid = ?", patientId, uid)
	return err
}

func (d *DataService) GetHomeNotificationsForPatient(patientId int64, typeMap map[string]reflect.Type) ([]*common.HomeNotification, error) {
	rows, err := d.DB.Query(`
		SELECT id, uid, tstamp, expires, dismissible, dismiss_on_action, priority, type, data
		FROM home_feed
		WHERE patient_id = ?
		ORDER BY priority DESC, tstamp DESC`, patientId)
	if err != nil {
		return nil, err
	}
	var notes []*common.HomeNotification
	var toDelete []int64
	now := time.Now()
	for rows.Next() {
		note := &common.HomeNotification{
			PatientId: patientId,
		}
		var data []byte
		err := rows.Scan(
			&note.Id, &note.UID, &note.Timestamp, &note.Expires, &note.Dismissible,
			&note.DismissOnAction, &note.Priority, &note.Type, &data,
		)
		if err != nil {
			return nil, err
		}
		// Collect expired notifications for deletion
		if note.Expires != nil && note.Expires.Before(now) {
			toDelete = append(toDelete, note.Id)
			continue
		}
		// If the type is unknown or the data failes to unmarshal then ignore the notification
		// but continue since it's better to filter out the bad notifications rather than
		// not returning any.
		t := typeMap[note.Type]
		if t == nil {
			golog.Errorf("Unknown notification type %s for %d", note.Type, note.Id)
			continue
		}
		note.Data = reflect.New(t).Interface().(common.NotificationData)
		if err := json.Unmarshal(data, &note.Data); err != nil {
			golog.Errorf("Failed to unmarshal home notification %d: %s", note.Id, err.Error())
			continue
		}
		notes = append(notes, note)
	}
	// Delete expired notifications in the background
	if len(toDelete) > 0 {
		go func() {
			if err := d.DeleteHomeNotifications(toDelete); err != nil {
				golog.Errorf("Failed to delete expired notifications: %s", err.Error())
			}
		}()
	}
	return notes, rows.Err()
}

func (d *DataService) InsertHomeNotification(note *common.HomeNotification) (int64, error) {
	data, err := json.Marshal(note.Data)
	if err != nil {
		return 0, err
	}
	res, err := d.DB.Exec(`
		INSERT INTO home_feed (patient_id, uid, expires, dismissible, dismiss_on_action, priority, type, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.PatientId, note.UID, &note.Expires, note.Dismissible,
		note.DismissOnAction, note.Priority, note.Data.TypeName(), data)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
