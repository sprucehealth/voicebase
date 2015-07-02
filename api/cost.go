package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/dbutil"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetItemCost(id int64) (*common.ItemCost, error) {
	row := d.db.QueryRow(`
		SELECT item_cost.id, sku_id, status
		FROM item_cost
		WHERE item_cost.id = ?`, id)
	return d.getItemCostFromRow(row)
}

func (d *DataService) GetActiveItemCost(skuType string) (*common.ItemCost, error) {
	skuID, err := d.skuIDFromType(skuType)
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRow(`
		SELECT item_cost.id, sku_id, status
		FROM item_cost
		WHERE status = ? and sku_id = ?`, StatusActive, skuID)
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
		return nil, ErrNotFound("item_cost")
	} else if err != nil {
		return nil, err
	}

	itemCost.SKUType, err = d.skuTypeFromID(skuID)
	if err != nil {
		return nil, err
	}

	itemCost.SKUCategory, err = d.CategoryForSKU(itemCost.SKUType)
	if err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, currency, description, amount 
		FROM line_item 
		WHERE item_cost_id = ?`, itemCost.ID)
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

	skuID, err := d.skuIDFromType(receipt.SKUType)
	if err != nil {
		return err
	}

	res, err := tx.Exec(`
		INSERT INTO patient_receipt (patient_id, sku_id, item_id, item_cost_id, receipt_reference_id, status) 
		VALUES (?,?,?,?,?,?)`, receipt.PatientID, skuID, receipt.ItemID, receipt.ItemCostID,
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

	_, err = tx.Exec(`
		INSERT INTO patient_charge_item (currency, description, amount, patient_receipt_id)
		VALUES `+strings.Join(vals, ","), params...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) UpdatePatientReceipt(id int64, update *PatientReceiptUpdate) error {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.StripeChargeID != nil {
		args.Append("stripe_charge_id", *update.StripeChargeID)
	}
	if args.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE patient_receipt SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id)...)
	return err
}

func (d *DataService) GetPatientReceipt(patientID, itemID int64, skuType string, includeLineItems bool) (*common.PatientReceipt, error) {
	skuID, err := d.skuIDFromType(skuType)
	if err != nil {
		return nil, err
	}

	var patientReceipt common.PatientReceipt
	var stripeChargeID sql.NullString
	if err := d.db.QueryRow(`
		SELECT patient_receipt.id, patient_id, item_id, item_cost_id, receipt_reference_id, stripe_charge_id, creation_timestamp, status
		FROM patient_receipt
		WHERE patient_id = ? AND item_id = ? AND sku_id = ?`, patientID, itemID, skuID).Scan(
		&patientReceipt.ID,
		&patientReceipt.PatientID,
		&patientReceipt.ItemID,
		&patientReceipt.ItemCostID,
		&patientReceipt.ReferenceNumber,
		&stripeChargeID,
		&patientReceipt.CreationTimestamp,
		&patientReceipt.Status,
	); err == sql.ErrNoRows {
		return nil, ErrNotFound("patient_receipt")
	} else if err != nil {
		return nil, err
	}
	patientReceipt.StripeChargeID = stripeChargeID.String
	patientReceipt.SKUType = skuType

	if !includeLineItems {
		return &patientReceipt, nil
	}

	rows, err := d.db.Query(`
		SELECT id, description, currency, amount
		FROM patient_charge_item
		WHERE patient_receipt_id = ?`, patientReceipt.ID)
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
	skuID, err := d.skuIDFromType(transaction.SKUType)
	if err != nil {
		return err
	}

	res, err := d.db.Exec(`
		REPLACE INTO doctor_transaction
		(doctor_id, item_cost_id, item_id, sku_id, patient_id) 
		VALUES (?,?,?,?,?)`, transaction.DoctorID, transaction.ItemCostID, transaction.ItemID,
		skuID, transaction.PatientID)
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
		WHERE doctor_id = ?
		ORDER BY created DESC`, doctorID)
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

		tItem.SKUType, err = d.skuTypeFromID(skuID)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, &tItem)
	}

	return transactions, rows.Err()
}

func (d *DataService) TransactionForItem(itemID, doctorID int64, skuType string) (*common.DoctorTransaction, error) {
	skuID, err := d.skuIDFromType(skuType)
	if err != nil {
		return nil, err
	}

	item := common.DoctorTransaction{
		SKUType: skuType,
	}
	err = d.db.QueryRow(`
		SELECT doctor_transaction.id, doctor_id, item_cost_id, item_id, patient_id 
		FROM doctor_transaction
		WHERE doctor_id = ? AND item_id = ? AND sku_id = ?
		ORDER BY created DESC`, doctorID, itemID, skuID).Scan(
		&item.ID,
		&item.DoctorID,
		&item.ItemCostID,
		&item.ItemID,
		&item.PatientID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("doctor_transaction")
	} else if err != nil {
		return nil, err
	}

	return &item, nil
}

func (d *DataService) VisitSKUs(activeOnly bool) ([]string, error) {

	var statusClause string
	if activeOnly {
		statusClause = " AND status = 'ACTIVE'"
	}

	rows, err := d.db.Query(`
		SELECT sku.type FROM sku
		INNER JOIN sku_category ON sku_category_id = sku_category.id
		WHERE sku_category.type in ('visit', 'followup')` + statusClause +
		`ORDER BY sku.type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skus []string
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			return nil, nil
		}
		skus = append(skus, sku)
	}

	return skus, rows.Err()
}
