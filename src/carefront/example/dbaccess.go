package main

import (
	"carefront/api"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
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
	authApi := &api.AuthService{db}
	_, accountId, err := authApi.Signup("k1k1@gmail.com", "12345")
	if err != nil {
		panic(err)
	}
	patientId, err := dataApi.RegisterPatient(accountId, "Kunal", "Jham", "male", "94115", time.Date(1987, 11, 8, 0, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Println(patientId)
}
