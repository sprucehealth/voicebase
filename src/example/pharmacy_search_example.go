package main

import (
	"carefront/api"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", "carefront", "changethis", "carefront-content-db.ckwporuc939i.us-east-1.rds.amazonaws.com", "pharmacy_db")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		panic(err)
	}

	pharmacySearchService := &api.PharmacySearchService{PharmacyDB: db}
	_, err = pharmacySearchService.GetPharmaciesAroundSearchLocation(37.781575, -122.432654, 10.0, 10)

	if err != nil {
		panic(err)
	}
}
