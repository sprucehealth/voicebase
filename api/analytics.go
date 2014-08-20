package api

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) AnalyticsReport(id int64) (*common.AnalyticsReport, error) {
	var rep common.AnalyticsReport
	if err := d.db.QueryRow(
		`SELECT id, owner_account_id, name, query, presentation, created, modified
		FROM analytics_report
		WHERE id = ?`, id,
	).Scan(
		&rep.ID, &rep.OwnerAccountID, &rep.Name, &rep.Query,
		&rep.Presentation, &rep.Created, &rep.Modified,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (d *DataService) ListAnalyticsReports() ([]*common.AnalyticsReport, error) {
	rows, err := d.db.Query(`
		SELECT id, owner_account_id, name, created, modified
		FROM analytics_report
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*common.AnalyticsReport
	for rows.Next() {
		rep := &common.AnalyticsReport{}
		if err := rows.Scan(&rep.ID, &rep.OwnerAccountID, &rep.Name, &rep.Created, &rep.Modified); err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}

	return reports, rows.Err()
}

func (d *DataService) CreateAnalyticsReport(ownerAccountID int64, name, query, presentation string) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO analytics_report (owner_account_id, name, query, presentation)
		VALUES (?, ?, ?, ?)`,
		ownerAccountID, name, query, presentation)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) UpdateAnalyticsReport(id int64, name, query, presentation *string) error {
	var cols []string
	var vals []interface{}
	if name != nil {
		cols = append(cols, "name = ?")
		vals = append(vals, *name)
	}
	if query != nil {
		cols = append(cols, "query = ?")
		vals = append(vals, *query)
	}
	if presentation != nil {
		cols = append(cols, "presentation = ?")
		vals = append(vals, *presentation)
	}
	if len(cols) == 0 {
		return nil
	}
	vals = append(vals, id)
	_, err := d.db.Exec("UPDATE analytics_report SET "+strings.Join(cols, ", ")+" WHERE id = ?", vals...)
	if err == sql.ErrNoRows {
		err = NoRowsError
	}
	return err
}
