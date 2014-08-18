package api

import "github.com/sprucehealth/backend/common"

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
