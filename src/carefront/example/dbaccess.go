package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func main() {

	db, err := sql.Open("mysql", "ejabberd:ejabberd@tcp(ejabberd-db-dev.c83wlsbcftxz.us-west-1.rds.amazonaws.com:3306)/hello")

	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	defer db.Close()
}
