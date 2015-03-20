package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/golog"
)

// command line options
var dbHost = flag.String("db_host", "", "mysql database host")
var dbPort = flag.Int("dp_port", 3306, "mysql database port")
var dbName = flag.String("db_name", "", "mysql database name")
var dbUsername = flag.String("db_username", "", "mysql database username")
var dbPassword = flag.String("db_password", "", "mysql database password")
var apiDomain = flag.String("api_domain", "", "api domain")

// Purpose of this script is to migrate the imageURLs in doctor promotions
// to the new api endpoint for retrieving doctor images.
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

	rows, err := db.Query(`
		SELECT promotion_code_id, referral_data FROM referral_program
		WHERE account_id in (select account_id from doctor)`)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	tx, err := db.Begin()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	err = func() error {
		for rows.Next() {
			var promoCodeID int64
			var data sql.RawBytes
			if err := rows.Scan(&promoCodeID, &data); err != nil {
				tx.Rollback()
				return err
			}

			var jsonMap map[string]interface{}
			if err := json.Unmarshal(data, &jsonMap); err != nil {
				tx.Rollback()
				return err
			}

			drPromo := jsonMap["route_doctor_promotion"].(map[string]interface{})
			currentImageURL := drPromo["image_url"].(string)
			doctorID := drPromo["doctor_id"].(float64)
			newImageURL := app_url.ThumbnailURL(*apiDomain, api.DOCTOR_ROLE, int64(doctorID))

			golog.Infof("Updating %s -> %s", currentImageURL, newImageURL)
			drPromo["image_url"] = newImageURL
			jsonMap["route_doctor_promotion"] = drPromo

			jsonData, err := json.Marshal(jsonMap)
			if err != nil {
				tx.Rollback()
				return err
			}

			if _, err := tx.Exec(`
				UPDATE referral_program
				SET referral_data = ?
				WHERE promotion_code_id = ?`, jsonData, promoCodeID); err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := rows.Err(); err != nil {
			tx.Rollback()
			return err
		}

		return nil
	}()

	if err != nil {
		golog.Fatalf(err.Error())
	}

	if err := tx.Commit(); err != nil {
		golog.Fatalf(err.Error())
	}
}
