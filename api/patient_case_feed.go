package api

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
)

func (d *DataService) PatientCaseFeed() ([]*common.PatientCaseFeedItem, error) {
	rows, err := d.db.Query(`
		SELECT f.doctor_id, f.patient_id, f.case_id, f.clinical_pathway_id, f.clinical_pathway_name,
			f.last_visit_time, f.last_visit_doctor, f.last_event, f.last_event_time,
			f.action_url, p.first_name, p.last_name
		FROM doctor_patient_case_feed f
		INNER JOIN patient p ON p.id = f.patient_id
		ORDER BY f.last_event_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPatientCaseFeedRows(rows)
}

func (d *DataService) PatientCaseFeedForDoctor(doctorID int64) ([]*common.PatientCaseFeedItem, error) {
	rows, err := d.db.Query(`
		SELECT f.doctor_id, f.patient_id, f.case_id, f.clinical_pathway_id, f.clinical_pathway_name,
			f.last_visit_time, f.last_visit_doctor, f.last_event, f.last_event_time,
			f.action_url, p.first_name, p.last_name
		FROM doctor_patient_case_feed f
		INNER JOIN patient p ON p.id = f.patient_id
		WHERE doctor_id = ?
		ORDER BY f.last_event_time DESC`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPatientCaseFeedRows(rows)
}

func scanPatientCaseFeedRows(rows *sql.Rows) ([]*common.PatientCaseFeedItem, error) {
	var items []*common.PatientCaseFeedItem
	for rows.Next() {
		item := &common.PatientCaseFeedItem{}
		var actionURL string
		if err := rows.Scan(&item.DoctorID, &item.PatientID, &item.CaseID, &item.PathwayID,
			&item.PathwayName, &item.LastVisitTime, &item.LastVisitDoctor, &item.LastEvent,
			&item.LastEventTime, &actionURL, &item.PatientFirstName, &item.PatientLastName,
		); err != nil {
			return nil, err
		}
		// Default to viewing the case if there's no other action
		if actionURL != "" {
			sa, err := app_url.ParseSpruceAction(actionURL)
			if err != nil {
				golog.Errorf("bad spruce action URL for doctor_patient_case (%d, %d, %d): '%s'",
					item.DoctorID, item.PatientID, item.CaseID, actionURL)
			} else {
				item.ActionURL = sa
			}
		}
		if item.ActionURL.IsZero() {
			item.ActionURL = *app_url.ViewCaseAction(item.CaseID)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DataService) UpdatePatientCaseFeedItem(item *common.PatientCaseFeedItem) error {
	if item.CaseID == 0 {
		return errors.New("CaseID required when updating case feed item")
	}
	if item.PatientID == 0 {
		return errors.New("PatientID required when updating case feed item")
	}
	if item.DoctorID == 0 {
		return errors.New("DoctorID required when updating case feed item")
	}
	if item.LastEvent == "" {
		return errors.New("LastEvent required when updating case feed item")
	}

	if item.LastEventTime.IsZero() {
		item.LastEventTime = time.Now()
	}

	// Fetch denormalized fields if not provided

	if item.LastVisitTime.IsZero() {
		openStatuses := common.OpenPatientVisitStates()
		err := d.db.QueryRow(`
			SELECT COALESCE(submitted_date, creation_date)
			FROM patient_visit
			WHERE patient_case_id = ?
				AND NOT status IN (`+dbutil.MySQLArgs(len(openStatuses))+`)
			ORDER BY creation_date DESC
			LIMIT 1`,
			dbutil.AppendStringsToInterfaceSlice([]interface{}{item.CaseID}, openStatuses)...,
		).Scan(&item.LastVisitTime)
		if err == sql.ErrNoRows {
			return fmt.Errorf("no visits for case %d when trying to update patient case feed", item.CaseID)
		} else if err != nil {
			return err
		}
	}

	if item.LastVisitDoctor == "" {
		err := d.db.QueryRow(`
			SELECT d.short_display_name
			FROM patient_case_care_provider_assignment a
			INNER JOIN doctor d ON d.id = a.provider_id
			WHERE a.role_type_id = ?
				AND a.patient_case_id = ?
				AND a.status = ?`,
			d.roleTypeMapping[DOCTOR_ROLE], item.CaseID, STATUS_ACTIVE,
		).Scan(&item.LastVisitDoctor)
		if err == sql.ErrNoRows {
			return fmt.Errorf("no active doctor for case %d when trying to update patient case feed", item.CaseID)
		} else if err != nil {
			return err
		}
	}

	if item.PathwayID == 0 {
		err := d.db.QueryRow(`
			SELECT pc.clinical_pathway_id, cp.name
			FROM patient_case pc
			INNER JOIN clinical_pathway cp ON cp.id = pc.clinical_pathway_id
			WHERE pc.id = ?`, item.CaseID,
		).Scan(&item.PathwayID, &item.PathwayName)
		if err != nil {
			return err
		}
	} else if item.PathwayName == "" {
		err := d.db.QueryRow(
			`SELECT cp.name FROM clinical_pathway cp WHERE id = ?`, item.PathwayID,
		).Scan(&item.PathwayName)
		if err != nil {
			return err
		}
	}

	_, err := d.db.Exec(`
		INSERT INTO doctor_patient_case_feed (doctor_id, patient_id, case_id, clinical_pathway_id,
			clinical_pathway_name, last_visit_time, last_visit_doctor, last_event, last_event_time, action_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			last_visit_time = ?, last_visit_doctor = ?,
			last_event_time = ?, last_event = ?, action_url = ?`,
		item.DoctorID, item.PatientID, item.CaseID, item.PathwayID, &item.PathwayName,
		item.LastVisitTime, item.LastVisitDoctor, item.LastEvent, item.LastEventTime,
		item.ActionURL.String(), item.LastVisitTime, item.LastVisitDoctor,
		item.LastEventTime, item.LastEvent, item.ActionURL.String())
	return err
}
