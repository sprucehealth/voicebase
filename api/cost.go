package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetItemCost(id int64) (*common.ItemCost, error) {
	row := d.db.QueryRow(`select id, item_type, status from item_cost where id = ?`, id)
	return d.getItemCostFromRow(row)
}

func (d *DataService) GetActiveItemCost(itemType string) (*common.ItemCost, error) {
	row := d.db.QueryRow(`select id, item_type, status from item_cost where status = ? and item_type = ?`, STATUS_ACTIVE, itemType)
	return d.getItemCostFromRow(row)
}

func (d *DataService) getItemCostFromRow(row *sql.Row) (*common.ItemCost, error) {
	var itemCost common.ItemCost
	err := row.Scan(
		&itemCost.ID,
		&itemCost.ItemType,
		&itemCost.Status)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`select id, currency, description, amount from line_item where item_cost_id = ?`, itemCost.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lItem common.LineItem
		err := rows.Scan(
			&lItem.ID,
			&lItem.Cost.Currency,
			&lItem.Description,
			&lItem.Cost.Amount,
		)
		if err != nil {
			return nil, err
		}
		itemCost.LineItems = append(itemCost.LineItems, &lItem)
	}
	return &itemCost, rows.Err()
}

func (d *DataService) CreatePatientReceipt(receipt *common.PatientReceipt) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec(`insert into patient_receipt (patient_id, item_type, item_id, receipt_reference_id, status) 
		values (?,?,?,?,?)`, receipt.PatientID, receipt.ItemType, receipt.ItemID,
		receipt.ReferenceNumber, receipt.Status.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	receipt.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	receipt.CreationTimestamp = time.Now()

	vals := make([]string, len(receipt.CostBreakdown.LineItems))
	params := make([]interface{}, len(receipt.CostBreakdown.LineItems)*4)
	for i, lItem := range receipt.CostBreakdown.LineItems {
		vals[i] = "(?,?,?,?)"
		params[i*4] = lItem.Cost.Currency
		params[i*4+1] = lItem.Description
		params[i*4+2] = lItem.Cost.Amount
		params[i*4+3] = receipt.ID
	}

	if len(vals) == 0 {
		return tx.Commit()
	}

	_, err = tx.Exec(`insert into patient_charge_item (currency, description, amount, patient_receipt_id) values `+strings.Join(vals, ","), params...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) UpdatePatientReceipt(id int64, update *PatientReceiptUpdate) error {
	var cols []string
	var vals []interface{}

	if update.Status != nil {
		cols = append(cols, "status = ?")
		vals = append(vals, update.Status.String())
	}
	if update.CreditCardID != nil {
		cols = append(cols, "credit_card_id = ?")
		vals = append(vals, *update.CreditCardID)
	}
	if update.StripeChargeID != nil {
		cols = append(cols, "stripe_charge_id = ?")
		vals = append(vals, *update.StripeChargeID)
	}

	if len(cols) == 0 {
		return nil
	}

	vals = append(vals, id)

	_, err := d.db.Exec(`update patient_receipt set `+strings.Join(cols, ", ")+` where id = ?`, vals...)
	return err
}

func (d *DataService) GetPatientReceipt(patientID, itemID int64, itemType string, includeLineItems bool) (*common.PatientReceipt, error) {
	var patientReceipt common.PatientReceipt
	var creditCardID sql.NullInt64
	var stripeChargeID sql.NullString
	if err := d.db.QueryRow(`select id, patient_id, credit_card_id, item_type, item_id, receipt_reference_id, stripe_charge_id, creation_timestamp, status from patient_receipt 
		where patient_id = ? and item_id = ? and item_type = ?`, patientID, itemID, itemType).Scan(
		&patientReceipt.ID,
		&patientReceipt.PatientID,
		&creditCardID,
		&patientReceipt.ItemType,
		&patientReceipt.ItemID,
		&patientReceipt.ReferenceNumber,
		&stripeChargeID,
		&patientReceipt.CreationTimestamp,
		&patientReceipt.Status,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	patientReceipt.CreditCardID = creditCardID.Int64
	patientReceipt.StripeChargeID = stripeChargeID.String

	if !includeLineItems {
		return &patientReceipt, nil
	}

	rows, err := d.db.Query(`select id, description, currency, amount from patient_charge_item where patient_receipt_id = ?`, patientReceipt.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []*common.LineItem
	for rows.Next() {
		var lItem common.LineItem
		if err := rows.Scan(
			&lItem.ID,
			&lItem.Description,
			&lItem.Cost.Currency,
			&lItem.Cost.Amount); err != nil {
			return nil, err
		}
		lineItems = append(lineItems, &lItem)
	}
	patientReceipt.CostBreakdown = &common.CostBreakdown{LineItems: lineItems}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &patientReceipt, nil
}
