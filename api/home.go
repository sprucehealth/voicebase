package api

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"database/sql"
	"encoding/json"
	"reflect"
	"time"
)

func (d *DataService) DeletePatientNotifications(ids []int64) error {
	switch len(ids) {
	case 0:
		return nil
	case 1:
		_, err := d.db.Exec("DELETE FROM patient_notifications WHERE id = ?", ids[0])
		return err
	}
	_, err := d.db.Exec("DELETE FROM patient_notifications WHERE id IN "+nReplacements(len(ids)), appendInt64sToInterfaceSlice(nil, ids)...)
	return err
}

func (d *DataService) DeletePatientNotificationByUID(patientId int64, uid string) error {
	_, err := d.db.Exec("DELETE FROM patient_notifications WHERE patient_id = ? AND uid = ?", patientId, uid)
	return err
}

func (d *DataService) GetNotificationCountForPatient(patientId int64) (int64, error) {
	var count int64
	err := d.db.QueryRow(`select count(*) from patient_notifications where patient_id = ?`, patientId).Scan(&count)
	return count, err
}

func (d *DataService) GetNotificationsForPatient(patientId int64, typeMap map[string]reflect.Type) ([]*common.Notification, []*common.Notification, error) {
	rows, err := d.db.Query(`
		SELECT id, uid, tstamp, expires, dismissible, dismiss_on_action, priority, type, data
		FROM patient_notifications
		WHERE patient_id = ?
		ORDER BY priority DESC, tstamp DESC`, patientId)
	if err == sql.ErrNoRows {
		return []*common.Notification{}, nil, nil
	} else if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var notes []*common.Notification
	var badNotes []*common.Notification
	var toDelete []int64
	now := time.Now()
	for rows.Next() {
		note := &common.Notification{}
		var dataType string
		var data []byte
		err := rows.Scan(
			&note.Id, &note.UID, &note.Timestamp, &note.Expires, &note.Dismissible,
			&note.DismissOnAction, &note.Priority, &dataType, &data,
		)
		if err != nil {
			return nil, nil, err
		}
		// Collect expired notifications for deletion
		if note.Expires != nil && note.Expires.Before(now) {
			toDelete = append(toDelete, note.Id)
			continue
		}
		// If the type is unknown or the data failes to unmarshal then ignore the notification
		// but continue since it's better to filter out the bad notifications rather than
		// not returning any.
		t := typeMap[dataType]
		if t == nil {
			golog.Errorf("Unknown notification type %s for %d", dataType, note.Id)
			note.Data = &common.TypedData{Data: data, Type: dataType}
			badNotes = append(badNotes, note)
			continue
		}
		note.Data = reflect.New(t).Interface().(common.Typed)
		if err := json.Unmarshal(data, &note.Data); err != nil {
			golog.Errorf("Failed to unmarshal home notification %d: %s", note.Id, err.Error())
			note.Data = &common.TypedData{Data: data, Type: dataType}
			badNotes = append(badNotes, note)
			continue
		}
		notes = append(notes, note)
	}
	// Delete expired notifications in the background
	if len(toDelete) > 0 {
		go func() {
			if err := d.DeletePatientNotifications(toDelete); err != nil {
				golog.Errorf("Failed to delete expired notifications: %s", err.Error())
			}
		}()
	}
	return notes, badNotes, rows.Err()
}

func (d *DataService) InsertPatientNotification(patientId int64, note *common.Notification) (int64, error) {
	data, err := json.Marshal(note.Data)
	if err != nil {
		return 0, err
	}
	res, err := d.db.Exec(`
		INSERT INTO patient_notifications (patient_id, uid, expires, dismissible, dismiss_on_action, priority, type, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		patientId, note.UID, note.Expires, note.Dismissible,
		note.DismissOnAction, note.Priority, note.Data.TypeName(), data)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) GetHealthLogForPatient(patientId int64, typeMap map[string]reflect.Type) ([]*common.HealthLogItem, []*common.HealthLogItem, error) {
	rows, err := d.db.Query(`
		SELECT id, uid, tstamp, type, data
		FROM health_log
		WHERE patient_id = ?
		ORDER BY tstamp DESC`, patientId)
	if err == sql.ErrNoRows {
		return []*common.HealthLogItem{}, nil, nil
	} else if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []*common.HealthLogItem
	var badItems []*common.HealthLogItem
	for rows.Next() {
		item := &common.HealthLogItem{}
		var dataType string
		var data []byte
		err := rows.Scan(&item.Id, &item.UID, &item.Timestamp, &dataType, &data)
		if err != nil {
			return nil, nil, err
		}
		// If the type is unknown or the data failes to unmarshal then ignore the item
		// but continue since it's better to filter out the bad items rather than
		// not returning any.
		t := typeMap[dataType]
		if t == nil {
			golog.Errorf("Unknown health log item type %s for %d", dataType, item.Id)
			item.Data = &common.TypedData{Data: data, Type: dataType}
			badItems = append(badItems, item)
			continue
		}
		item.Data = reflect.New(t).Interface().(common.Typed)
		if err := json.Unmarshal(data, &item.Data); err != nil {
			golog.Errorf("Failed to unmarshal health log item %d: %s", item.Id, err.Error())
			item.Data = &common.TypedData{Data: data, Type: dataType}
			badItems = append(badItems, item)
			continue
		}
		items = append(items, item)
	}
	return items, badItems, rows.Err()
}

func (d *DataService) InsertOrUpdatePatientHealthLogItem(patientId int64, item *common.HealthLogItem) (int64, error) {
	data, err := json.Marshal(item.Data)
	if err != nil {
		return 0, err
	}
	res, err := d.db.Exec(`
		REPLACE INTO health_log (patient_id, uid, type, data)
		VALUES (?, ?, ?, ?)`,
		patientId, item.UID, item.Data.TypeName(), data)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
