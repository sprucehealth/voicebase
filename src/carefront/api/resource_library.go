package api

import (
	"carefront/common"
	"database/sql"
)

func (d *DataService) GetResourceGuide(id int64) (*common.ResourceGuide, error) {
	var guide common.ResourceGuide
	row := d.db.QueryRow(`SELECT id, title, photo_url, layout FROM resource_guide WHERE id = ?`, id)
	err := row.Scan(
		&guide.Id,
		&guide.Title,
		&guide.PhotoURL,
		&guide.Layout,
	)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	}
	return &guide, nil
}

func (d *DataService) ListResourceGuides() ([]*common.ResourceGuide, error) {
	rows, err := d.db.Query(`SELECT id, title, photo_url FROM resource_guide ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Next()
	var guides []*common.ResourceGuide
	for rows.Next() {
		var guide common.ResourceGuide
		err := rows.Scan(
			&guide.Id,
			&guide.Title,
			&guide.PhotoURL,
		)
		if err != nil {
			return nil, err
		}
		guides = append(guides, &guide)
	}
	return guides, nil
}
