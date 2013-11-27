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
	healthCondition := &info_intake.HealthCondition{}
	err := json.Unmarshal(fileContents, &healthCondition)
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
	healthConditionLayoutProcessor := &info_intake.HealthConditionIntakeModelProcessor{dataApi}
	err = healthConditionLayoutProcessor.FillInDetailsFromDatabase(healthCondition, 1)
	if err != nil {
		panic(err.Error())
	}

	jsonData, err := json.MarshalIndent(healthCondition, "", " ")
	fmt.Println(string(jsonData))
}
