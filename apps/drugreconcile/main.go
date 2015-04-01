package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

var clinicianID = flag.Int64("clinician_id", 0, "clinician id")
var clinicID = flag.Int64("clinic_id", 0, "clinic id")
var clinicKey = flag.String("clinic_key", "", "clinic key")
var soapEndpoint = flag.String("soap_endpoint", "", "soap endpoint")
var apiEndpoint = flag.String("api_endpoint", "", "endpoint")
var listCSV = flag.String("csv", "list.csv", "csv")
var action = flag.String("action", "identify", "action")
var tableName = flag.String("table_name", "", "table name")

func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	csvFile, err := os.Open(*listCSV)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	switch *action {
	case "identify":
		identifyDrugs(reader)
	case "sql":
		createUpdateStatements(reader)
	}

}

// identifyDrugs outputs which drugs were found/missing in the third party drug database
func identifyDrugs(reader *csv.Reader) {
	dosespotCLI := erx.NewDoseSpotService(*clinicID, *clinicianID, *clinicKey, *soapEndpoint, *apiEndpoint, nil)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			golog.Fatalf(err.Error())
		}

		drugName := row[0]
		dosageStrength := row[1]

		res, err := dosespotCLI.SelectMedication(*clinicianID, drugName, dosageStrength)
		if err != nil {
			golog.Fatalf(err.Error())
		}

		status := "FOUND"
		if res == nil {
			status = "MISSING"
		}

		fmt.Printf("'%s\t'%s\t'%s\n", drugName, dosageStrength, status)
	}
}

// createUpdateStatements creates sql statements that can then be applied
// to any database
func createUpdateStatements(reader *csv.Reader) {
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			golog.Fatalf(err.Error())
		}

		if row[2] == "MISSING" {
			fmt.Printf("UPDATE %s\nSET drug_internal_name = '%s', dosage_strength = '%s'\nWHERE drug_internal_name = '%s'\nAND dosage_strength = '%s';\n\n", *tableName, row[3], row[4], row[0], row[1])
		}
	}

}
