package api

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) FavoriteTreatmentPlansForDoctor(doctorID int64, pathwayTag string) ([]*common.FavoriteTreatmentPlan, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`
		SELECT ftp.id
		FROM dr_favorite_treatment_plan ftp
		INNER JOIN dr_favorite_treatment_plan_membership ftpm ON ftp.id = ftpm.dr_favorite_treatment_plan_id
		WHERE ftpm.doctor_id = ? AND ftpm.clinical_pathway_id = ?`,
		doctorID, pathwayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	ftps := make([]*common.FavoriteTreatmentPlan, len(ids))
	for i, id := range ids {
		ftp, err := d.FavoriteTreatmentPlan(id)
		if err != nil {
			return nil, err
		}
		ftps[i] = ftp
	}

	return ftps, rows.Err()
}

func (d *DataService) FavoriteTreatmentPlan(id int64) (*common.FavoriteTreatmentPlan, error) {
	var ftp common.FavoriteTreatmentPlan
	var note sql.NullString
	err := d.db.QueryRow(`
		SELECT id, name, modified_date, creator_id, note
		FROM dr_favorite_treatment_plan
		WHERE id = ?`,
		id).Scan(&ftp.ID, &ftp.Name, &ftp.ModifiedDate, &ftp.CreatorID, &note)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("dr_favorite_treatment_plan")
	} else if err != nil {
		return nil, err
	}

	ftp.Note = note.String
	ftp.TreatmentList = &common.TreatmentList{}
	ftp.TreatmentList.Treatments, err = d.GetTreatmentsInFavoriteTreatmentPlan(id)
	if err != nil {
		return nil, err
	}

	ftp.RegimenPlan, err = d.GetRegimenPlanInFavoriteTreatmentPlan(id)
	if err != nil {
		return nil, err
	}

	ftp.ScheduledMessages, err = d.listFavoriteTreatmentPlanScheduledMessages(id)
	if err != nil {
		return nil, err
	}

	ftp.ResourceGuides, err = d.listFavoriteTreatmentPlanResourceGuides(id)
	if err != nil {
		return nil, err
	}

	return &ftp, nil
}

func (d *DataService) InsertFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan, pathwayTag string, treatmentPlanID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	pathway, err := d.PathwayForTag(pathwayTag, PONone)
	if err != nil {
		tx.Rollback()
		return err
	}

	cols := []string{"name", "creator_id", "note"}
	vals := []interface{}{ftp.Name, ftp.CreatorID, ftp.Note}

	// If updating treatment plan, delete the membership to the old FTP and create a new FTP with the new contents and a new membership
	if ftp.ID.Int64() != 0 {
		if _, err := d.deleteFTPMembership(tx, ftp.ID.Int64(), ftp.CreatorID, pathway.ID); err != nil {
			tx.Rollback()
			return err
		}
		cols = append(cols, "parent_id")
		vals = append(vals, ftp.ID.Int64())
	}

	res, err := tx.Exec(`
			INSERT INTO dr_favorite_treatment_plan (`+strings.Join(cols, ",")+`) 
			VALUES (`+dbutil.MySQLArgs(len(vals))+`)`,
		vals...)
	if err != nil {
		tx.Rollback()
		return err
	}

	favoriteTreatmentPlanID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = d.createFTPMembership(tx, favoriteTreatmentPlanID, ftp.CreatorID, pathway.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	ftp.ID = encoding.NewObjectID(favoriteTreatmentPlanID)

	// Add all treatments
	if ftp.TreatmentList != nil {
		for _, treatment := range ftp.TreatmentList.Treatments {
			params := make(map[string]interface{})
			params["dr_favorite_treatment_plan_id"] = ftp.ID.Int64()
			err := d.addTreatment(doctorFavoriteTreatmentType, treatment, params, tx)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Add regimen plan
	if ftp.RegimenPlan != nil {
		secStmt, err := tx.Prepare(`
			INSERT INTO dr_favorite_regimen_section (dr_favorite_treatment_plan_id, title)
			VALUES (?,?)`)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer secStmt.Close()
		for _, section := range ftp.RegimenPlan.Sections {
			res, err := secStmt.Exec(ftp.ID.Int64(), section.Name)
			if err != nil {
				tx.Rollback()
				return err
			}
			sectionID, err := res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
			for _, step := range section.Steps {
				cols := "dr_favorite_treatment_plan_id, dr_favorite_regimen_section_id, text, status"
				values := []interface{}{ftp.ID.Int64(), sectionID, step.Text, STATUS_ACTIVE}
				if step.ParentID.Int64() > 0 {
					cols += ", dr_regimen_step_id"
					values = append(values, step.ParentID.Int64())
				}

				_, err = tx.Exec(`
					INSERT INTO dr_favorite_regimen (`+cols+`)
					VALUES (`+dbutil.MySQLArgs(len(values))+`)`, values...)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	if len(ftp.ResourceGuides) != 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO dr_favorite_treatment_plan_resource_guide
				(dr_favorite_treatment_plan_id, resource_guide_id)
			VALUES (?, ?)`)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()
		for _, guide := range ftp.ResourceGuides {
			_, err = stmt.Exec(ftp.ID.Int64(), guide.ID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if treatmentPlanID > 0 {
		_, err := tx.Exec(`
			REPLACE INTO treatment_plan_content_source
			(treatment_plan_id, content_source_id, content_source_type, doctor_id)
			VALUES (?,?,?,?)`,
			treatmentPlanID, ftp.ID.Int64(),
			common.TPContentSourceTypeFTP, ftp.CreatorID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanID, doctorID int64, pathwayTag string) error {
	pathway, err := d.PathwayForTag(pathwayTag, PONone)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if _, err := d.deleteFTPMembership(tx, favoriteTreatmentPlanID, doctorID, pathway.ID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) createFTPMembership(db db, ftpID, doctorID, pathwayID int64) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO dr_favorite_treatment_plan_membership
			(dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id)
			VALUES (?, ?, ?)`, ftpID, doctorID, pathwayID)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (d *DataService) deleteFTPMembership(db db, ftpID, doctorID, pathwayID int64) (int64, error) {
	res, err := db.Exec(`
		DELETE FROM dr_favorite_treatment_plan_membership
			WHERE dr_favorite_treatment_plan_id = ?
			AND doctor_id = ?
			AND clinical_pathway_id = ?`, ftpID, doctorID, pathwayID)
	if err != nil {
		return 0, err
	}

	// delete any content source information for treatment plans that may have selected this treatment plan with this doctor as its
	// content source
	_, err = db.Exec(`
		DELETE FROM treatment_plan_content_source
		WHERE content_source_type = ? 
		AND content_source_id = ?
		AND doctor_id = ?`, common.TPContentSourceTypeFTP, ftpID, doctorID)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func (d *DataService) GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(`
		SELECT dr_favorite_treatment.id,  drug_internal_name, dosage_strength, type,
			dispense_value, dispense_unit_id, ltext, refills, substitutions_allowed,
			days_supply, pharmacy_notes, patient_instructions, creation_date, status,
			drug_name.name, drug_route.name, drug_form.name
		FROM dr_favorite_treatment
		INNER JOIN dispense_unit ON dr_favorite_treatment.dispense_unit_id = dispense_unit.id
		INNER JOIN localized_text ON localized_text.app_text_id = dispense_unit.dispense_unit_text_id
		LEFT OUTER JOIN drug_name ON drug_name_id = drug_name.id
		LEFT OUTER JOIN drug_route ON drug_route_id = drug_route.id
		LEFT OUTER JOIN drug_form ON drug_form_id = drug_form.id
		WHERE status = ?
			AND dr_favorite_treatment_plan_id = ?
			AND localized_text.language_id = ?`,
		common.TStatusCreated.String(), favoriteTreatmentPlanID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		var treatment common.Treatment
		var medicationType string
		var drugName, drugForm, drugRoute sql.NullString
		err := rows.Scan(&treatment.ID, &treatment.DrugInternalName, &treatment.DosageStrength, &medicationType,
			&treatment.DispenseValue, &treatment.DispenseUnitID, &treatment.DispenseUnitDescription,
			&treatment.NumberRefills, &treatment.SubstitutionsAllowed, &treatment.DaysSupply, &treatment.PharmacyNotes,
			&treatment.PatientInstructions, &treatment.CreationDate, &treatment.Status,
			&drugName, &drugRoute, &drugForm)
		if err != nil {
			return nil, err
		}
		treatment.DrugName = drugName.String
		treatment.DrugForm = drugForm.String
		treatment.DrugRoute = drugRoute.String
		treatment.OTC = medicationType == treatmentOTC

		err = d.fillInDrugDBIdsForTreatment(&treatment, treatment.ID.Int64(), possibleTreatmentTables[doctorFavoriteTreatmentType])
		if err != nil {
			return nil, err
		}
		treatments = append(treatments, &treatment)
	}

	return treatments, rows.Err()
}

func (d *DataService) GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) (*common.RegimenPlan, error) {
	regimenPlanRows, err := d.db.Query(`
		SELECT r.id, title, dr_regimen_step_id, text
		FROM dr_favorite_regimen r
		INNER JOIN dr_favorite_regimen_section rs ON rs.id = r.dr_favorite_regimen_section_id
		WHERE r.dr_favorite_treatment_plan_id = ?
			AND status = 'ACTIVE'
		ORDER BY r.id`, favoriteTreatmentPlanID)
	if err != nil {
		return nil, err
	}
	defer regimenPlanRows.Close()

	return getRegimenPlanFromRows(regimenPlanRows)
}

func (d *DataService) listFavoriteTreatmentPlanResourceGuides(ftpID int64) ([]*common.ResourceGuide, error) {
	rows, err := d.db.Query(`
		SELECT id, section_id, ordinal, title, photo_url
		FROM dr_favorite_treatment_plan_resource_guide
		INNER JOIN resource_guide rg ON rg.id = resource_guide_id
		WHERE dr_favorite_treatment_plan_id = ?`,
		ftpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guides []*common.ResourceGuide
	for rows.Next() {
		g := &common.ResourceGuide{}
		if err := rows.Scan(&g.ID, &g.SectionID, &g.Ordinal, &g.Title, &g.PhotoURL); err != nil {
			return nil, err
		}
		guides = append(guides, g)
	}

	return guides, rows.Err()
}
