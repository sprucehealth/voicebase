package api

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dbutil"
)

// doctorFTPQueryMaxThreads is a Server configurable value for maximum number of gorutines to use in this lookup
var doctorFTPQueryMaxThreads = &cfg.ValueDef{
	Name:        "Doctor.FTP.Query.Max.Threads",
	Description: "Change the number of goroutines used in the collection of FTPs.",
	Type:        cfg.ValueTypeInt,
	Default:     25,
}

type ftpPathwaysPair struct {
	ftp      *common.FavoriteTreatmentPlan
	pathways []string
}

func (d *DataService) FavoriteTreatmentPlansForDoctor(doctorID int64, pathwayTag string) (map[string][]*common.FavoriteTreatmentPlan, error) {
	// Collect a list of FTP memberships for the given doctor
	q :=
		`SELECT ftp.id, cp.tag
      FROM dr_favorite_treatment_plan ftp
      INNER JOIN dr_favorite_treatment_plan_membership ftpm ON ftp.id = ftpm.dr_favorite_treatment_plan_id
      INNER JOIN clinical_pathway cp ON ftpm.clinical_pathway_id = cp.id
      WHERE ftpm.doctor_id = ?`
	v := []interface{}{doctorID}

	// If a pathway is specified filter our results to that set
	if pathwayTag != "" {
		pathwayID, err := d.pathwayIDFromTag(pathwayTag)
		if err != nil {
			return nil, err
		}
		q += ` AND ftpm.clinical_pathway_id = ?`
		v = append(v, pathwayID)
	}
	rows, err := d.db.Query(q, v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Since building out an FTP is complex, we'll partition it on FTP id for now
	ftpsByPathway := make(map[string][]*common.FavoriteTreatmentPlan)
	ftpPathwaysByFTPID := make(map[int64][]string)
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return nil, err
		}
		ftpPathwaysByFTPID[id] = append(ftpPathwaysByFTPID[id], tag)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// Start a go routine for each unique FTP ID we've found in our membership set
	errs := make(chan error, len(ftpPathwaysByFTPID))
	ftps := make(chan *ftpPathwaysPair, len(ftpPathwaysByFTPID))
	var totalFTPCreated int

	// We don't want to eat all the DB connections so throttle the max goroutines to a configurable value
	maxGORoutines := d.cfgStore.Snapshot().Int(doctorFTPQueryMaxThreads.Name)

	// From this point forward we cannot rely on the len of ftpPathwaysByFTPID for anything as we will be removing elements as we deal with them
	totalFTPsExpected := len(ftpPathwaysByFTPID)
	var routinesRunning int
	for totalFTPCreated < totalFTPsExpected {
		// Only create more if we aren't at our max goroutines and there are ftps remaining to process
		if routinesRunning < maxGORoutines && len(ftpPathwaysByFTPID) > 0 {
			for ftpID, pathways := range ftpPathwaysByFTPID {
				go func(ftpID int64, pathways []string) {
					favoriteTreatmentPlan, err := d.FavoriteTreatmentPlan(ftpID)
					if err != nil {
						errs <- err
						return
					}
					ftps <- &ftpPathwaysPair{ftp: favoriteTreatmentPlan, pathways: pathways}
				}(ftpID, pathways)

				// Once we have started dealing with an FTP we can remove it from our list of unique IDs so we don't double process
				// This allows us to also not have to rewalk the list of ftps that might be in flight
				delete(ftpPathwaysByFTPID, ftpID)

				// Count each goroutine we start and quit our loop if we have hit our cap
				routinesRunning++
				if routinesRunning >= maxGORoutines {
					break
				}
			}
		}

		// Note: This is dangerous if we ever have a miss, we rely on the underlying code to throw rather than return nil on a miss.
		select {
		case err := <-errs:
			return nil, err
		case ftpPair := <-ftps:
			// When we recieve an FTP via our channel map it to every pathway it has a membership for
			for _, v := range ftpPair.pathways {
				ftpsByPathway[v] = append(ftpsByPathway[v], ftpPair.ftp)
			}

			// Report that a routine has completed and increment that we have completed another FTP
			routinesRunning--
			totalFTPCreated++
		}
	}

	// Sort the FTPs in a more useable format for the caller
	// Note: In the future we could parameterize this
	for _, ftps := range ftpsByPathway {
		sort.Sort(common.FavoriteTreatmentPlanByName(ftps))
	}
	return ftpsByPathway, nil
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

func (d *DataService) GlobalFavoriteTreatmentPlans(lifecycles []string) ([]*common.FavoriteTreatmentPlan, error) {
	if len(lifecycles) == 0 {
		return nil, errors.New("No lifecycles provided for gloal FTP query. Cannot complete.")
	}
	var ftps []*common.FavoriteTreatmentPlan
	var note sql.NullString
	rows, err := d.db.Query(`
		SELECT id, name, modified_date, creator_id, note
		FROM dr_favorite_treatment_plan
		WHERE (creator_id = 0
		OR creator_id IS NULL)
		AND lifecycle IN (`+dbutil.MySQLArgs(len(lifecycles))+`)
		ORDER BY name ASC`, dbutil.AppendStringsToInterfaceSlice(nil, lifecycles)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ftp := &common.FavoriteTreatmentPlan{}
		err := rows.Scan(&ftp.ID, &ftp.Name, &ftp.ModifiedDate, &ftp.CreatorID, &note)
		if err != nil {
			return nil, err
		}

		ftp.Note = note.String
		ftp.TreatmentList = &common.TreatmentList{}
		ftp.TreatmentList.Treatments, err = d.GetTreatmentsInFavoriteTreatmentPlan(ftp.ID.Int64())
		if err != nil {
			return nil, err
		}

		ftp.RegimenPlan, err = d.GetRegimenPlanInFavoriteTreatmentPlan(ftp.ID.Int64())
		if err != nil {
			return nil, err
		}

		ftp.ScheduledMessages, err = d.listFavoriteTreatmentPlanScheduledMessages(ftp.ID.Int64())
		if err != nil {
			return nil, err
		}

		ftp.ResourceGuides, err = d.listFavoriteTreatmentPlanResourceGuides(ftp.ID.Int64())
		if err != nil {
			return nil, err
		}
		ftps = append(ftps, ftp)
	}

	return ftps, rows.Err()
}

func (d *DataService) InsertFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan, pathwayTag string, treatmentPlanID int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	id, err := d.insertFavoriteTreatmentPlan(tx, ftp, pathwayTag, treatmentPlanID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit()
}

func (d *DataService) insertFavoriteTreatmentPlan(db db, ftp *common.FavoriteTreatmentPlan, pathwayTag string, treatmentPlanID int64) (int64, error) {
	if ftp.Lifecycle == "" {
		ftp.Lifecycle = "ACTIVE"
	}

	pathway, pathwayErr := d.PathwayForTag(pathwayTag, PONone)
	cols := []string{"name", "creator_id", "note", "lifecycle"}
	vals := []interface{}{ftp.Name, ftp.CreatorID, ftp.Note, ftp.Lifecycle}

	// If updating treatment plan, delete the membership to the old FTP and create a new FTP with the new contents and a new membership
	if ftp.ID.Int64() != 0 {
		if pathwayErr != nil {
			return 0, pathwayErr
		}
		if _, err := d.deleteFTPMembership(db, ftp.ID.Int64(), *ftp.CreatorID, pathway.ID); err != nil {
			return 0, err
		}
		cols = append(cols, "parent_id")
		vals = append(vals, ftp.ID.Int64())
	}

	res, err := db.Exec(`
			INSERT INTO dr_favorite_treatment_plan (`+strings.Join(cols, ",")+`)
			VALUES (`+dbutil.MySQLArgs(len(vals))+`)`,
		vals...)
	if err != nil {
		return 0, err
	}

	favoriteTreatmentPlanID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if ftp.CreatorID != nil {
		if pathwayErr != nil {
			return 0, pathwayErr
		}
		_, err = d.createFTPMembership(db, favoriteTreatmentPlanID, *ftp.CreatorID, pathway.ID)
		if err != nil {
			return 0, err
		}
	}

	ftp.ID = encoding.NewObjectID(favoriteTreatmentPlanID)

	// Add all treatments
	if ftp.TreatmentList != nil {
		for _, treatment := range ftp.TreatmentList.Treatments {
			params := make(map[string]interface{})
			params["dr_favorite_treatment_plan_id"] = ftp.ID.Int64()
			err := d.addTreatment(doctorFavoriteTreatmentType, treatment, params, db)
			if err != nil {
				return 0, err
			}
		}
	}

	// Add regimen plan
	if ftp.RegimenPlan != nil {
		secStmt, err := db.Prepare(`
			INSERT INTO dr_favorite_regimen_section (dr_favorite_treatment_plan_id, title)
			VALUES (?,?)`)
		if err != nil {
			return 0, err
		}
		defer secStmt.Close()
		for _, section := range ftp.RegimenPlan.Sections {
			res, err := secStmt.Exec(ftp.ID.Int64(), section.Name)
			if err != nil {
				return 0, err
			}
			sectionID, err := res.LastInsertId()
			if err != nil {
				return 0, err
			}
			for _, step := range section.Steps {
				cols := "dr_favorite_treatment_plan_id, dr_favorite_regimen_section_id, text, status"
				values := []interface{}{ftp.ID.Int64(), sectionID, step.Text, StatusActive}
				if step.ParentID.Int64() > 0 {
					cols += ", dr_regimen_step_id"
					values = append(values, step.ParentID.Int64())
				}

				_, err = db.Exec(`
					INSERT INTO dr_favorite_regimen (`+cols+`)
					VALUES (`+dbutil.MySQLArgs(len(values))+`)`, values...)
				if err != nil {
					return 0, err
				}
			}
		}
	}

	if len(ftp.ResourceGuides) != 0 {
		stmt, err := db.Prepare(`
			INSERT INTO dr_favorite_treatment_plan_resource_guide
				(dr_favorite_treatment_plan_id, resource_guide_id)
			VALUES (?, ?)`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()
		for _, guide := range ftp.ResourceGuides {
			_, err = stmt.Exec(ftp.ID.Int64(), guide.ID)
			if err != nil {
				return 0, err
			}
		}
	}

	if treatmentPlanID > 0 {
		_, err := db.Exec(`
			REPLACE INTO treatment_plan_content_source
			(treatment_plan_id, content_source_id, content_source_type, doctor_id)
			VALUES (?,?,?,?)`,
			treatmentPlanID, ftp.ID.Int64(),
			common.TPContentSourceTypeFTP, ftp.CreatorID)
		if err != nil {
			return 0, err
		}
	}

	return favoriteTreatmentPlanID, nil
}

// Note: This should be removed once we remove the dependency on the spreadsheet
func (d *DataService) InsertGlobalFTPsAndUpdateMemberships(ftpsByPathwayID map[int64][]*common.FavoriteTreatmentPlan) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// Clean up memberships to the old FTPs
	rows, err := tx.Query(`
		SELECT dr_favorite_treatment_plan_membership.id, dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id
			FROM dr_favorite_treatment_plan_membership
			JOIN dr_favorite_treatment_plan on dr_favorite_treatment_plan.id = dr_favorite_treatment_plan_membership.dr_favorite_treatment_plan_id
			WHERE creator_id is null`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var memberships []*common.FTPMembership
	for rows.Next() {
		membership := &common.FTPMembership{}
		if err = rows.Scan(&membership.ID, &membership.DoctorFavoritePlanID, &membership.DoctorID, &membership.ClinicalPathwayID); err != nil {
			tx.Rollback()
			return err
		}
		memberships = append(memberships, membership)
	}

	for _, membership := range memberships {
		_, err = d.deleteFTPMembership(tx, membership.DoctorFavoritePlanID, membership.DoctorID, membership.ClinicalPathwayID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Deprecate the old global FTP's
	_, err = tx.Exec(`
		UPDATE dr_favorite_treatment_plan
			SET lifecycle = 'DEPRECATED'
			WHERE creator_id IS NULL`)
	if err != nil {
		tx.Rollback()
		return err
	}

	threads := len(ftpsByPathwayID)
	done, errs := make(chan bool, len(ftpsByPathwayID)), make(chan error, len(ftpsByPathwayID))
	for pathwayID, ftps := range ftpsByPathwayID {
		go func(threadLocalPathwayID int64, threadLocalFTPs []*common.FavoriteTreatmentPlan, done chan bool, errs chan error) {
			for _, ftp := range threadLocalFTPs {
				if ftp.CreatorID != nil || ftp.ID.Int64() != 0 || ftp.ParentID != nil {
					errs <- fmt.Errorf("Cannot insert FTP as global that already has an ID - %v, CreatorID - %v, or ParentID - %v", ftp.ID, ftp.CreatorID, ftp.ParentID)
					return
				}
				ftpID, err := d.insertFavoriteTreatmentPlan(tx, ftp, "", 0)
				if err != nil {
					errs <- err
					return
				}

				for _, sm := range ftp.ScheduledMessages {
					sm.TreatmentPlanID = ftpID
					_, err = d.createTreatmentPlanScheduledMessage(tx, "dr_favorite_treatment_plan", common.ClaimerTypeFavoriteTreatmentPlanScheduledMessage, 0, sm)
					if err != nil {
						errs <- err
						return
					}
				}

				_, err = tx.Exec(`
					INSERT INTO dr_favorite_treatment_plan_membership
					(dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id)
					SELECT ?, doctor.id, ? FROM doctor`,
					ftpID, threadLocalPathwayID)
				if err != nil {
					errs <- err
					return
				}
			}
			done <- true
		}(pathwayID, ftps, done, errs)
	}

	completed := 0
	for completed < threads {
		select {
		case <-done:
			completed++
		case err := <-errs:
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

func (d *DataService) CreateFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error) {
	return d.createFTPMembership(d.db, ftpID, doctorID, pathwayID)
}

func (d *DataService) CreateFTPMemberships(memberships []*common.FTPMembership) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, v := range memberships {
		_, err := d.createFTPMembership(tx, v.DoctorFavoritePlanID, v.DoctorID, v.ClinicalPathwayID)
		if err != nil {
			tx.Rollback()
			return err
		}
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

func (d *DataService) DeleteFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error) {
	return d.deleteFTPMembership(d.db, ftpID, doctorID, pathwayID)
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

func (d *DataService) FTPMemberships(ftpID int64) ([]*common.FTPMembership, error) {
	return d.ftpMemberships(d.db, ftpID)
}

func (d *DataService) ftpMemberships(db db, ftpID int64) ([]*common.FTPMembership, error) {
	var memberships []*common.FTPMembership
	rows, err := db.Query(`
		SELECT id, dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id
		FROM dr_favorite_treatment_plan_membership
		WHERE dr_favorite_treatment_plan_id = ?`, ftpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		membership := &common.FTPMembership{}
		if err = rows.Scan(&membership.ID, &membership.DoctorFavoritePlanID, &membership.DoctorID, &membership.ClinicalPathwayID); err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	return memberships, rows.Err()
}

func (d *DataService) FTPMembershipsForDoctor(doctorID int64) ([]*common.FTPMembership, error) {
	return d.ftpMembershipsForDoctor(d.db, doctorID)
}

func (d *DataService) ftpMembershipsForDoctor(db db, doctorID int64) ([]*common.FTPMembership, error) {
	var memberships []*common.FTPMembership
	rows, err := db.Query(`
		SELECT id, dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id
		FROM dr_favorite_treatment_plan_membership
		WHERE doctor_id = ?`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		membership := &common.FTPMembership{}
		if err = rows.Scan(&membership.ID, &membership.DoctorFavoritePlanID, &membership.DoctorID, &membership.ClinicalPathwayID); err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	return memberships, rows.Err()
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
		common.TStatusCreated.String(), favoriteTreatmentPlanID, LanguageIDEnglish)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var treatments []*common.Treatment
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
