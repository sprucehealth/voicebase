package api

import (
	"database/sql"
	"encoding/json"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) Dashboard(id int64) (*common.AdminDashboard, error) {
	dash := &common.AdminDashboard{
		ID: id,
	}
	row := d.db.QueryRow(`
		SELECT name, created_date, modified_date
		FROM admin_dashboard
		WHERE id = ?`, id)
	if err := row.Scan(&dash.Name, &dash.Created, &dash.Modified); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`
		SELECT id, ordinal, columns, type, config
		FROM admin_dashboard_panel
		WHERE admin_dashboard_id = ?
		ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		panel := &common.AdminDashboardPanel{}
		var config []byte
		if err := rows.Scan(&panel.ID, &panel.Ordinal, &panel.Columns, &panel.Type, &config); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(config, &panel.Config); err != nil {
			return nil, err
		}
		dash.Panels = append(dash.Panels, panel)
	}

	return dash, nil
}
