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
	row := d.db.QueryRow(`SELECT id, section_id, ordinal, title, photo_url, layout FROM resource_guide WHERE id = ?`, id)
	err := row.Scan(
		&guide.Id,
		&guide.SectionId,
		&guide.Ordinal,
		&guide.Title,
		&guide.PhotoURL,
		&layout,
	)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(layout, &guide.Layout); err != nil {
		return nil, err
	}
	return &guide, nil
}

func (d *DataService) ListResourceGuideSections() ([]*common.ResourceGuideSection, error) {
	rows, err := d.db.Query(`SELECT id, ordinal, title FROM resource_guide_section ORDER BY ordinal`)
	if err != nil {
		return nil, err
	}
	defer rows.Next()
	var sections []*common.ResourceGuideSection
	for rows.Next() {
		var sec common.ResourceGuideSection
		err := rows.Scan(
			&sec.Id,
			&sec.Ordinal,
			&sec.Title,
		)
		if err != nil {
			return nil, err
		}
		sections = append(sections, &sec)
	}
	return sections, nil
}

func (d *DataService) ListResourceGuides() ([]*common.ResourceGuideSection, map[int64][]*common.ResourceGuide, error) {
	sections, err := d.ListResourceGuideSections()
	if err != nil {
		return nil, nil, err
	}

	rows, err := d.db.Query(`SELECT id, section_id, ordinal, title, photo_url FROM resource_guide ORDER BY ordinal`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Next()
	guides := map[int64][]*common.ResourceGuide{}
	for rows.Next() {
		var guide common.ResourceGuide
		err := rows.Scan(
			&guide.Id,
			&guide.SectionId,
			&guide.Ordinal,
			&guide.Title,
			&guide.PhotoURL,
		)
		if err != nil {
			return nil, nil, err
		}
		guides[guide.SectionId] = append(guides[guide.SectionId], &guide)
	}
	return sections, guides, nil
}

func (d *DataService) CreateResourceGuideSection(sec *common.ResourceGuideSection) (int64, error) {
	if sec.Title == "" || sec.Ordinal == 0 {
		return 0, fmt.Errorf("api.CreateResourceGuideSection: Title and Ordinal may not be empty")
	}
	res, err := d.db.Exec("INSERT INTO resource_guide_section (title, ordinal) VALUES (?, ?)", sec.Title, sec.Ordinal)
	if err != nil {
		return 0, err
	}
	sec.Id, err = res.LastInsertId()
	return sec.Id, err
}

func (d *DataService) UpdateResourceGuideSection(sec *common.ResourceGuideSection) error {
	if sec.Id <= 0 {
		return fmt.Errorf("api.UpdateResourceGuideSection: ID may not be 0")
	}
	var columns []string
	var values []interface{}
	if sec.Title != "" {
		columns = append(columns, "title = ?")
		values = append(values, sec.Title)
	}
	if sec.Ordinal > 0 {
		columns = append(columns, "ordinal = ?")
		values = append(values, sec.Ordinal)
	}
	if len(columns) == 0 {
		return fmt.Errorf("api.UpdateResourceGuideSection: nothing to update")
	}
	values = append(values, sec.Id)
	_, err := d.db.Exec("UPDATE resource_guide_section SET "+strings.Join(columns, ",")+" WHERE id = ?", values...)
	return err
}

func (d *DataService) CreateResourceGuide(guide *common.ResourceGuide) (int64, error) {
	if guide.Title == "" || guide.PhotoURL == "" || guide.Layout == nil {
		return 0, fmt.Errorf("api.CreateResourceGuide: Title, PhotoURL, and Layout may not be empty")
	}
	layout, err := json.Marshal(guide.Layout)
	if err != nil {
		return 0, err
	}
	res, err := d.db.Exec("INSERT INTO resource_guide (title, section_id, ordinal, photo_url, layout) VALUES (?, ?, ?, ?, ?)",
		guide.Title, guide.SectionId, guide.Ordinal, guide.PhotoURL, layout)
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
	if guide.SectionId != 0 {
		columns = append(columns, "section_id = ?")
		values = append(values, guide.SectionId)
	}
	if guide.Ordinal > 0 {
		columns = append(columns, "ordinal = ?")
		values = append(values, guide.Ordinal)
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
