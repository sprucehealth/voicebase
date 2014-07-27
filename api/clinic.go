package api

import "github.com/sprucehealth/backend/common"

func (d *DataService) GetAllDoctorsInClinic() ([]*common.Doctor, error) {
	rows, err := d.db.Query(`select id from doctor`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doctorIds := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		doctorIds = append(doctorIds, id)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	doctors := make([]*common.Doctor, len(doctorIds))
	for i, doctorId := range doctorIds {
		doctors[i], err = d.GetDoctorFromId(doctorId)
		if err != nil {
			return nil, err
		}
	}

	return doctors, nil
}
