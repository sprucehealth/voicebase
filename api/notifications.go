package api

import (
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
)

func (d *dataService) GetPushConfigData(deviceToken string) (*common.PushConfigData, error) {
	rows, err := d.db.Query(`
		SELECT id, account_id, device_token, push_endpoint, platform, platform_version,
			app_version, app_type, app_env, app_version, device, device_model,
			device_id, creation_date
		FROM push_config
		WHERE device_token = ?`, deviceToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pushConfigDataList, err := getPushConfigDataFromRows(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(pushConfigDataList); {
	case l == 0:
		return nil, ErrNotFound("push_config")
	case l == 1:
		return pushConfigDataList[0], nil
	}

	return nil, fmt.Errorf("Expected 1 push config data but got %d", len(pushConfigDataList))
}

func (d *dataService) SnoozeConfigsForAccount(accountID int64) ([]*common.SnoozeConfig, error) {
	rows, err := d.db.Query(`
		SELECT account_id, start_hour, num_hours
		FROM communication_snooze
		WHERE account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snoozeConfigs []*common.SnoozeConfig
	for rows.Next() {
		var config common.SnoozeConfig
		if err := rows.Scan(
			&config.AccountID,
			&config.StartHour,
			&config.NumHours); err != nil {
			return nil, err
		}

		snoozeConfigs = append(snoozeConfigs, &config)
	}

	return snoozeConfigs, rows.Err()
}

func (d *dataService) DeletePushCommunicationPreferenceForAccount(accountID int64) error {
	_, err := d.db.Exec(`DELETE FROM push_config WHERE account_id=?`, accountID)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = d.db.Exec(`DELETE FROM communication_preference WHERE communication_type = ? AND account_id = ?`, common.Push.String(), accountID)
	return errors.Trace(err)
}

func (d *dataService) GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error) {
	rows, err := d.db.Query(`select id, account_id, device_token, push_endpoint, platform, platform_version, app_version, app_type, app_env, app_version, device, device_model, device_id, creation_date from push_config where account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getPushConfigDataFromRows(rows)
}

func getPushConfigDataFromRows(rows *sql.Rows) ([]*common.PushConfigData, error) {
	var pushConfigs []*common.PushConfigData
	for rows.Next() {
		var pushConfigData common.PushConfigData
		err := rows.Scan(&pushConfigData.ID, &pushConfigData.AccountID, &pushConfigData.DeviceToken, &pushConfigData.PushEndpoint, &pushConfigData.Platform, &pushConfigData.PlatformVersion, &pushConfigData.AppVersion, &pushConfigData.AppType, &pushConfigData.AppEnvironment,
			&pushConfigData.AppVersion, &pushConfigData.Device, &pushConfigData.DeviceModel, &pushConfigData.DeviceID, &pushConfigData.CreationDate)
		if err != nil {
			return nil, err
		}
		pushConfigs = append(pushConfigs, &pushConfigData)
	}
	return pushConfigs, rows.Err()
}

func (d *dataService) GetCommunicationPreferencesForAccount(accountID int64) ([]*common.CommunicationPreference, error) {
	rows, err := d.db.Query(`select id, account_id, communication_type, creation_date, status from communication_preference where account_id=? and status=?`, accountID, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var communicationPreferences []*common.CommunicationPreference
	for rows.Next() {
		var communicationPreference common.CommunicationPreference
		if err := rows.Scan(&communicationPreference.ID, &communicationPreference.AccountID,
			&communicationPreference.CommunicationType, &communicationPreference.CreationDate,
			&communicationPreference.Status); err != nil {
			return nil, err
		}
		communicationPreferences = append(communicationPreferences, &communicationPreference)
	}
	return communicationPreferences, rows.Err()
}

func (d *dataService) SetOrReplacePushConfigData(pushConfigData *common.PushConfigData) error {
	// begin transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// get account id of device token if one exists
	var accountID int64
	if err := d.db.QueryRow(`select account_id from push_config where device_token = ?`, pushConfigData.DeviceToken).Scan(&accountID); err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	// if account id is different, we know it will be replaced with the new account id
	// associated with the device token
	if accountID > 0 && accountID != pushConfigData.AccountID {
		var count int64
		if err := d.db.QueryRow(`select count(*) from push_config where device_token = ?`, pushConfigData.DeviceToken).Scan(&count); err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}

		// delete push communication entry if there are no other device tokens associated with account
		if count == 1 {
			_, err = tx.Exec(`delete from communication_preference where account_id = ? and communication_type = ?`, accountID, common.Push.String())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// replace entry with the new one
	_, err = tx.Exec(`replace into push_config (account_id, device_token, push_endpoint, platform, platform_version, app_version, app_type, app_env, device, device_model, device_id)
		values (?,?,?,?,?,?,?,?,?,?,?)`, pushConfigData.AccountID, pushConfigData.DeviceToken, pushConfigData.PushEndpoint, pushConfigData.Platform.String(),
		pushConfigData.PlatformVersion, pushConfigData.AppVersion, pushConfigData.AppType, pushConfigData.AppEnvironment, pushConfigData.Device, pushConfigData.DeviceModel, pushConfigData.DeviceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`replace into communication_preference (account_id, communication_type, status) values (?,?,?)`, pushConfigData.AccountID, common.Push.String(), StatusActive)
	if err != nil {
		tx.Rollback()
		return err
	}

	// commit transaction
	return tx.Commit()
}

func (d *dataService) SetPushPromptStatus(accountID int64, pStatus common.PushPromptStatus) error {
	_, err := d.db.Exec(`replace into notification_prompt_status (prompt_status, account_id) values (?,?)`, pStatus.String(), accountID)
	return err
}

func (d *dataService) GetPushPromptStatus(accountID int64) (common.PushPromptStatus, error) {
	var pStatusString string
	if err := d.db.QueryRow(`select prompt_status from notification_prompt_status where account_id = ?`, accountID).Scan(&pStatusString); err == sql.ErrNoRows {
		return common.Unprompted, nil
	} else if err != nil {
		return common.PushPromptStatus(""), err
	}
	return common.GetPushPromptStatus(pStatusString)
}
