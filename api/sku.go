package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
)

func (d *dataService) SKUForPathway(pathwayTag string, category common.SKUCategoryType) (*common.SKU, error) {

	// we assume the form of a SKU to be <pathway_tag>_<sku_category>
	skuType := pathwayTag + "_" + category.String()

	// special case acne's SKUs for legacy reasons
	// which is the only pathway that does not conform to <pathway_tag>_<sku_category>
	if pathwayTag == AcnePathwayTag {
		skuType = "acne_" + category.String()
	}

	row := d.db.QueryRow(`
		SELECT sku.id, sku.type, sku_category.type
		FROM sku
		INNER JOIN sku_category ON sku_category.id = sku_category_id
		WHERE sku.type = ?`,
		skuType)

	return scanSKU(row)
}

func (d *dataService) SKU(skuType string) (*common.SKU, error) {
	row := d.db.QueryRow(`
		SELECT sku.id, sku.type, sku_category.type
		FROM sku
		INNER JOIN sku_category ON sku_category.id = sku_category_id
		WHERE sku.type = ?`, skuType)
	return scanSKU(row)
}

func scanSKU(s scannable) (*common.SKU, error) {
	var sku common.SKU
	if err := s.Scan(&sku.ID, &sku.Type, &sku.CategoryType); err == sql.ErrNoRows {
		return nil, ErrNotFound("sku")
	} else if err != nil {
		return nil, err
	}

	return &sku, nil
}

func (d *dataService) CategoryForSKU(skuType string) (*common.SKUCategoryType, error) {

	var skuCategory common.SKUCategoryType
	if err := d.db.QueryRow(`
		SELECT sku_category.type
		FROM sku
		INNER JOIN sku_category ON sku_category.id = sku_category_id
		WHERE sku.type = ?`,
		skuType).Scan(&skuCategory); err == sql.ErrNoRows {
		return nil, ErrNotFound("sku_category")
	} else if err != nil {
		return nil, err
	}

	return &skuCategory, nil
}

func (d *dataService) CreateSKU(sku *common.SKU) (int64, error) {
	// ensure that the category exists
	var categoryID int64
	err := d.db.QueryRow(`
		SELECT id FROM sku_category WHERE type = ?`, sku.CategoryType.String()).Scan(&categoryID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound("sku_category")
	} else if err != nil {
		return 0, err
	}

	res, err := d.db.Exec(`
		INSERT INTO sku (type, sku_category_id, status) VALUES (?,?,?)`,
		sku.Type, categoryID, StatusActive)
	if err != nil {
		return 0, err
	}

	sku.ID, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return sku.ID, nil
}
