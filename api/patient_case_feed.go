package api

import (
	"database/sql"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
)

func (d *DataService) PatientCaseFeed() ([]*common.PatientCaseFeedItem, error) {
	rows, err := d.db.Query(`
		SELECT pca.patient_case_id, pc.clinical_pathway_id, pc.name,
			COALESCE(pv.submitted_date, pv.creation_date),
			p.first_name, p.last_name, p.id, pv.id,
			d.id, d.long_display_name
		FROM patient_case_care_provider_assignment pca
		INNER JOIN doctor d ON d.id = pca.provider_id
		INNER JOIN patient_case pc ON pc.id = pca.patient_case_id
		INNER JOIN patient_visit pv ON pv.patient_case_id = pca.patient_case_id
			AND pv.status IN ('ROUTED', 'REVIEWING', 'TRIAGED', 'TREATED')
		INNER JOIN patient p ON p.id = pc.patient_id
		WHERE role_type_id = ?
		ORDER BY pv.submitted_date DESC`, d.roleTypeMapping[RoleDoctor])
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	// Track cases to only include the most recent visit
	casesSeen := make(map[int64]struct{})
	var items []*common.PatientCaseFeedItem
	for rows.Next() {
		item := &common.PatientCaseFeedItem{}
		var pathwayID int64
		if err := rows.Scan(&item.CaseID, &pathwayID, &item.PathwayName,
			&item.LastVisitTime, &item.PatientFirstName, &item.PatientLastName,
			&item.PatientID, &item.LastVisitID, &item.DoctorID, &item.LastVisitDoctor,
		); err != nil {
			return nil, errors.Trace(err)
		}
		if _, ok := casesSeen[item.CaseID]; ok {
			continue
		}
		casesSeen[item.CaseID] = struct{}{}
		item.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DataService) PatientCaseFeedForDoctor(doctorID int64) ([]*common.PatientCaseFeedItem, error) {
	var doctorName string
	err := d.db.QueryRow(`SELECT long_display_name FROM doctor WHERE id = ?`, doctorID).Scan(&doctorName)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("doctor"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	rows, err := d.db.Query(`
		SELECT pca.patient_case_id, pc.clinical_pathway_id, pc.name,
			COALESCE(pv.submitted_date, pv.creation_date),
			p.first_name, p.last_name, p.id, pv.id
		FROM patient_case_care_provider_assignment pca
		INNER JOIN patient_case pc ON pc.id = pca.patient_case_id
		INNER JOIN patient_visit pv ON pv.patient_case_id = pca.patient_case_id
			AND pv.status IN ('ROUTED', 'REVIEWING', 'TRIAGED', 'TREATED')
		INNER JOIN patient p ON p.id = pc.patient_id
		WHERE role_type_id = ? AND provider_id = ?
		ORDER BY pv.submitted_date DESC`, d.roleTypeMapping[RoleDoctor], doctorID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	// Track cases to only include the most recent visit
	casesSeen := make(map[int64]struct{})
	var items []*common.PatientCaseFeedItem
	for rows.Next() {
		item := &common.PatientCaseFeedItem{
			DoctorID:        doctorID,
			LastVisitDoctor: doctorName,
		}
		var pathwayID int64
		if err := rows.Scan(&item.CaseID, &pathwayID, &item.PathwayName,
			&item.LastVisitTime, &item.PatientFirstName, &item.PatientLastName,
			&item.PatientID, &item.LastVisitID,
		); err != nil {
			return nil, errors.Trace(err)
		}
		if _, ok := casesSeen[item.CaseID]; ok {
			continue
		}
		casesSeen[item.CaseID] = struct{}{}
		item.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
