package main

import (
	"carefront/api"
	"carefront/layout_transformer"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../layout_transformer/condition_intake.json")
	treatmentRes := &layout_transformer.Treatment{}
	err := json.Unmarshal(fileContents, &treatmentRes)
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
	clientLayoutProcessor := &layout_transformer.ClientLayoutProcessor{dataApi}
	err = clientLayoutProcessor.TransformIntakeIntoClientLayout(treatmentRes)
	if err != nil {
		panic(err)
	}

	marshalledBytes, err := json.MarshalIndent(treatmentRes, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(marshalledBytes))
}
