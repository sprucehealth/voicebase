package api

import (
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) GetLineItemsForType(itemType string) ([]*common.LineItem, error) {
	rows, err := d.db.Query(`select id, currency, description, amount, item_type from cost_item where item_type = ?`, itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []*common.LineItem
	for rows.Next() {
		var lItem common.LineItem
		err := rows.Scan(
			&lItem.ID,
			&lItem.Cost.Currency,
			&lItem.Description,
			&lItem.Cost.Amount,
			&lItem.ItemType,
		)
		if err != nil {
			return nil, err
		}
		lineItems = append(lineItems, &lItem)
	}
	return lineItems, rows.Err()
}

func (d *DataService) CreatePatientReceipt(receipt *common.PatientReceipt) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec(`insert into patient_receipt (patient_id, credit_card_id, item_type, item_id, receipt_reference_id, stripe_charge_id, status) 
		values (?,?,?,?,?,?,?)`, receipt.PatientID, receipt.CreditCardID, receipt.ItemType, receipt.ItemID,
		receipt.ReferenceNumber, receipt.StripeChargeID, receipt.Status)
	if err != nil {
		tx.Rollback()
		return err
	}

	patientReceiptID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	vals := make([]string, len(receipt.CostBreakdown.LineItems))
	params := make([]interface{}, len(receipt.CostBreakdown.LineItems)*4)
	for i, lItem := range receipt.CostBreakdown.LineItems {
		vals[i] = "(?,?,?,?)"
		params[i] = lItem.Cost.Currency
		params[i+1] = lItem.Description
		params[i+2] = lItem.Cost.Amount
		params[i+3] = patientReceiptID
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
