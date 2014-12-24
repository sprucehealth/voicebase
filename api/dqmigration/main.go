package main

import (
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	dbName     = flag.String("db_name", "", "database name")
	dbUserName = flag.String("db_username", "", "database username")
	dbPassword = flag.String("db_password", "", "database password")
	dbHost     = flag.String("db_host", "", "database host")
	dbPort     = flag.Int("db_port", 3306, "db port")
	apiDomain  = flag.String("api_domain", "", "api domain")
)

func main() {

	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		*dbUserName, *dbPassword, *dbHost, *dbPort, *dbName))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		golog.Fatalf(err.Error())
	}
	defer db.Close()

	dataAPI, err := api.NewDataService(db, *apiDomain)
	if err != nil {
		golog.Fatalf("Unable to initialize data service layer: %s", err)
	}

	numItems, err := dataAPI.GetTotalNumberOfDoctorQueueItemsWithoutDescription()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	golog.Infof("Total number of items to process: %d", numItems)

	batchSize := 50
	for remainingItems := numItems; remainingItems > 0; {

		if remainingItems < batchSize {
			batchSize = remainingItems
		}
		remainingItems -= batchSize

		queueItems, err := dataAPI.GetNDQItemsWithoutDescription(batchSize)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		for _, queueItem := range queueItems {
			queueItem.Description, queueItem.ShortDescription, err = getLongAndShortDescription(dataAPI, queueItem)
			if err != nil {
				golog.Fatalf(err.Error())
			}

			queueItem.ActionURL, err = getActionURL(dataAPI, queueItem)
			if err != nil {
				golog.Fatalf(err.Error())
			}

			queueItem.PatientID, err = getPatientID(dataAPI, queueItem)
			if err != nil {
				golog.Fatalf(err.Error())
			}
		}

		// update the description and action url for this items
		if err := dataAPI.UpdateDoctorQueueItems(queueItems); err != nil {
			golog.Fatalf(err.Error())
		}

		golog.Infof("Successfully added description and action url for %d events", batchSize)
	}

}
