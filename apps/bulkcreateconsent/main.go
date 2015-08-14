package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient"
)

var dbHost = flag.String("db_host", "", "mysql database host")
var dbPort = flag.Int("dp_port", 3306, "mysql database port")
var dbName = flag.String("db_name", "", "mysql database name")
var dbUsername = flag.String("db_username", "", "mysql database username")
var dbPassword = flag.String("db_password", "", "mysql database password")
var webDomain = flag.String("web_domain", "", "web domain")
var listCSV = flag.String("csv", "", "csv")

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

		childPatientID, err := common.ParsePatientID(strings.TrimSpace(row[0]))
		if err != nil {
			golog.Fatalf(err.Error())
		}

		url, err := patient.ParentalConsentURL(dataAPI, *webDomain, childPatientID)
		if err != nil {
			golog.Fatalf(err.Error())
		}
		fmt.Println(url)
	}

}
