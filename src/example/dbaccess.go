package main

import (
	"carefront/api"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func main() {

	db, err := sql.Open("mysql", "carefront:changethis@tcp(dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com:3306)/carefront_db?parseTime=true")

	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	dataApi := &api.DataService{DB: db}
	patientVisit, err := dataApi.GetPatientVisitFromId(85)
	if err != nil {
		panic(err)
	}
	fmt.Println(patientVisit)
}
