package api

import (
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) SearchDoctors(query string) ([]*common.DoctorSearchResult, error) {
	// TODO: this is VERY inefficient but works for now due to the small number of doctors.
	query = "%" + strings.ToLower(query) + "%"
	rows, err := d.db.Query(`
		SELECT doctor.id, account.id, first_name, last_name, email
		FROM doctor
		INNER JOIN account ON account.id = doctor.account_id
		WHERE lower(first_name) LIKE ? OR lower(last_name) LIKE ? OR lower(email) LIKE ?`,
		query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*common.DoctorSearchResult
	for rows.Next() {
		r := &common.DoctorSearchResult{}
		err := rows.Scan(&r.DoctorID, &r.AccountID, &r.FirstName, &r.LastName, &r.Email)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
