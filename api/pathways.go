package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) Pathway(id int64, opts PathwayOption) (*common.Pathway, error) {
	if opts&POWithDetails != 0 {
		return scanPathway(opts,
			d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status, details_json FROM clinical_pathway WHERE id = ?`, id))
	}
	return scanPathway(opts,
		d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status FROM clinical_pathway WHERE id = ?`, id))
}

func (d *DataService) PathwayForTag(tag string, opts PathwayOption) (*common.Pathway, error) {
	if opts&POWithDetails != 0 {
		return scanPathway(opts,
			d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status, details_json FROM clinical_pathway WHERE tag = ?`, tag))
	}
	return scanPathway(opts,
		d.db.QueryRow(`SELECT id, tag, name, medicine_branch, status FROM clinical_pathway WHERE tag = ?`, tag))
}

func (d *DataService) Pathways(ids []int64, opts PathwayOption) (map[int64]*common.Pathway, error) {
	var withDetailsQuery string
	if opts&POWithDetails != 0 {
		withDetailsQuery = ", details_json"
	}
	rows, err := d.db.Query(`
		SELECT id, tag, name, medicine_branch, status`+withDetailsQuery+`
		FROM clinical_pathway
		WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pathways := make(map[int64]*common.Pathway)
	for rows.Next() {
		p, err := scanPathway(opts, rows)
		if err != nil {
			return nil, err
		}
		pathways[p.ID] = p
	}
	return pathways, rows.Err()
}

func (d *DataService) ListPathways(opts PathwayOption) ([]*common.Pathway, error) {
	var withDetailsQuery string
	if opts&POWithDetails != 0 {
		withDetailsQuery = ", details_json"
	}
	var rows *sql.Rows
	var err error
	if opts&POActiveOnly != 0 {
		rows, err = d.db.Query(`
			SELECT id, tag, name, medicine_branch, status`+withDetailsQuery+`
			FROM clinical_pathway
			WHERE status = ?
			ORDER BY name`, common.PathwayActive.String())
	} else {
		rows, err = d.db.Query(`
			SELECT id, tag, name, medicine_branch, status` + withDetailsQuery + `
			FROM clinical_pathway
			ORDER BY name`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pathways []*common.Pathway
	for rows.Next() {
		p, err := scanPathway(opts, rows)
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
	var detailsJS []byte
	if pathway.Details != nil {
		var err error
		detailsJS, err = json.Marshal(pathway.Details)
		if err != nil {
			return err
		}
	}
	res, err := d.db.Exec(`
		INSERT INTO clinical_pathway (tag, name, medicine_branch, status, details_json)
		VALUES (?, ?, ?, ?, ?)`,
		pathway.Tag, pathway.Name, pathway.MedicineBranch, pathway.Status.String(), detailsJS)
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

func (d *DataService) UpdatePathway(id int64, details *common.PathwayDetails) error {
	js, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE clinical_pathway SET details_json = ? WHERE id = ?`, js, id)
	return err
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

func (d *DataService) DoctorsForPathway(pathwayID int64, limit int) ([]*common.Doctor, error) {
	if limit <= 0 {
		return nil, nil
	}
	// Arbitrary limit we should never hit to make sure we don't blow things up
	if limit > 100 {
		limit = 100
	}
	rows, err := d.db.Query(`
		SELECT provider_id
		FROM care_providing_state cps
		INNER JOIN care_provider_state_elligibility cpse ON cpse.care_providing_state_id = cps.id
		WHERE role_type_id = ?
			AND clinical_pathway_id = ?
		LIMIT ?`,
		d.roleTypeMapping[DOCTOR_ROLE],
		pathwayID,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	doctors := make([]*common.Doctor, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		// TODO: This is pretty inefficient but there's no good alternative at
		// the moment. Hopefully this won't be too terrible.
		dr, err := d.Doctor(id, true)
		if err != nil {
			return nil, err
		}
		doctors = append(doctors, dr)
	}
	return doctors, rows.Err()
}

func scanPathway(opts PathwayOption, row scannable) (*common.Pathway, error) {
	p := &common.Pathway{}
	if opts&POWithDetails == 0 {
		err := row.Scan(&p.ID, &p.Tag, &p.Name, &p.MedicineBranch, &p.Status)
		if err == sql.ErrNoRows {
			return nil, ErrNotFound("clinical_pathway")
		} else if err != nil {
			return nil, err
		}
	} else {
		var js []byte
		err := row.Scan(&p.ID, &p.Tag, &p.Name, &p.MedicineBranch, &p.Status, &js)
		if err == sql.ErrNoRows {
			return nil, ErrNotFound("clinical_pathway")
		} else if err != nil {
			return nil, err
		}
		if js != nil {
			if err := json.Unmarshal(js, &p.Details); err != nil {
				return nil, fmt.Errorf("parsing failed for pathway details %d: %s", p.ID, err)
			}
		}
	}
	return p, nil
}
