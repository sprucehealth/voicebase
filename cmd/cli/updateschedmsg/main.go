package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
)

var dbHost = flag.String("db_host", "", "mysql database host")
var dbPort = flag.Int("dp_port", 3306, "mysql database port")
var dbName = flag.String("db_name", "", "mysql database name")
var dbUsername = flag.String("db_username", "", "mysql database username")
var dbPassword = flag.String("db_password", "", "mysql database password")
var listCSV = flag.String("csv", "list.csv", "csv")

// purpose of this script is to replace scheduled messages for a given list of
// treamtent plan scheduled messages with a generic check in style scheduled message
func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	// connect to the database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=Local&interpolateParams=true",
		*dbUsername, *dbPassword, *dbHost, *dbPort, *dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
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

	csvFile, err := os.Open(*listCSV)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			golog.Fatalf(err.Error())
		}

		tpSchedMsgID, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		if err := updateScheduledMessage(db, dataAPI, tpSchedMsgID); err != nil {
			golog.Fatalf("Unable to update scheduled message for id %d. Error: %s", tpSchedMsgID, err.Error())
		}

		golog.Infof("Updated message for tpSchedMsgID: %d", tpSchedMsgID)
	}
}

func updateScheduledMessage(db *sql.DB, dataAPI api.DataAPI, tpSchedMsgID int64) error {

	var tpID, smID int64
	var patientID common.PatientID
	var scheduledDays int
	if err := db.QueryRow(`
			SELECT treatment_plan_id, tp.patient_id, scheduled_days, scheduled_message_id
			FROM treatment_plan_scheduled_message tpsm
			INNER JOIN treatment_plan tp ON tp.id = tpsm.treatment_plan_id
			WHERE tpsm.id = ?`, tpSchedMsgID).Scan(&tpID, &patientID, &scheduledDays, &smID); err != nil {
		return err
	}

	patient, err := dataAPI.Patient(patientID, true)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Unclaim all attached media
	_, err = tx.Exec(`
		DELETE FROM media_claim
		WHERE claimer_type = ? AND claimer_id = ?`,
		common.ClaimerTypeTreatmentPlanScheduledMessage, tpSchedMsgID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// delete scheduled message attachments associated with the given
	// message
	_, err = tx.Exec(`
		DELETE FROM treatment_plan_scheduled_message_attachment
		WHERE treatment_plan_scheduled_message_id = ?`, tpSchedMsgID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the message while leaving other properties re: the treatment plan
	// scheduled message untouched.
	msg := fmt.Sprintf("Hi %s -- I wanted to check in on your progress and whether your treatment plan is working so far. How are things going?", strings.TrimSpace(patient.FirstName))
	_, err = tx.Exec(`
		UPDATE treatment_plan_scheduled_message
		SET message = ?
		WHERE id = ?`, msg, tpSchedMsgID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
