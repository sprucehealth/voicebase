package main

import (
	"carefront/api"
	"carefront/info_intake"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../info_intake/condition_intake.json")
	treatment := &info_intake.Treatment{}
	err := json.Unmarshal(fileContents, &treatment)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("mysql", "carefront:changethis@tcp(dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com:3306)/carefront_db")

	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	dataApi := &api.DataService{db}
	treatmentLayoutProcessor := &info_intake.TreatmentIntakeModelProcessor{dataApi}
	treatmentLayoutProcessor.FillInDetailsFromDatabase(treatment, 1)

	jsonData, err := json.MarshalIndent(treatment, "", " ")
	fmt.Println(string(jsonData))
}
