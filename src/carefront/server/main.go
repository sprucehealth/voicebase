package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"carefront/api"
	"carefront/apiservice"
	_ "github.com/go-sql-driver/mysql"
)

var (
	flagListenAddr   = flag.String("listen", ":8080", "Address and port to listen on")
	flagCertLocation = flag.String("cert_key", "", "Location of certificate for SSL")
	flagKeyLocation  = flag.String("private_key", "", "Location of key for SSL")
	flagS3CaseBucket = flag.String("case_bucket", "carefront-cases", "Bucket name holding case information on S3")
	flagAWSSecretKey = flag.String("aws_secret_key", "", "AWS Secret Key for uploading files to S3")
	flagAWSAccessKey = flag.String("aws_access_key", "", "AWS Access Key to upload files to S3")
	flagDBUser       = flag.String("db_user", "", "Username for accessing database")
	flagDBPassword   = flag.String("db_password", "", "Password for accessing database")
	flagDBHost       = flag.String("db_host", "", "Database host url")
	flagDBName       = flag.String("db_name", "", "Database name on database server")
	flagDebugMode    = flag.Bool("debug", false, "Flag to indicate whether we are running in debug or production mode")
	flagLogPath      = flag.String("log_path", "", "Path for log file. If not given then default to stderr")
)

func parseFlags() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "@") {
		f, err := os.Open(os.Args[1][1:])
		if err == nil {
			argBytes, err := ioutil.ReadAll(f)
			f.Close()
			if err == nil {
				args := strings.Split(strings.TrimSpace(string(argBytes)), "\n")
				filteredArgs := make([]string, 0, len(args))
				for _, a := range args {
					if !strings.HasPrefix(a, "#") {
						filteredArgs = append(filteredArgs, strings.TrimSpace(a))
					}
				}
				os.Args = append(append(os.Args[:1], filteredArgs...), os.Args[2:]...)
			}
		}
	}
	flag.VisitAll(func(fl *flag.Flag) {
		val := os.Getenv("arg_" + strings.Replace(fl.Name, ".", "_", -1))
		if val != "" {
			fl.Value.Set(val)
		}
	})
	flag.Parse()
}

func main() {
	parseFlags()

	if *flagLogPath != "" {
		// check if the file exists
		_, err := os.Stat(*flagLogPath)
		var file *os.File
		if os.IsNotExist(err) {
			// file doesn't exist so lets create it
			file, err = os.Create(*flagLogPath)
			if err != nil {
				log.Fatalf("Failed to create log: %s", err.Error())
			}
		} else {
			file, err = os.OpenFile(*flagLogPath, os.O_RDWR|os.O_APPEND, 0660)
			if err != nil {
				log.Printf("Could not open logfile %s", err.Error())
			}
		}
		log.SetOutput(file)
	}
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	if *flagDBUser == "" || *flagDBPassword == "" || *flagDBHost == "" || *flagDBName == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", *flagDBUser, *flagDBPassword, *flagDBHost, *flagDBName)

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

	authApi := &api.AuthService{db}
	dataApi := &api.DataService{db}
	mux := &apiservice.AuthServeMux{*http.NewServeMux(), authApi}
	cloudStorageApi := api.NewCloudStorageService("AKIAINP33PBIN5GW4GKQ", "rbqPao4jDqTBTXBHk4BRnzWmYsfvSslg9mYhG45w")
	authHandler := &apiservice.AuthenticationHandler{authApi}
	pingHandler := apiservice.PingHandler(0)
	photoHandler := &apiservice.PhotoUploadHandler{&api.PhotoService{*flagAWSAccessKey, *flagAWSSecretKey}, *flagS3CaseBucket, dataApi}
	getSignedUrlsHandler := &apiservice.GetSignedUrlsHandler{&api.PhotoService{*flagAWSAccessKey, *flagAWSSecretKey}, *flagS3CaseBucket}
	generateModelIntakeHandler := &apiservice.GenerateClientIntakeModelHandler{dataApi, cloudStorageApi}

	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/signup", authHandler)
	mux.Handle("/v1/logout", authHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/upload", photoHandler)
	mux.Handle("/v1/imagesforcase/", getSignedUrlsHandler)
	mux.Handle("/v1/generate_client_model/", generateModelIntakeHandler)

	s := &http.Server{
		Addr:           *flagListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if *flagDebugMode && *flagCertLocation == "" && *flagKeyLocation == "" {
		log.Fatal(s.ListenAndServe())
	} else {
		log.Fatal(s.ListenAndServeTLS(*flagCertLocation, *flagKeyLocation))
	}
}
