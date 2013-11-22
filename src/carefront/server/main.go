package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/config"
	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultMaxInMemoryPhotoMB = 2
)

type DBConfig struct {
	User     string `long:"db_user" description:"Username for accessing database"`
	Password string `long:"db_password" description:"Password for accessing database"`
	Host     string `long:"db_host" description:"Database host"`
	Name     string `long:"db_name" description:"Database name"`
}

type Config struct {
	config.BaseConfig
	ListenAddr            string   `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	CertLocation          string   `long:"cert_key" description:"Path of SSL certificate"`
	KeyLocation           string   `long:"private_key" description:"Path of SSL private key"`
	S3CaseBucket          string   `long:"case_bucket" description:"S3 Bucket name for case information"`
	DB                    DBConfig `group:"Database" toml:"database"`
	MaxInMemoryForPhotoMB int64    `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
}

var DefaultConfig = Config{
	ListenAddr:            ":8080",
	S3CaseBucket:          "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
}

func main() {
	conf := DefaultConfig
	_, err := config.ParseFlagsAndConfig(&conf, nil)
	if err != nil {
		log.Fatal(err)
	}

	if conf.DB.User == "" || conf.DB.Password == "" || conf.DB.Host == "" || conf.DB.Name == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Name)

	// this gives us a connection pool to the sql instance
	// without executing any statements against the sql database
	// or checking the network connection and authentication to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}

	authApi := &api.AuthService{db}
	dataApi := &api.DataService{db}
	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	photoAnswerCloudStorageApi := api.NewCloudStorageService(awsAuth)
	authHandler := &apiservice.AuthenticationHandler{authApi}
	signupPatientHandler := &apiservice.SignupPatientHandler{dataApi, authApi}
	patientVisitHandler := apiservice.NewPatientVisitHandler(dataApi, authApi, cloudStorageApi, photoAnswerCloudStorageApi)
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(dataApi)
	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(dataApi, photoAnswerCloudStorageApi, conf.S3CaseBucket, conf.AWSRegion, conf.MaxInMemoryForPhotoMB*1024*1024)
	pingHandler := apiservice.PingHandler(0)
	generateModelIntakeHandler := &apiservice.GenerateClientIntakeModelHandler{
		DataApi:         dataApi,
		CloudStorageApi: cloudStorageApi,
		AWSRegion:       conf.AWSRegion,
	}

	mux := &apiservice.AuthServeMux{*http.NewServeMux(), authApi}

	mux.Handle("/v1/patient", signupPatientHandler)
	mux.Handle("/v1/visit", patientVisitHandler)
	mux.Handle("/v1/answer", answerIntakeHandler)
	mux.Handle("/v1/answer/photo", photoAnswerIntakeHandler)
	mux.Handle("/v1/client_model", generateModelIntakeHandler)

	mux.Handle("/v1/signup", authHandler)
	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/logout", authHandler)
	mux.Handle("/v1/ping", pingHandler)

	s := &http.Server{
		Addr:           conf.ListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if conf.CertLocation == "" && conf.KeyLocation == "" {
		log.Fatal(s.ListenAndServe())
	} else {
		log.Fatal(s.ListenAndServeTLS(conf.CertLocation, conf.KeyLocation))
	}
}
