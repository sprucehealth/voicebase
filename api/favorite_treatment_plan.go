package api

import (
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

func (d *DataService) GetFavoriteTreatmentPlansForDoctor(doctorId int64) ([]*common.FavoriteTreatmentPlan, error) {
	rows, err := d.db.Query(`select id from dr_favorite_treatment_plan where doctor_id = ?`, doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	favoriteTreatmentPlanIds := make([]int64, 0)
	for rows.Next() {
		var favoriteTreatmentPlanId int64
		err := rows.Scan(&favoriteTreatmentPlanId)
		if err != nil {
			return nil, err
		}

		favoriteTreatmentPlanIds = append(favoriteTreatmentPlanIds, favoriteTreatmentPlanId)
	}

	favoriteTreatmentPlans := make([]*common.FavoriteTreatmentPlan, len(favoriteTreatmentPlanIds))
	for i, favoriteTreatmentPlanId := range favoriteTreatmentPlanIds {
		favoriteTreatmentPlan, err := d.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
		if err != nil {
			return nil, err
		}
		favoriteTreatmentPlans[i] = favoriteTreatmentPlan
	}

	return favoriteTreatmentPlans, rows.Err()
}

func (d *DataService) GetFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.FavoriteTreatmentPlan, error) {
	var favoriteTreatmentPlan common.FavoriteTreatmentPlan
	err := d.db.QueryRow(`select id, name, modified_date, doctor_id from dr_favorite_treatment_plan where id=?`, favoriteTreatmentPlanId).Scan(&favoriteTreatmentPlan.Id, &favoriteTreatmentPlan.Name, &favoriteTreatmentPlan.ModifiedDate, &favoriteTreatmentPlan.DoctorId)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	favoriteTreatmentPlan.TreatmentList = &common.TreatmentList{}
	favoriteTreatmentPlan.TreatmentList.Treatments, err = d.GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err != nil {
		return nil, err
	}

	favoriteTreatmentPlan.RegimenPlan, err = d.GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err != nil {
		return nil, err
	}

	favoriteTreatmentPlan.Advice, err = d.GetAdviceInFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err != nil {
		return nil, err
	}

	return &favoriteTreatmentPlan, err
}

func (d *DataService) CreateOrUpdateFavoriteTreatmentPlan(favoriteTreatmentPlan *common.FavoriteTreatmentPlan, treatmentPlanId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// If updating treatment plan, delete all items that currently make up this favorited treatment plan
	if favoriteTreatmentPlan.Id.Int64() != 0 {
		if err := deleteComponentsOfFavoriteTreatmentPlan(tx, favoriteTreatmentPlan.Id.Int64()); err != nil {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec(`update dr_favorite_treatment_plan set name=? where id=?`, favoriteTreatmentPlan.Name, favoriteTreatmentPlan.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		lastInsertId, err := tx.Exec(`insert into dr_favorite_treatment_plan (name, doctor_id) values (?,?)`, favoriteTreatmentPlan.Name, favoriteTreatmentPlan.DoctorId)
		if err != nil {
			tx.Rollback()
			return err
		}

		favoriteTreatmentPlanId, err := lastInsertId.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
		favoriteTreatmentPlan.Id = encoding.NewObjectId(favoriteTreatmentPlanId)
	}

	// Add all treatments
	if favoriteTreatmentPlan.TreatmentList != nil {
		for _, treatment := range favoriteTreatmentPlan.TreatmentList.Treatments {
			params := make(map[string]interface{})
			params["dr_favorite_treatment_plan_id"] = favoriteTreatmentPlan.Id.Int64()
			err := d.addTreatment(doctorFavoriteTreatmentType, treatment, params, tx)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Add regimen plan
	if favoriteTreatmentPlan.RegimenPlan != nil {
		for _, regimenSection := range favoriteTreatmentPlan.RegimenPlan.Sections {
			for _, regimenStep := range regimenSection.Steps {

				cols := "dr_favorite_treatment_plan_id, regimen_type, text, status"
				values := []interface{}{favoriteTreatmentPlan.Id.Int64(), regimenSection.Name, regimenStep.Text, STATUS_ACTIVE}
				if regimenStep.ParentID.Int64() > 0 {
					cols += ", dr_regimen_step_id"
					values = append(values, regimenStep.ParentID.Int64())
				}

				_, err = tx.Exec(`insert into dr_favorite_regimen (`+cols+`) values (`+nReplacements(len(values))+`)`, values...)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// add avice
	if favoriteTreatmentPlan.Advice != nil {
		for _, advicePoint := range favoriteTreatmentPlan.Advice.SelectedAdvicePoints {
			_, err = tx.Exec(`
				INSERT INTO dr_favorite_advice (dr_favorite_treatment_plan_id, dr_advice_point_id, text, status)
				VALUES (?, ?, ?, ?)`,
				favoriteTreatmentPlan.Id.Int64(), advicePoint.ParentID.Int64(),
				advicePoint.Text, STATUS_ACTIVE)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if treatmentPlanId > 0 {
		_, err := tx.Exec(`
			REPLACE INTO treatment_plan_content_source (treatment_plan_id, content_source_id, content_source_type, doctor_id)
			VALUES (?,?,?,?)`,
			treatmentPlanId, favoriteTreatmentPlan.Id.Int64(),
			common.TPContentSourceTypeFTP, favoriteTreatmentPlan.DoctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanID, doctorID int64) error {
	// ensure that the doctor owns the favorite treatment plan before deleting it
	var doctorIDFromFTP int64
	err := d.db.QueryRow(`
		select doctor_id from dr_favorite_treatment_plan where id = ?`, favoriteTreatmentPlanID).Scan(&doctorIDFromFTP)
	if err == sql.ErrNoRows {
		return NoRowsError
	} else if err != nil {
		return err
	} else if doctorID != doctorIDFromFTP {
		return fmt.Errorf("Doctor is not the owner of the favorite tretment plan")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// delete any content source information for treatment plans that may have selected this treatment plan as its
	// content source
	_, err = tx.Exec(`delete from treatment_plan_content_source where content_source_type = ? and content_source_id = ?`, common.TPContentSourceTypeFTP, favoriteTreatmentPlanID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`delete from dr_favorite_treatment_plan where id=?`, favoriteTreatmentPlanID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(`select dr_favorite_treatment.id,  drug_internal_name, dosage_strength, type, 
				dispense_value, dispense_unit_id, ltext, refills, substitutions_allowed,
				days_supply, pharmacy_notes, patient_instructions, creation_date, status,
				drug_name.name, drug_route.name, drug_form.name
			 		from dr_favorite_treatment 
						inner join dispense_unit on dr_favorite_treatment.dispense_unit_id = dispense_unit.id
						inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
						left outer join drug_name on drug_name_id = drug_name.id
						left outer join drug_route on drug_route_id = drug_route.id
						left outer join drug_form on drug_form_id = drug_form.id
			 					where status=? and dr_favorite_treatment_plan_id = ? and localized_text.language_id=?`,
		common.TStatusCreated.String(), favoriteTreatmentPlanId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		var treatment common.Treatment
		var medicationType string
		var drugName, drugForm, drugRoute sql.NullString
		err := rows.Scan(&treatment.Id, &treatment.DrugInternalName, &treatment.DosageStrength, &medicationType, &treatment.DispenseValue, &treatment.DispenseUnitId, &treatment.DispenseUnitDescription,
			&treatment.NumberRefills, &treatment.SubstitutionsAllowed, &treatment.DaysSupply, &treatment.PharmacyNotes, &treatment.PatientInstructions, &treatment.CreationDate, &treatment.Status,
			&drugName, &drugRoute, &drugForm)
		if err != nil {
			return nil, err
		}
		treatment.DrugName = drugName.String
		treatment.DrugForm = drugForm.String
		treatment.DrugRoute = drugRoute.String
		treatment.OTC = medicationType == treatmentOTC

		err = d.fillInDrugDBIdsForTreatment(&treatment, treatment.Id.Int64(), possibleTreatmentTables[doctorFavoriteTreatmentType])
		if err != nil {
			return nil, err
		}
		treatments = append(treatments, &treatment)
	}

	return treatments, rows.Err()
}

func (d *DataService) GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.RegimenPlan, error) {
	regimenPlanRows, err := d.db.Query(`select id, regimen_type, dr_regimen_step_id, text 
								from dr_favorite_regimen where dr_favorite_treatment_plan_id = ? and status = 'ACTIVE' order by id`, favoriteTreatmentPlanId)
	if err != nil {
		return nil, err
	}
	defer regimenPlanRows.Close()

	return getRegimenPlanFromRows(regimenPlanRows)
}

func (d *DataService) GetAdviceInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.Advice, error) {
	advicePointsRows, err := d.db.Query(`select id, dr_advice_point_id, text from dr_favorite_advice 
			where dr_favorite_treatment_plan_id = ?  and status = ?`, favoriteTreatmentPlanId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer advicePointsRows.Close()

	selectedAdvicePoints, err := getAdvicePointsFromRows(advicePointsRows)
	if err != nil {
		return nil, err
	}

	return &common.Advice{
		SelectedAdvicePoints: selectedAdvicePoints,
	}, nil
}

func deleteComponentsOfFavoriteTreatmentPlan(tx *sql.Tx, favoriteTreatmentPlanId int64) error {

	_, err := tx.Exec(`delete from dr_favorite_treatment where dr_favorite_treatment_plan_id = ?`, favoriteTreatmentPlanId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`delete from dr_favorite_regimen where dr_favorite_treatment_plan_id=?`, favoriteTreatmentPlanId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`delete from dr_favorite_advice where dr_favorite_treatment_plan_id=?`, favoriteTreatmentPlanId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`delete from dr_favorite_patient_visit_follow_up where dr_favorite_treatment_plan_id=?`, favoriteTreatmentPlanId)
	if err != nil {
		return err
	}

	return nil
}
