package main

import (
	"carefront/api"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func main() {

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
	bucket, key, region, err := dataApi.GetCurrentActiveLayoutInfoForTreatment(1)
	if err != nil {
		panic(err)
	}
	fmt.Println(bucket, key, region)
}
