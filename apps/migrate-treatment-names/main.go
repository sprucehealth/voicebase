package main

import (
	"database/sql"
	"flag"
	"fmt"
	"strings"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	dbConfig       config.DB
	doseSpotConfig config.DosespotConfig
	verbose        bool
)

type drug struct {
	name     string
	strength string
}

func (d drug) String() string {
	return fmt.Sprintf("%s %s", d.name, d.strength)
}

func init() {
	flag.StringVar(&dbConfig.Name, "db.name", "", "database name")
	flag.StringVar(&dbConfig.User, "db.user", "", "database username")
	flag.StringVar(&dbConfig.Password, "db.pass", "", "database password")
	flag.StringVar(&dbConfig.Host, "db.host", "", "database host")
	flag.IntVar(&dbConfig.Port, "db.port", 0, "database port")
	flag.Int64Var(&doseSpotConfig.ClinicID, "ds.climit_id", 0, "DoseSpot clinic ID")
	flag.Int64Var(&doseSpotConfig.ProxyID, "ds.proxy_id", 0, "DoseSpot proxy ID")
	flag.StringVar(&doseSpotConfig.ClinicKey, "ds.clinic_key", "", "DoseSpot clinit key")
	flag.StringVar(&doseSpotConfig.SOAPEndpoint, "ds.soap_endpoint", "", "DoseSpot SOAP endpoint")
	flag.StringVar(&doseSpotConfig.APIEndpoint, "ds.api_endpoint", "", "DoseSpot API endpoint")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
}

func main() {
	flag.Parse()
	if verbose {
		golog.Default().SetLevel(golog.DEBUG)
	} else {
		golog.Default().SetLevel(golog.INFO)
	}

	db, err := dbConfig.ConnectMySQL(nil)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	doseSpotService := erx.NewDoseSpotService(
		doseSpotConfig.ClinicID, doseSpotConfig.ProxyID, doseSpotConfig.ClinicKey,
		doseSpotConfig.SOAPEndpoint, doseSpotConfig.APIEndpoint,
		metrics.NewRegistry())

	// Fetch unique medication to minimize queries against DoseSpot
	drugs, err := uniqueDrugs(db)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	for _, drug := range drugs {
		genericName, err := findGenericName(doseSpotService, drug)
		if err != nil {
			golog.Errorf("Failed to get generic name for %+v: %s", drug, err.Error())
			continue
		}
		golog.Infof("%s -> %s", drug.String(), genericName)
		// To avoid a race, first try to insert and only if that fails (exists) then do the select.
		// This isn't very efficient, but since this is just a migration it shouldn't matter much.
		res, err := db.Exec(`INSERT IGNORE INTO drug_name (name) VALUES (?)`, genericName)
		if err != nil {
			golog.Errorf("Failed to create drug name '%s': %s", genericName, err.Error())
			continue
		}
		id, err := res.LastInsertId()
		if err != nil {
			golog.Errorf("Failed to get last insert ID: %s", err.Error())
			continue
		}
		if id == 0 {
			// Didn't insert
			if err := db.QueryRow(`SELECT id FROM drug_name WHERE name = ?`, genericName).Scan(&id); err != nil {
				golog.Errorf("Failed to get id: %s", err.Error())
				continue
			}
		}
		if _, err := db.Exec(`
			UPDATE dr_treatment_template
			SET generic_drug_name_id = ?
			WHERE drug_internal_name = ? AND dosage_strength = ?`,
			id, drug.name, drug.strength,
		); err != nil {
			golog.Fatalf("Failed to update dr_treatment_template: %s", err.Error())
		}
		if _, err := db.Exec(`
			UPDATE treatment
			SET generic_drug_name_id = ?
			WHERE drug_internal_name = ? AND dosage_strength = ?`,
			id, drug.name, drug.strength,
		); err != nil {
			golog.Fatalf("Failed to update treatment: %s", err.Error())
		}
	}
}

func findGenericName(doseSpotService erx.ERxAPI, drug drug) (string, error) {
	parsedName := drug.name
	// Remove the (route - form) from the name
	if i := strings.IndexByte(parsedName, '('); i >= 0 {
		parsedName = parsedName[:i-1]
	}
	golog.Debugf(parsedName)
	names, err := doseSpotService.GetDrugNamesForDoctor(0, parsedName)
	if err != nil {
		return "", err
	}
	golog.Debugf("\tNames: %+v", names)
	if len(names) == 0 {
		return "", fmt.Errorf("no names found")
	}

	// First try an exact name match. Fall back to first in the list.
	name := names[0]
	for _, n := range names {
		if n == parsedName {
			name = n
			break
		}
	}

	strengths, err := doseSpotService.SearchForMedicationStrength(0, name)
	if err != nil {
		return "", err
	}
	golog.Debugf("\tStrengths: %+v", strengths)
	if len(strengths) == 0 {
		return "", fmt.Errorf("no strengths found")
	}

	// First try an exact strength match. Fall back to first in the list.
	strength := strengths[0]
	for _, s := range strengths {
		if s == drug.strength {
			strength = s
			break
		}
	}

	med, err := doseSpotService.SelectMedication(0, name, strength)
	if err != nil {
		return "", err
	}
	golog.Debugf("\tMedication: %+v", med)
	genericName, err := erx.ParseGenericName(med)
	if err != nil {
		return "", err
	}
	golog.Debugf("\tGeneric Name: %s", genericName)
	if genericName == "" {
		return "", fmt.Errorf("empty generic name")
	}
	return genericName, nil
}

func uniqueDrugs(db *sql.DB) ([]drug, error) {
	rows, err := db.Query(`
		SELECT DISTINCT drug_internal_name, dosage_strength
		FROM
		(
		SELECT DISTINCT drug_internal_name, dosage_strength FROM dr_treatment_template WHERE generic_drug_name_id IS NULL
		UNION
		SELECT DISTINCT drug_internal_name, dosage_strength FROM treatment WHERE generic_drug_name_id IS NULL
		) a
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drugs []drug
	for rows.Next() {
		var d drug
		if err := rows.Scan(&d.name, &d.strength); err != nil {
			return nil, err
		}
		drugs = append(drugs, d)
	}
	return drugs, rows.Err()
}
