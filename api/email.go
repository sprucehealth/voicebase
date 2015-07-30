package api

import (
	"strings"

	"github.com/sprucehealth/backend/libs/dbutil"

	"database/sql"
)

func (d *dataService) EmailUpdateOptOut(accountID int64, emailType string, optout bool) error {
	if !optout {
		_, err := d.db.Exec(`DELETE FROM account_email_optout WHERE account_id = ? AND type = ?`, accountID, emailType)
		return err
	}
	_, err := d.db.Exec(`REPLACE INTO account_email_optout (account_id, type) VALUES (?, ?)`, accountID, emailType)
	return err
}

func (d *dataService) EmailRecipients(accountIDs []int64) ([]*Recipient, error) {
	rows, err := d.db.Query(`
		SELECT a.id, a.email, p.first_name || ' ' || p.last_name, d.first_name || ' ' || d.last_name
		FROM account a
		LEFT OUTER JOIN patient p ON p.account_id = a.id
		LEFT OUTER JOIN doctor d ON d.account_id = d.id
		WHERE a.id IN (`+dbutil.MySQLArgs(len(accountIDs))+`)
	`, dbutil.AppendInt64sToInterfaceSlice(nil, accountIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pName, dName sql.NullString
	rcpt := make([]*Recipient, 0, len(accountIDs))
	for rows.Next() {
		r := &Recipient{}
		if err := rows.Scan(&r.AccountID, &r.Email, &pName, &dName); err != nil {
			return nil, err
		}
		if pName.Valid {
			r.Name = pName.String
		}
		if dName.Valid {
			r.Name = dName.String
		}
		rcpt = append(rcpt, r)
	}
	return rcpt, rows.Err()
}

func (d *dataService) EmailRecipientsWithOptOut(accountIDs []int64, emailType string, onlyOnce bool) ([]*Recipient, error) {
	var rows *sql.Rows
	var err error
	if !onlyOnce {
		rows, err = d.db.Query(`
			SELECT a.id, a.email, p.first_name || ' ' || p.last_name, d.first_name || ' ' || d.last_name
			FROM account a
			LEFT OUTER JOIN account_email_optout o ON o.account_id = a.id AND (o.type = ? OR o.type = 'all')
			LEFT OUTER JOIN patient p ON p.account_id = a.id
			LEFT OUTER JOIN doctor d ON d.account_id = d.id
			WHERE o.type IS NULL AND a.id IN (`+dbutil.MySQLArgs(len(accountIDs))+`)
		`, dbutil.AppendInt64sToInterfaceSlice([]interface{}{emailType}, accountIDs)...)
	} else {
		rows, err = d.db.Query(`
			SELECT a.id, a.email, p.first_name || ' ' || p.last_name, d.first_name || ' ' || d.last_name
			FROM account a
			LEFT OUTER JOIN account_email_optout o ON o.account_id = a.id AND o.type = (o.type = ? OR o.type = 'all')
			LEFT OUTER JOIN account_email_sent s ON s.account_id = a.id AND s.type = ?
			LEFT OUTER JOIN patient p ON p.account_id = a.id
			LEFT OUTER JOIN doctor d ON d.account_id = d.id
			WHERE o.type IS NULL AND s.type IS NULL AND a.id IN (`+dbutil.MySQLArgs(len(accountIDs))+`)
		`, dbutil.AppendInt64sToInterfaceSlice([]interface{}{emailType, emailType}, accountIDs)...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pName, dName sql.NullString
	rcpt := make([]*Recipient, 0, len(accountIDs))
	for rows.Next() {
		r := &Recipient{}
		if err := rows.Scan(&r.AccountID, &r.Email, &pName, &dName); err != nil {
			return nil, err
		}
		if pName.Valid {
			r.Name = pName.String
		}
		if dName.Valid {
			r.Name = dName.String
		}
		rcpt = append(rcpt, r)
	}
	return rcpt, rows.Err()
}

func (d *dataService) EmailRecordSend(accountIDs []int64, emailType string) error {
	reps := make([]string, len(accountIDs))
	vals := make([]interface{}, 0, len(accountIDs)*2)
	for i, id := range accountIDs {
		reps[i] = "(?,?)"
		vals = append(vals, id, emailType)
	}
	_, err := d.db.Exec(`
		INSERT INTO account_email_sent (account_id, type)
		VALUES `+strings.Join(reps, ","), vals...)
	return err
}

func (d *dataService) EmailCampaignState(key string) ([]byte, error) {
	var data []byte
	row := d.db.QueryRow(`SELECT "data" FROM email_campaign_state WHERE "key" = ?`, key)
	err := row.Scan(&data)
	if err == sql.ErrNoRows {
		err = nil
	}
	return data, err
}

func (d *dataService) UpdateEmailCampaignState(key string, state []byte) error {
	_, err := d.db.Exec(`REPLACE INTO email_campaign_state ("key", "data")  VALUES (?, ?)`, key, state)
	return err
}
