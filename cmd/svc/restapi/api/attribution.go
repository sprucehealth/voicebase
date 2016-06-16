package api

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/restapi/attribution/model"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

func (d *dataService) LatestAccountAttributionData(accountID int64) (*model.AttributionData, error) {
	ad, err := d.latestAttributionData(ptr.Int64(accountID), nil)
	return ad, errors.Trace(err)
}

func (d *dataService) LatestDeviceAttributionData(deviceID string) (*model.AttributionData, error) {
	ad, err := d.latestAttributionData(nil, ptr.String(deviceID))
	return ad, errors.Trace(err)
}

func (d *dataService) latestAttributionData(accountID *int64, deviceID *string) (*model.AttributionData, error) {
	ad := &model.AttributionData{}
	var jsonData []byte
	v := make([]interface{}, 0, 2)
	q := `
    SELECT id, account_id, device_id, json_data, creation_date, last_modified
    FROM attribution_data
    WHERE`
	if accountID == nil {
		q += ` account_id IS NULL`
	} else {
		q += ` account_id = ?`
		v = append(v, *accountID)
	}
	q += ` AND`
	if deviceID == nil {
		q += ` device_id IS NULL`
	} else {
		q += ` device_id = ?`
		v = append(v, *deviceID)
	}
	q += ` ORDER BY id DESC
    LIMIT 1`
	if err := d.db.QueryRow(q, v...).Scan(
		&ad.ID, &ad.AccountID, &ad.DeviceID, &jsonData, &ad.CreationDate, &ad.LastModified); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(fmt.Sprintf("attribution_data not found for account_id: %v, device_id: %v", accountID, deviceID)))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	err := json.Unmarshal([]byte(jsonData), &ad.Data)
	return ad, errors.Trace(err)
}

func (d *dataService) InsertAttributionData(attributionData *model.AttributionData) (int64, error) {
	data, err := json.Marshal(attributionData.Data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	res, err := d.db.Exec(`
    INSERT INTO attribution_data (account_id, device_id, json_data)
    VALUES (?,?,?)`, attributionData.AccountID, attributionData.DeviceID, data)
	if err != nil {
		return 0, errors.Trace(err)
	}
	id, err := res.LastInsertId()
	return id, errors.Trace(err)
}

func (d *dataService) DeleteAttributionData(deviceID string) (int64, error) {
	res, err := d.db.Exec(`DELETE FROM attribution_data WHERE device_id = ?`, deviceID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}
