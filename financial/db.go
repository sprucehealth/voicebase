package financial

import (
	"database/sql"
	"time"
)

type dataAccess struct {
	db *sql.DB
}

func NewDataAccess(db *sql.DB) Financial {
	return &dataAccess{
		db: db,
	}
}

const (
	maxItems = 1000
)

func (d *dataAccess) IncomingItems(from, to time.Time) ([]*IncomingItem, error) {
	rows, err := d.db.Query(`
		SELECT pr.creation_timestamp, pr.stripe_charge_id, sku.type, pr.receipt_reference_id, pr.item_id, pl.state
		FROM patient_receipt as pr
		INNER JOIN patient_location as pl ON pl.patient_id = pr.patient_id
		INNER JOIN sku ON sku.id = pr.sku_id
		WHERE pr.creation_timestamp >= ? AND pr.creation_timestamp < ?
		LIMIT ?`, from, to, maxItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*IncomingItem
	for rows.Next() {
		var item IncomingItem
		if err := rows.Scan(
			&item.Created,
			&item.ChargeID,
			&item.SKUType,
			&item.ReceiptID,
			&item.ItemID,
			&item.State); err != nil {
			return nil, err
		}

		items = append(items, &item)
	}

	return items, rows.Err()
}

func (d *dataAccess) OutgoingItems(from, to time.Time) ([]*OutgoingItem, error) {
	rows, err := d.db.Query(`
		SELECT dt.created, sku.type, pr.receipt_reference_id, dt.item_id, pl.state, concat(d.first_name, ' ', d.last_name) as doctor_name
		FROM doctor_transaction as dt
		INNER JOIN patient_receipt as pr ON pr.item_id = dt.item_id AND pr.sku_id = dt.sku_id
		INNER JOIN sku ON sku.id = dt.sku_id
		INNER JOIN patient_location as pl on pl.patient_id = dt.patient_id
		INNER JOIN doctor as d ON dt.doctor_id = d.id
		WHERE dt.created >= ? AND dt.created < ?
		LIMIT ?`, from, to, maxItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*OutgoingItem
	for rows.Next() {
		var item OutgoingItem
		if err := rows.Scan(
			&item.Created,
			&item.SKUType,
			&item.ReceiptID,
			&item.ItemID,
			&item.State,
			&item.Name); err != nil {
			return nil, err
		}

		items = append(items, &item)
	}

	return items, rows.Err()
}
