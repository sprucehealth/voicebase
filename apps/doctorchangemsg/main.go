package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/schedmsg"
)

var dbHost = flag.String("db_host", "", "mysql database host")
var dbPort = flag.Int("dp_port", 3306, "mysql database port")
var dbName = flag.String("db_name", "", "mysql database name")
var dbUsername = flag.String("db_username", "", "mysql database username")
var dbPassword = flag.String("db_password", "", "mysql database password")
var apiDomain = flag.String("api_domain", "", "api domain")
var listCSV = flag.String("csv", "list.csv", "csv")
var ccID = flag.Int64("cc_id", 24, "care coordinator id")
var msgFile = flag.String("msg_file", "", "file that contains message to send to patient")

type context struct {
	PatientFirstName string
	CCName           string
}

// Purpose of this script is to send a case message to the provided list of caseIDs
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

	// care coordinator
	cc, err := dataAPI.Doctor(*ccID, true)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	personID, err := dataAPI.GetPersonIDByRole(api.RoleCC, *ccID)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	msgData, err := ioutil.ReadFile(*msgFile)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	msg := string(msgData)

	tmpl, err := template.New("").Parse(msg)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// read in the list of caseIDs, and create a scheduled message for each.
	csvFile, err := os.Open(*listCSV)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	var b bytes.Buffer

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			golog.Fatalf(err.Error())
		}

		caseID, err := strconv.ParseInt(strings.TrimSpace(row[0]), 10, 64)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		pc, err := dataAPI.GetPatientCaseFromID(caseID)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		patient, err := dataAPI.Patient(pc.PatientID, true)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		b.Reset()
		ctxt := &context{
			PatientFirstName: patient.FirstName,
			CCName:           cc.ShortDisplayName,
		}
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
