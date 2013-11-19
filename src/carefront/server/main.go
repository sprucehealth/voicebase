package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/aws"
	"carefront/util"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	flags "github.com/jessevdk/go-flags"
	goamz "launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
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
	ListenAddr            string   `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	CertLocation          string   `long:"cert_key" description:"Path of SSL certificate"`
	KeyLocation           string   `long:"private_key" description:"Path of SSL private key"`
	S3CaseBucket          string   `long:"case_bucket" description:"S3 Bucket name for case information"`
	AWSRegion             string   `long:"aws_region" description:"AWS region"`
	AWSRole               string   `long:"aws_role" description:"AWS role for fetching temporary credentials"`
	AWSSecretKey          string   `long:"aws_secret_key" description:"AWS secret key"`
	AWSAccessKey          string   `long:"aws_access_key" description:"AWS access key id"`
	DB                    DBConfig `group:"Database" toml:"database"`
	Debug                 bool     `long:"debug" description:"Enable debugging"`
	LogPath               string   `long:"log_path" description:"Path for log file. IF not given then default to stderr"`
	LayoutBucketAccessKey string   `long:"layout_bucket_access_key" description:"AWS access key for buckets where patient, client and doctor layouts are stored"`
	LayoutBucketSecretKey string   `long:"layout_bucket_secret_key" description:"AWS secrety key for buckets where patient, client and doctor layouts are stored"`
	MaxInMemoryForPhotoMB int64    `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
	ConfigPath            string   `short:"c" long:"config" description:"Path to config file"`

	awsAuth aws.Auth
}

func (c *Config) AWSAuth() (aws.Auth, error) {
	var err error
	if c.awsAuth == nil {
		if c.AWSRole != "" {
			c.awsAuth, err = aws.CredentialsForRole(c.AWSRole)
		} else {
			keys := aws.KeysFromEnvironment()
			if c.AWSAccessKey != "" && c.AWSSecretKey != "" {
				keys.AccessKey = c.AWSAccessKey
				keys.SecretKey = c.AWSSecretKey
			} else {
				c.AWSAccessKey = keys.AccessKey
				c.AWSSecretKey = keys.SecretKey
			}
			c.awsAuth = keys
		}
	}
	if err != nil {
		c.awsAuth = nil
	}
	return c.awsAuth, err
}

var DefaultConfig = Config{
	ListenAddr:            ":8080",
	S3CaseBucket:          "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
}

func parseFlagsAndConfig() (*Config, []string) {
	config := DefaultConfig
	args, err := flags.ParseArgs(&config, os.Args[1:])
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			log.Fatalf("Failed to parse flags: %+v", err)
		}
		os.Exit(1)
	}

	if config.ConfigPath != "" {
		if strings.Contains(config.ConfigPath, "://") {
			awsAuth, err := config.AWSAuth()
			if err != nil {
				log.Fatalf("Failed to get AWS auth: %+v", err)
			}
			ur, err := url.Parse(config.ConfigPath)
			if err != nil {
				log.Fatalf("Failed to parse config url %s: %+v", config.ConfigPath, err)
			}
			var rd io.ReadCloser
			if ur.Scheme == "s3" {
				s3 := s3.New(util.AWSAuthAdapter(awsAuth), goamz.USEast)
				rd, err = s3.Bucket(ur.Host).GetReader(ur.Path)
				if err != nil {
					log.Fatalf("Failed to get config from s3 %s: %+v", config.ConfigPath, err)
				}
			} else {
				if res, err := http.Get(config.ConfigPath); err != nil {
					log.Fatalf("Failed to fetch config from URL %s: %+v", config.ConfigPath, err)
				} else if res.StatusCode != 200 {
					log.Fatalf("Failed to fetch config from URL %s: status code %d", config.ConfigPath, res.StatusCode)
				} else {
					rd = res.Body
				}
			}
			if _, err := toml.DecodeReader(rd, &config); err != nil {
				log.Fatalf("Failed to parse config file: %+v", err)
			}
			rd.Close()
		} else if _, err := toml.DecodeFile(config.ConfigPath, &config); err != nil {
			log.Fatalf("Failed to parse config file: %+v", err)
		}
		// Make sure command line overrides config
		flags.ParseArgs(&config, os.Args[1:])
	}

	if config.AWSRegion == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			log.Fatalf("No region specified and failed to get from instance metadata: %+v", err)
		}
		config.AWSRegion = az[:len(az)-1]
		log.Printf("Got region from metadata: %s", config.AWSRegion)
	}

	if config.LogPath != "" {
		// check if the file exists
		_, err := os.Stat(config.LogPath)
		var file *os.File
		if os.IsNotExist(err) {
			// file doesn't exist so lets create it
			file, err = os.Create(config.LogPath)
			if err != nil {
				log.Fatalf("Failed to create log: %s", err.Error())
			}
		} else {
			file, err = os.OpenFile(config.LogPath, os.O_RDWR|os.O_APPEND, 0660)
			if err != nil {
				log.Printf("Could not open logfile %s", err.Error())
			}
		}
		log.SetOutput(file)
	}
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	if config.DB.User == "" || config.DB.Password == "" || config.DB.Host == "" || config.DB.Name == "" {
		fmt.Fprintf(os.Stderr, "Missing either one of user, password, host, or name for the database.\n")
		os.Exit(1)
	}

	return &config, args
}

func main() {
	config, _ := parseFlagsAndConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", config.DB.User, config.DB.Password, config.DB.Host, config.DB.Name)

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

	awsAuth, err := config.AWSAuth()
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
	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(dataApi, photoAnswerCloudStorageApi, config.S3CaseBucket, config.MaxInMemoryForPhotoMB*1024*1024)
	pingHandler := apiservice.PingHandler(0)
	generateModelIntakeHandler := &apiservice.GenerateClientIntakeModelHandler{dataApi, cloudStorageApi}

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
		Addr:           config.ListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if config.CertLocation == "" && config.KeyLocation == "" {
		log.Fatal(s.ListenAndServe())
	} else {
		log.Fatal(s.ListenAndServeTLS(config.CertLocation, config.KeyLocation))
	}
}
