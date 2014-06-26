package api

import (
	"database/sql"
	"fmt"
	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetPushConfigData(deviceToken string) (*common.PushConfigData, error) {

	rows, err := d.db.Query(`select id, account_id, device_token, push_endpoint, platform, platform_version, app_version, app_type, app_env, app_version, device, device_model, device_id, creation_date from push_config where device_token = ?`, deviceToken)
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
		return nil, NoRowsError
	case l == 1:
		return pushConfigDataList[0], nil
	}

	return nil, fmt.Errorf("Expected 1 push config data but got %d", len(pushConfigDataList))
}

func (d *DataService) DeletePushCommunicationPreferenceForAccount(accountId int64) error {
	_, err := d.db.Exec(`delete from push_config where account_id=?`, accountId)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`delete from communication_preference where communication_type = ? and account_id = ?`, common.Push.String(), accountId)
	return err
}

func (d *DataService) GetPushConfigDataForAccount(accountId int64) ([]*common.PushConfigData, error) {
	rows, err := d.db.Query(`select id, account_id, device_token, push_endpoint, platform, platform_version, app_version, app_type, app_env, app_version, device, device_model, device_id, creation_date from push_config where account_id = ?`, accountId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getPushConfigDataFromRows(rows)
}

func getPushConfigDataFromRows(rows *sql.Rows) ([]*common.PushConfigData, error) {
	pushConfigs := make([]*common.PushConfigData, 0)
	for rows.Next() {
		var pushConfigData common.PushConfigData
		err := rows.Scan(&pushConfigData.Id, &pushConfigData.AccountId, &pushConfigData.DeviceToken, &pushConfigData.PushEndpoint, &pushConfigData.Platform, &pushConfigData.PlatformVersion, &pushConfigData.AppVersion, &pushConfigData.AppType, &pushConfigData.AppEnvironment,
			&pushConfigData.AppVersion, &pushConfigData.Device, &pushConfigData.DeviceModel, &pushConfigData.DeviceID, &pushConfigData.CreationDate)
		if err != nil {
			return nil, err
		}
		pushConfigs = append(pushConfigs, &pushConfigData)
	}
	return pushConfigs, rows.Err()
}

func (d *DataService) GetCommunicationPreferencesForAccount(accountId int64) ([]*common.CommunicationPreference, error) {
	rows, err := d.db.Query(`select id, account_id, communication_type, creation_date, status from communication_preference where account_id=? and status=?`, accountId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	communicationPreferences := make([]*common.CommunicationPreference, 0)
	for rows.Next() {
		var communicationPreference common.CommunicationPreference
		if err := rows.Scan(&communicationPreference.Id, &communicationPreference.AccountId,
			&communicationPreference.CommunicationType, &communicationPreference.CreationDate,
			&communicationPreference.Status); err != nil {
			return nil, err
		}
		communicationPreferences = append(communicationPreferences, &communicationPreference)
	}
	return communicationPreferences, rows.Err()
}

func (d *DataService) SetOrReplacePushConfigData(pushConfigData *common.PushConfigData) error {
	// begin transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// get account id of device token if one exists
	var accountId int64
	if err := d.db.QueryRow(`select account_id from push_config where device_token = ?`, pushConfigData.DeviceToken).Scan(&accountId); err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	// if account id is different, we know it will be replaced with the new account id
	// associated with the device token
	if accountId > 0 && accountId != pushConfigData.AccountId {
		var count int64
		if err := d.db.QueryRow(`select count(*) from push_config where device_token = ?`, pushConfigData.DeviceToken).Scan(&count); err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}

		// delete push communication entry if there are no other device tokens associated with account
		if count == 1 {
			_, err = tx.Exec(`delete from communication_preference where account_id = ? and communication_type = ?`, accountId, common.Push.String())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// replace entry with the new one
	_, err = tx.Exec(`replace into push_config (account_id, device_token, push_endpoint, platform, platform_version, app_version, app_type, app_env, device, device_model, device_id) 
		values (?,?,?,?,?,?,?,?,?,?,?)`, pushConfigData.AccountId, pushConfigData.DeviceToken, pushConfigData.PushEndpoint, pushConfigData.Platform.String(),
		pushConfigData.PlatformVersion, pushConfigData.AppVersion, pushConfigData.AppType, pushConfigData.AppEnvironment, pushConfigData.Device, pushConfigData.DeviceModel, pushConfigData.DeviceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`replace into communication_preference (account_id, communication_type, status) values (?,?,?)`, pushConfigData.AccountId, common.Push.String(), STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	// commit transaction
	return tx.Commit()
}

func (d *DataService) SetPushPromptStatus(accountId int64, pStatus common.PushPromptStatus) error {
	_, err := d.db.Exec(`replace into notification_prompt_status (prompt_status, account_id) values (?,?)`, pStatus.String(), accountId)
	return err
}

func (d *DataService) GetPushPromptStatus(accountId int64) (common.PushPromptStatus, error) {
	var pStatusString string
	if err := d.db.QueryRow(`select prompt_status from notification_prompt_status where account_id = ?`, accountId).Scan(&pStatusString); err == sql.ErrNoRows {
		return common.Unprompted, nil
	} else if err != nil {
		return common.PushPromptStatus(""), err
	}
	return common.GetPushPromptStatus(pStatusString)
}
