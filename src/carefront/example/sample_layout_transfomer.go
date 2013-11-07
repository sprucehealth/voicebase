package main

import (
	"carefront/api"
	"carefront/layout_transformer"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../layout_transformer/condition_intake.json")
	treatment := &layout_transformer.Treatment{}
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
	cloudObjectService := api.NewService("AKIAINP33PBIN5GW4GKQ", "rbqPao4jDqTBTXBHk4BRnzWmYsfvSslg9mYhG45w")
	layoutService := &api.LayoutService{dataApi, cloudObjectService}

	layoutService.VerifyAndUploadIncomingLayout(fileContents, treatment.TreatmentTag)
}
