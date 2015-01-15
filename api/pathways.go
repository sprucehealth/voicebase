package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/sprucehealth/backend/libs/dbutil"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) Pathway(id int64) (*common.Pathway, error) {
	return scanPathway(
		d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status FROM clinical_pathway WHERE id = ?`, id))
}

func (d *DataService) PathwayForTag(tag string) (*common.Pathway, error) {
	return scanPathway(
		d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status FROM clinical_pathway WHERE tag = ?`, tag))
}

func (d *DataService) Pathways(ids []int64) (map[int64]*common.Pathway, error) {
	rows, err := d.db.Query(`
			SELECT id, tag, name, medicine_branch, status
			FROM clinical_pathway
			WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pathways := make(map[int64]*common.Pathway)
	for rows.Next() {
		p, err := scanPathway(rows)
		if err != nil {
			return nil, err
		}
		pathways[p.ID] = p
	}
	return pathways, rows.Err()
}

func (d *DataService) ListPathways(activeOnly bool) ([]*common.Pathway, error) {
	var rows *sql.Rows
	var err error
	if activeOnly {
		rows, err = d.db.Query(`
			SELECT id, tag, name, medicine_branch, status
			FROM clinical_pathway
			WHERE status = ?
			ORDER BY id`, common.PathwayActive)
	} else {
		rows, err = d.db.Query(`
			SELECT id, tag, name, medicine_branch, status
			FROM clinical_pathway
			ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pathways []*common.Pathway
	for rows.Next() {
		p, err := scanPathway(rows)
		if err != nil {
			return nil, err
		}
		pathways = append(pathways, p)
	}
	return pathways, rows.Err()
}

func (d *DataService) CreatePathway(pathway *common.Pathway) error {
	if pathway.Tag == "" {
		return errors.New("pathway tag required")
	}
	if pathway.Name == "" {
		return errors.New("pathway name required")
	}
	if pathway.MedicineBranch == "" {
		return errors.New("pathway medicine branch required")
	}
	if pathway.Status == "" {
		return errors.New("pathway status required")
	}
	res, err := d.db.Exec(`
		INSERT INTO clinical_pathway (tag, name, medicine_branch, status)
		VALUES (?, ?, ?, ?)`,
		pathway.Tag, pathway.Name, pathway.MedicineBranch, pathway.Status.String())
	if err != nil {
		return err
	}
	pathway.ID, err = res.LastInsertId()
	return err
}

func (d *DataService) PathwayMenu() (*common.PathwayMenu, error) {
	var js []byte
	row := d.db.QueryRow(`
		SELECT json
		FROM clinical_pathway_menu
		WHERE status = ?
		ORDER BY created DESC
		LIMIT 1`, STATUS_ACTIVE)
	if err := row.Scan(&js); err == sql.ErrNoRows {
		return nil, ErrNotFound("clinical_pathway_menu")
	} else if err != nil {
		return nil, err
	}
	menu := &common.PathwayMenu{}
	return menu, json.Unmarshal(js, menu)
}

func (d *DataService) UpdatePathwayMenu(menu *common.PathwayMenu) error {
	js, err := json.Marshal(menu)
	if err != nil {
		return err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`UPDATE clinical_pathway_menu SET status = ? WHERE status = ?`,
		STATUS_INACTIVE, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO clinical_pathway_menu (json, status, created)
		VALUES (?, ?, ?)`, js, STATUS_ACTIVE, time.Now().UTC())
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func scanPathway(row scannable) (*common.Pathway, error) {
	p := &common.Pathway{}
	err := row.Scan(&p.ID, &p.Tag, &p.Name, &p.MedicineBranch, &p.Status)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("clinical_pathway")
	}
	return p, err
}
