// The purpose of this script is to mass migrate patient cases in a provided list of states
// from one doctor to another, and inform patients via a scheduled message of the change in their doctor.
package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/schedmsg"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type migratePatientsCmd struct {
	db      *sql.DB
	dataAPI api.DataAPI
}

func newMigratePatientsCmd(cnf *conf) (command, error) {
	db, err := cnf.db()
	if err != nil {
		return nil, err
	}
	return &migratePatientsCmd{
		db:      db,
		dataAPI: cnf.DataAPI,
	}, nil
}

type context struct {
	OldDoctorName    string
	NewDoctorName    string
	PatientFirstName string
	CCName           string
}

const msg = `Hi {{.PatientFirstName}},

I'm writing to let you know that {{.OldDoctorName}} is no longer practicing on Spruce. To ensure that you can continue to receive care, {{.NewDoctorName}} — another dermatologist on Spruce — will be the physician on your care team going forward.

{{.NewDoctorName}} has a reputation for providing excellent care for Spruce patients. I'm sure you'll enjoy working together!

Please let me know if you have any questions.
Warmly,
{{.CCName}}`

func (c *migratePatientsCmd) run(args []string) error {
	fs := flag.NewFlagSet("migratepatients", flag.ExitOnError)
	oldDoctorID := fs.Int64("old_doctor_id", 0, "doctorID of the old doctor")
	newDoctorID := fs.Int64("new_doctor_id", 0, "doctorID of the new doctor")
	state := fs.String("state", "", "state from which cases are to be migrated. Values are `all`, `CA`, `VA`, etc.")
	ccID := fs.Int64("cc_id", 24, "care coordinator id")
	scheduleMessage := fs.Bool("schedule_message", false, "flag to indicate whether or not to schedule a message to patient")

	scn := bufio.NewScanner(os.Stdin)

	if *oldDoctorID == 0 {
		var err error
		*oldDoctorID, err = strconv.ParseInt(prompt(scn, "Old Doctor ID: "), 10, 64)
		if err != nil {
			return errors.Errorf("Invalid Old Doctor ID")
		}
	}

	if *newDoctorID == 0 {
		var err error
		*newDoctorID, err = strconv.ParseInt(prompt(scn, "neW Doctor ID: "), 10, 64)
		if err != nil {
			return errors.Errorf("Invalid New Doctor ID")
		}
	}

	if *ccID == 0 {
		var err error
		*ccID, err = strconv.ParseInt(prompt(scn, "Care Coordinator ID: "), 10, 64)
		if err != nil {
			return errors.Errorf("Invalid New Doctor ID")
		}
	}

	if *state == "" {
		*state = prompt(scn, "State: ")
	}
	if *state == "" {
		return errors.New("State required")
	}

	// old doctor
	oldDoctor, err := c.dataAPI.Doctor(*oldDoctorID, true)
	if err != nil {
		golog.Fatalf("Could not get old doctor: %s", err.Error())
	}

	// new doctor
	newDoctor, err := c.dataAPI.Doctor(*newDoctorID, true)
	if err != nil {
		golog.Fatalf("Could not get new doctor: %s", err.Error())
	}

	// care coordinator
	cc, err := c.dataAPI.Doctor(*ccID, true)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	personID, err := c.dataAPI.GetPersonIDByRole(api.RoleCC, *ccID)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	patientIDs, caseIDs, err := getCaseIDs(c.db, *state, *oldDoctorID)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	if len(caseIDs) == 0 {
		golog.Fatalf("No cases to migrate from %s to %s in state: %s", oldDoctor.ShortDisplayName, newDoctor.ShortDisplayName, *state)
	}

	tmpl, err := template.New("").Parse(msg)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	if err := migrateDoctorForCases(patientIDs, caseIDs, *oldDoctorID, *newDoctorID, c.db); err != nil {
		golog.Fatalf(err.Error())
	}

	golog.Infof("Successfully migrate doctor for following %d cases from %s to %s: %v", len(caseIDs), oldDoctor.ShortDisplayName, newDoctor.ShortDisplayName, caseIDs)

	if *scheduleMessage {
		for _, caseID := range caseIDs {

			pc, err := c.dataAPI.GetPatientCaseFromID(caseID)
			if err != nil {
				golog.Fatalf(err.Error())
			}

			patient, err := c.dataAPI.Patient(pc.PatientID, true)
			if err != nil {
				golog.Fatalf(err.Error())
			}

			ctxt := &context{
				PatientFirstName: patient.FirstName,
				CCName:           cc.FirstName,
				OldDoctorName:    oldDoctor.LongDisplayName,
				NewDoctorName:    newDoctor.LongDisplayName,
			}

			var b bytes.Buffer
			if err := tmpl.Execute(&b, ctxt); err != nil {
				golog.Fatalf(err.Error())
			}

			scheduledMessage := &common.ScheduledMessage{
				Event:     "doctor_change",
				PatientID: patient.ID,
				Message: &schedmsg.CaseMessage{
					Message:        b.String(),
					PatientCaseID:  caseID,
					SenderPersonID: personID,
					SenderRole:     api.RoleCC,
					ProviderID:     cc.ID.Int64(),
				},
				Scheduled: time.Now().Add(2 * time.Hour),
				Status:    common.SMScheduled,
			}

			if _, err := c.dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
				golog.Fatalf("Unable to create scheduled message for caseID %d: %s", caseID, err)
			}

			golog.Infof("Successfully scheduled message for %d", caseID)
		}
	}

	return nil
}

func getCaseIDs(db *sql.DB, state string, oldDoctorID int64) (patientIDs []int64, caseIDs []int64, err error) {
	// first, get a list of caseIDs to migrate
	var filterStates string
	if state != "all" {
		filterStates = fmt.Sprintf("AND patient_location.state = '%s'", state)
	}

	rows, err := db.Query(`SELECT patient_case.patient_id, patient_case_id 
FROM patient_case_care_provider_assignment 
INNER JOIN patient_case ON patient_case.id = patient_case_id 
INNER JOIN patient_location ON patient_location.patient_id = patient_case.patient_id 
WHERE provider_id = ? AND patient_case.status != 'INACTIVE' `+filterStates, oldDoctorID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var caseID, patientID int64
		if err := rows.Scan(&patientID, &caseID); err != nil {
			return nil, nil, err
		}
		patientIDs = append(patientIDs, patientID)
		caseIDs = append(caseIDs, caseID)
	}
	return patientIDs, caseIDs, rows.Err()
}

func migrateDoctorForCases(patientIDs, caseIDs []int64, oldDoctorID, newDoctorID int64, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i, caseID := range caseIDs {

		_, err := tx.Exec(`
		UPDATE IGNORE patient_care_provider_assignment
		SET provider_id = ?
		WHERE patient_id = ?
		AND provider_id = ?`, newDoctorID, patientIDs[i], oldDoctorID)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`
		UPDATE IGNORE patient_case_care_provider_assignment
		SET provider_id = ?
		WHERE patient_case_id = ?
		AND provider_id = ?`, newDoctorID, caseID, oldDoctorID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
