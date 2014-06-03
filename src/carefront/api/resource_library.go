package api

import (
	"carefront/common"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func (d *DataService) GetResourceGuide(id int64) (*common.ResourceGuide, error) {
	var guide common.ResourceGuide
	var layout []byte
	row := d.db.QueryRow(`SELECT id, title, photo_url, layout FROM resource_guide WHERE id = ?`, id)
	err := row.Scan(
		&guide.Id,
		&guide.Title,
		&guide.PhotoURL,
		&layout,
	)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	}
	if err := json.Unmarshal(layout, &guide.Layout); err != nil {
		return nil, err
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

func (d *DataService) CreateResourceGuide(guide *common.ResourceGuide) (int64, error) {
	if guide.Title == "" || guide.PhotoURL == "" || guide.Layout == nil {
		return 0, fmt.Errorf("api.CreateResourceGuide: Title, PhotoURL, and Layout may not be empty")
	}
	layout, err := json.Marshal(guide.Layout)
	if err != nil {
		return 0, err
	}
	res, err := d.db.Exec("INSERT INTO resource_guide (title, photo_url, layout) VALUES (?, ?, ?)", guide.Title, guide.PhotoURL, layout)
	if err != nil {
		return 0, err
	}
	guide.Id, err = res.LastInsertId()
	return guide.Id, err
}

func (d *DataService) UpdateResourceGuide(guide *common.ResourceGuide) error {
	if guide.Id <= 0 {
		return fmt.Errorf("api.UpdateResourceGuide: ID may not be 0")
	}
	var columns []string
	var values []interface{}
	if guide.Title != "" {
		columns = append(columns, "title = ?")
		values = append(values, guide.Title)
	}
	if guide.PhotoURL != "" {
		columns = append(columns, "photo_url = ?")
		values = append(values, guide.PhotoURL)
	}
	if guide.Layout != nil {
		columns = append(columns, "layout = ?")
		b, err := json.Marshal(guide.Layout)
		if err != nil {
			return err
		}
		values = append(values, b)
	}
	if len(columns) == 0 {
		return fmt.Errorf("api.UpdateResourceGuide: nothing to update")
	}
	values = append(values, guide.Id)
	_, err := d.db.Exec("UPDATE resource_guide SET "+strings.Join(columns, ",")+" WHERE id = ?", values...)
	return err
}
