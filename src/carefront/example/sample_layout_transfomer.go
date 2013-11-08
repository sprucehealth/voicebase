package main

import (
	"carefront/api"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../layout_transformer/condition_intake.json")
	treatment := &api.Treatment{}
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
	treatmentLayoutProcessor := &api.TreatmentLayoutProcessor{dataApi}
	treatmentLayoutProcessor.TransformIntakeIntoClientLayout(treatment, 1)

	jsonData, err := json.MarshalIndent(treatment, "", " ")
	fmt.Println(string(jsonData))
}
