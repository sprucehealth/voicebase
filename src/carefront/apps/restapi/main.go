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
	"carefront/common/config"
	"carefront/libs/svcclient"
	"carefront/libs/svcreg"
	thriftapi "carefront/thrift/api"
	_ "github.com/go-sql-driver/mysql"
	"github.com/samuel/go-metrics/metrics"
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
	*config.BaseConfig
	ListenAddr               string   `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	CertLocation             string   `long:"cert_key" description:"Path of SSL certificate"`
	KeyLocation              string   `long:"private_key" description:"Path of SSL private key"`
	DB                       DBConfig `group:"Database" toml:"database"`
	MaxInMemoryForPhotoMB    int64    `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
	CertKeyLocation          string   `long:"cert_key" description:"Path of SSL certificate"`
	PrivateKeyLocation       string   `long:"private_key" description:"Path of SSL private key"`
	CaseBucket               string   `long:"case_bucket" description:"S3 Bucket name for case information"`
	PatientLayoutBucket      string   `long:"client_layout_bucket" description:"S3 Bucket name for client digestable layout for patient information intake"`
	VisualLayoutBucket       string   `long:"patient_layout_bucket" description:"S3 Bucket name for human readable layout for patient information intake"`
	DoctorVisualLayoutBucket string   `long:"doctor_visual_layout_bucket" description:"S3 Bucket name for patient overview for doctor's viewing"`
	DoctorLayoutBucket       string   `long:"doctor_layout_bucket" description:"S3 Bucket name for pre-processed patient overview for doctor's viewing"`
	Debug                    bool     `long:"debug" description:"Enable debugging"`
}

var DefaultConfig = Config{
	BaseConfig: &config.BaseConfig{
		AppName: "resetapi",
	},
	ListenAddr:            ":8080",
	CaseBucket:            "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
}

func main() {
	conf := DefaultConfig
	_, err := config.Parse(&conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.DB.User == "" || conf.DB.Password == "" || conf.DB.Host == "" || conf.DB.Name == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	metricsRegistry := metrics.NewRegistry()
	conf.StartReporters(metricsRegistry)

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

	svcReg, err := conf.ServiceRegistry()
	if err != nil {
		log.Fatalf("Failed to create service registry: %+v", err)
	}
	secureSvcClientBuilder, err := svcclient.NewThriftServiceClientBuilder(svcReg, svcreg.ServiceId{Environment: conf.Environment, Name: "secure"})
	if err != nil {
		log.Fatalf("Failed to create client builder for secure service: %+v", err)
	}
	secureSvcClient := svcclient.NewClient("restapi", 4, secureSvcClientBuilder, metricsRegistry.Scope("securesvc-client"))

	authApi := &thriftapi.AuthClient{Client: secureSvcClient}
	dataApi := &api.DataService{DB: db}
	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	photoAnswerCloudStorageApi := api.NewCloudStorageService(awsAuth)
	authHandler := &apiservice.AuthenticationHandler{AuthApi: authApi}
	signupPatientHandler := &apiservice.SignupPatientHandler{DataApi: dataApi, AuthApi: authApi}
	patientVisitHandler := apiservice.NewPatientVisitHandler(dataApi, authApi, cloudStorageApi, photoAnswerCloudStorageApi)
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(dataApi)
	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(dataApi, photoAnswerCloudStorageApi, conf.CaseBucket, conf.AWSRegion, conf.MaxInMemoryForPhotoMB*1024*1024)
	generateDoctorLayoutHandler := &apiservice.GenerateDoctorLayoutHandler{
		DataApi:                  dataApi,
		CloudStorageApi:          cloudStorageApi,
		DoctorLayoutBucket:       conf.DoctorLayoutBucket,
		DoctorVisualLayoutBucket: conf.DoctorVisualLayoutBucket,
		MaxInMemoryForPhoto:      conf.MaxInMemoryForPhotoMB,
		AWSRegion:                conf.AWSRegion,
	}
	pingHandler := apiservice.PingHandler(0)
	generateModelIntakeHandler := &apiservice.GenerateClientIntakeModelHandler{
		DataApi:             dataApi,
		CloudStorageApi:     cloudStorageApi,
		VisualLayoutBucket:  conf.VisualLayoutBucket,
		PatientLayoutBucket: conf.PatientLayoutBucket,
		AWSRegion:           conf.AWSRegion,
	}
	doctorPatientVisitReviewHandler := &apiservice.DoctorPatientVisitReviewHandler{
		DataApi:                    dataApi,
		LayoutStorageService:       cloudStorageApi,
		PatientPhotoStorageService: photoAnswerCloudStorageApi,
	}

	mux := &apiservice.AuthServeMux{ServeMux: *http.NewServeMux(), AuthApi: authApi}

	mux.Handle("/v1/patient", signupPatientHandler)
	mux.Handle("/v1/visit", patientVisitHandler)
	mux.Handle("/v1/patient_visit_review", doctorPatientVisitReviewHandler)
	mux.Handle("/v1/answer", answerIntakeHandler)
	mux.Handle("/v1/answer/photo", photoAnswerIntakeHandler)
	mux.Handle("/v1/client_model", generateModelIntakeHandler)
	mux.Handle("/v1/doctor_layout", generateDoctorLayoutHandler)

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

	if conf.CertKeyLocation == "" && conf.PrivateKeyLocation == "" {
		log.Fatal(s.ListenAndServe())
	} else {
		log.Fatal(s.ListenAndServeTLS(conf.CertKeyLocation, conf.PrivateKeyLocation))
	}
}
