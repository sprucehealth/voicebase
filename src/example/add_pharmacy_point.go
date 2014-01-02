package main

import (
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

	rows, err := db.Query(`select id,loc_LAT_centroid, loc_LONG_centroid from dump_pharmacies where id > 5913`)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var longitude, latitude string
		var id int64
		rows.Scan(&id, &latitude, &longitude)
		_, err = db.Exec(`update dump_pharmacies set loc_pt = POINT(?,?) where id = ?`, longitude, latitude, id)
		if err != nil {
			panic(err.Error())
		}
	}
}
