package api

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

func (d *dataService) CreateRXReminder(r *common.RXReminder) error {
	_, err := d.db.Exec(`
    INSERT INTO rx_reminder 
      (treatment_id, text, reminder_interval, days, times)
      VALUES (?, ?, ?, ?, ?)`, r.TreatmentID, r.ReminderText, r.Interval.String(), common.JoinRXRDaySlice(r.Days), r.Times)
	return errors.Trace(err)
}

func (d *dataService) DeleteRXReminder(treatmentID int64) (int64, error) {
	res, err := d.db.Exec(`
    DELETE FROM rx_reminder
      WHERE treatment_id = ?`, treatmentID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	return n, errors.Trace(err)
}

func (d *dataService) RXReminders(treatmentIDs []int64) (map[int64]*common.RXReminder, error) {
	if len(treatmentIDs) == 0 {
		return nil, nil
	}
	reminders := make(map[int64]*common.RXReminder)
	rows, err := d.db.Query(`
    SELECT treatment_id, text, reminder_interval, days, times, created 
      FROM rx_reminder
      WHERE treatment_id IN (`+dbutil.MySQLArgs(len(treatmentIDs))+`)`, dbutil.AppendInt64sToInterfaceSlice(nil, treatmentIDs)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		r := &common.RXReminder{}
		var days string
		if err := rows.Scan(&r.TreatmentID, &r.ReminderText, &r.Interval, &days, &r.Times, &r.CreationDate); err != nil {
			return nil, errors.Trace(err)
		}
		rxrDays, err := common.SplitRXRDayString(days)
		if err != nil {
			return nil, errors.Trace(err)
		}
		r.Days = rxrDays
		reminders[r.TreatmentID] = r
	}
	return reminders, errors.Trace(rows.Err())
}

func (d *dataService) UpdateRXReminder(treatmentID int64, r *common.RXReminder) (int64, error) {
	varArgs := dbutil.MySQLVarArgs()
	varArgs.Append(`text`, r.ReminderText)
	varArgs.Append(`reminder_interval`, r.Interval.String())
	varArgs.Append(`days`, common.JoinRXRDaySlice(r.Days))
	varArgs.Append(`times`, r.Times)
	res, err := d.db.Exec(`
    UPDATE rx_reminder
      SET `+varArgs.ColumnsForUpdate()+
		`WHERE treatment_id = ?`, append(varArgs.Values(), treatmentID)...)
	if err != nil {
		return 0, errors.Trace(err)
	}
	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}
