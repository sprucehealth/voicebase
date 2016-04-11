package main

import (
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"text/template"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/schedmsg"
)

var (
	dbHost      = flag.String("db_host", "", "mysql database host")
	dbPort      = flag.Int("dp_port", 3306, "mysql database port")
	dbName      = flag.String("db_name", "", "mysql database name")
	dbUsername  = flag.String("db_username", "", "mysql database username")
	dbPassword  = flag.String("db_password", "", "mysql database password")
	oldDoctorID = flag.Int64("old_doctor_id", 0, "doctorID of the old doctor")
	newDoctorID = flag.Int64("new_doctor_id", 0, "doctorID of the new doctor")
	state       = flag.String("state", "", "state from which cases are to be migrated. Values are `all`, `CA`, `VA`, etc.")
	ccID        = flag.Int64("cc_id", 24, "care coordinator id")
)

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

func validate() error {
	if *dbHost == "" {
		return errors.New("DB Host required")
	}
	if *dbName == "" {
		return errors.New("DB Name required")
	}
	if *dbUsername == "" {
		return errors.New("DB Username required")
	}
	if *dbPassword == "" {
		return errors.New("DB Password required")
	}
	if *oldDoctorID == 0 {
		return errors.New("Old DoctorID required")
	}
	if *newDoctorID == 0 {
		return errors.New("New DoctorID required")
	}
	if *state == "" {
		return errors.New("State required")
	}
	if *ccID == 0 {
		return errors.New("CC ID required")
	}
	return nil
}

// Purpose of this script is to send a case message to the provided list of caseIDs
func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	if err := validate(); err != nil {
		golog.Fatalf(err.Error())
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     *dbHost,
		Port:     *dbPort,
		Name:     *dbName,
		User:     *dbUsername,
		Password: *dbPassword,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	cfgStore, err := cfg.NewLocalStore(config.CfgDefs())
	if err != nil {
		golog.Fatalf("Failed to initialize local cfg store: %s", err)
	}

	dataAPI, err := api.NewDataService(db, cfgStore, metrics.NewRegistry())
	if err != nil {
		golog.Fatalf("Unable to initialize data service layer: %s", err)
	}

	// old doctor
	oldDoctor, err := dataAPI.Doctor(*oldDoctorID, true)
	if err != nil {
		golog.Fatalf("Could not get old doctor: %s", err.Error())
	}

	// new doctor
	newDoctor, err := dataAPI.Doctor(*newDoctorID, true)
	if err != nil {
		golog.Fatalf("Could not get new doctor: %s", err.Error())
	}

	// care coordinator
	cc, err := dataAPI.Doctor(*ccID, true)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	personID, err := dataAPI.GetPersonIDByRole(api.RoleCC, *ccID)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	patientIDs, caseIDs, err := getCaseIDs(db)
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

	if err := migrateDoctorForCases(patientIDs, caseIDs, db); err != nil {
		golog.Fatalf(err.Error())
	}

	golog.Infof("Successfully migrate doctor for following %d cases from %s to %s: %v", len(caseIDs), oldDoctor.ShortDisplayName, newDoctor.ShortDisplayName, caseIDs)

	for _, caseID := range caseIDs {

		pc, err := dataAPI.GetPatientCaseFromID(caseID)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		patient, err := dataAPI.Patient(pc.PatientID, true)
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

		if _, err := dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
			golog.Fatalf("Unable to create scheduled message for caseID %d: %s", caseID, err)
		}

		golog.Infof("Successfully scheduled message for %d", caseID)
	}
}

func getCaseIDs(db *sql.DB) (patientIDs []int64, caseIDs []int64, err error) {
	// first, get a list of caseIDs to migrate
	var filterStates string
	if *state != "all" {
		filterStates = fmt.Sprintf("AND patient_location.state = '%s'", *state)
	}

	rows, err := db.Query(`SELECT patient_case.patient_id, patient_case_id 
FROM patient_case_care_provider_assignment 
INNER JOIN patient_case ON patient_case.id = patient_case_id 
INNER JOIN patient_location ON patient_location.patient_id = patient_case.patient_id 
WHERE provider_id = ? AND patient_case.status != 'INACTIVE' `+filterStates, *oldDoctorID)
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

func migrateDoctorForCases(patientIDs, caseIDs []int64, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i, caseID := range caseIDs {

		_, err := tx.Exec(`
		UPDATE IGNORE patient_care_provider_assignment
		SET provider_id = ?
		WHERE patient_id = ?
		AND provider_id = ?`, *newDoctorID, patientIDs[i], *oldDoctorID)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`
		UPDATE IGNORE patient_case_care_provider_assignment
		SET provider_id = ?
		WHERE patient_case_id = ?
		AND provider_id = ?`, *newDoctorID, caseID, *oldDoctorID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
