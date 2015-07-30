package api

import (
	"database/sql"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) AddBankAccount(bankAccount *common.BankAccount) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO bank_account (
			account_id, stripe_recipient_id, default_account, verify_amount_1, verify_amount_2,
			verify_transfer1_id, verify_transfer2_id, verify_expires, verified
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bankAccount.AccountID,
		bankAccount.StripeRecipientID,
		bankAccount.Default,
		bankAccount.VerifyAmount1,
		bankAccount.VerifyAmount2,
		bankAccount.VerifyTransfer1ID,
		bankAccount.VerifyTransfer2ID,
		bankAccount.VerifyExpires,
		bankAccount.Verified)
	if err != nil {
		return 0, err
	}

	bankAccount.ID, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return bankAccount.ID, nil
}

func (d *dataService) DeleteBankAccount(id int64) error {
	_, err := d.db.Exec(`DELETE FROM bank_account WHERE id = ?`, id)
	return err
}

func (d *dataService) ListBankAccounts(userAccountID int64) ([]*common.BankAccount, error) {
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

func (d *dataService) UpdateBankAccount(id int64, update *BankAccountUpdate) (int, error) {
	args := dbutil.MySQLVarArgs()
	if update.StripeRecipientID != nil {
		args.Append("stripe_recipient_id", *update.StripeRecipientID)
	}
	if update.Default != nil {
		args.Append("default_account", *update.Default)
	}
	if update.Verified != nil {
		args.Append("verified", *update.Verified)
	}
	if update.VerifyAmount1 != nil {
		if *update.VerifyAmount1 == 0 {
			args.Append("verify_amount_1", nil)
		} else {
			args.Append("verify_amount_1", *update.VerifyAmount1)
		}
	}
	if update.VerifyAmount2 != nil {
		if *update.VerifyAmount2 == 0 {
			args.Append("verify_amount_2", nil)
		} else {
			args.Append("verify_amount_2", *update.VerifyAmount2)
		}
	}
	if update.VerifyTransfer1ID != nil {
		if *update.VerifyTransfer1ID == "" {
			args.Append("verify_transfer1_id", nil)
		} else {
			args.Append("verify_transfer1_id", *update.VerifyTransfer1ID)
		}
	}
	if update.VerifyTransfer2ID != nil {
		if *update.VerifyTransfer2ID == "" {
			args.Append("verify_transfer2_id", nil)
		} else {
			args.Append("verify_transfer2_id", *update.VerifyTransfer2ID)
		}
	}
	if update.VerifyExpires != nil {
		if update.VerifyExpires.IsZero() {
			args.Append("verify_expires", nil)
		} else {
			args.Append("verify_expires", *update.VerifyExpires)
		}
	}
	if args.IsEmpty() {
		return 0, nil
	}
	values := append(args.Values(), id)
	res, err := d.db.Exec(`UPDATE bank_account SET `+args.Columns()+` WHERE id = ?`, values...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}
