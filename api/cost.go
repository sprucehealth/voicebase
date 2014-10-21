package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/sku"
)

func (d *DataService) GetItemCost(id int64) (*common.ItemCost, error) {
	row := d.db.QueryRow(`
		SELECT item_cost.id, sku_id, status 
		FROM item_cost 
		WHERE item_cost.id = ?`, id)
	return d.getItemCostFromRow(row)
}

func (d *DataService) GetActiveItemCost(itemType sku.SKU) (*common.ItemCost, error) {
	row := d.db.QueryRow(`
		SELECT item_cost.id, sku_id, status 
		FROM item_cost 
		WHERE status = ? and sku_id = ?`, STATUS_ACTIVE, d.skuMapping[itemType.String()])
	return d.getItemCostFromRow(row)
}

func (d *DataService) getItemCostFromRow(row *sql.Row) (*common.ItemCost, error) {
	var itemCost common.ItemCost
	var skuID int64
	err := row.Scan(
		&itemCost.ID,
		&skuID,
		&itemCost.Status)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	itemCost.ItemType, err = sku.GetSKU(d.skuIDToTypeMapping[skuID])
	if err != nil {
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

	res, err := tx.Exec(`insert into patient_receipt (patient_id, sku_id, item_id, item_cost_id, receipt_reference_id, status) 
		values (?,?,?,?,?,?)`, receipt.PatientID, d.skuMapping[receipt.ItemType.String()], receipt.ItemID, receipt.ItemCostID,
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

func (d *DataService) GetPatientReceipt(patientID, itemID int64, itemType sku.SKU, includeLineItems bool) (*common.PatientReceipt, error) {
	var patientReceipt common.PatientReceipt
	var creditCardID sql.NullInt64
	var stripeChargeID sql.NullString
	if err := d.db.QueryRow(`
		SELECT patient_receipt.id, patient_id, credit_card_id, item_id, item_cost_id, receipt_reference_id, stripe_charge_id, creation_timestamp, status 
		FROM patient_receipt 
		WHERE patient_id = ? AND item_id = ? AND sku_id = ?`, patientID, itemID, d.skuMapping[itemType.String()]).Scan(
		&patientReceipt.ID,
		&patientReceipt.PatientID,
		&creditCardID,
		&patientReceipt.ItemID,
		&patientReceipt.ItemCostID,
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
	patientReceipt.ItemType = itemType

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

func (d *DataService) CreateDoctorTransaction(transaction *common.DoctorTransaction) error {
	res, err := d.db.Exec(`
		REPLACE INTO doctor_transaction
		(doctor_id, item_cost_id, item_id, sku_id, patient_id) 
		VALUES (?,?,?,?,?)`, transaction.DoctorID, transaction.ItemCostID, transaction.ItemID,
		d.skuMapping[transaction.ItemType.String()], transaction.PatientID)
	if err != nil {
		return err
	}

	transactionID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	transaction.ID = transactionID
	return nil
}

func (d *DataService) TransactionsForDoctor(doctorID int64) ([]*common.DoctorTransaction, error) {
	rows, err := d.db.Query(`
		SELECT doctor_transaction.id, doctor_id, item_cost_id, item_id, sku_id, patient_id 
		FROM doctor_transaction
		WHERE doctor_id = ?`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*common.DoctorTransaction
	for rows.Next() {
		var tItem common.DoctorTransaction
		var skuID int64
		if err := rows.Scan(
			&tItem.ID,
			&tItem.DoctorID,
			&tItem.ItemCostID,
			&tItem.ItemID,
			&skuID,
			&tItem.PatientID); err != nil {
			return nil, err
		}

		tItem.ItemType, err = sku.GetSKU(d.skuIDToTypeMapping[skuID])
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, &tItem)
	}

	return transactions, rows.Err()
}
