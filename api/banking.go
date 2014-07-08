package api

import (
	"database/sql"
	"time"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) AddBankAccount(userAccountID int64, stripeRecipientID string, defaultAccount bool) (int64, error) {
	res, err := d.db.Exec(`INSERT INTO bank_account (account_id, stripe_recipient_id, default_account) VALUES (?, ?, ?)`,
		userAccountID, stripeRecipientID, defaultAccount)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) DeleteBankaccount(id int64) error {
	_, err := d.db.Exec(`DELETE FROM bank_account WHERE id = ?`, id)
	return err
}

func (d *DataService) ListBankAccounts(userAccountID int64) ([]*common.BankAccount, error) {
	rows, err := d.db.Query(`
		SELECT id, stripe_recipient_id, creation_date, default_account, verified,
			verify_amount_1, verify_amount_2, verify_transfer1_id, verify_transfer2_id, verify_expires
		FROM bank_account
		WHERE account_id = ?
		ORDER BY default_account DESC, id`, userAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*common.BankAccount
	for rows.Next() {
		acc := &common.BankAccount{
			AccountID: userAccountID,
		}
		var amount1, amount2 sql.NullInt64
		var t1ID, t2ID sql.NullString
		var expires *time.Time
		err := rows.Scan(
			&acc.ID, &acc.StripeRecipientID, &acc.Created, &acc.Default,
			&acc.Verified, &amount1, &amount2, &t1ID, &t2ID, &expires)
		if err != nil {
			return nil, err
		}
		acc.VerifyAmount1 = int(amount1.Int64)
		acc.VerifyAmount2 = int(amount2.Int64)
		acc.VerifyTransfer1ID = t1ID.String
		acc.VerifyTransfer2ID = t2ID.String
		if expires != nil {
			acc.VerifyExpires = *expires
		}
		accounts = append(accounts, acc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (d *DataService) UpdateBankAccountVerficiation(id int64, amount1, amount2 int, transfer1ID, transfer2ID string, expires time.Time, verified bool) error {
	if verified {
		_, err := d.db.Exec(`
			UPDATE bank_account
			SET verified = true, verify_amount_1 = NULL, verify_amount_2 = NULL, verify_expires = NULL
			WHERE id = ?`, id)
		return err
	}
	_, err := d.db.Exec(`
			UPDATE bank_account
			SET verified = false, verify_amount_1 = ?, verify_amount_2 = ?, verify_transfer1_id = ?, verify_transfer2_id = ?, verify_expires = ?
			WHERE id = ?`, amount1, amount2, transfer1ID, transfer2ID, expires, id)
	return err
}
