package main

import (
	"carefront/api"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	flagListenAddr = flag.String("listen", ":8080", "Address and port to listen on")
)

const (
	CertKeyLocation    string = "CERT_KEY"
	PrivateKeyLocation string = "PRIVATE_KEY"
)

func main() {
	flag.Parse()

	dbUsername := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUsername, dbPassword, dbHost, dbName)

	// this gives us a connection pool to the sql instance
	// without executing any statements against the sql database
	// or checking the network connection and authentication to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err.Error())
	}

	// test the connection to the database by running a ping against it
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	authApi := &api.AuthService{db}
	dataApi := &api.DataService{db}
	mux := &AuthServeMux{*http.NewServeMux(), authApi}

	authHandler := &AuthenticationHandler{authApi}
	pingHandler := PingHandler(0)
	photoHandler := &PhotoUploadHandler{api.PhotoService(0), dataApi}
	getSignedUrlsHandler := &GetSignedUrlsHandler{api.PhotoService(0)}

	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/signup", authHandler)
	mux.Handle("/v1/logout", authHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/upload", photoHandler)
	mux.Handle("/v1/imagesForCase/", getSignedUrlsHandler)

	s := &http.Server{
		Addr:           *flagListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServeTLS(os.Getenv(CertKeyLocation), os.Getenv(PrivateKeyLocation)))
}
