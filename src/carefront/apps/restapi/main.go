package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/common/config"
	"carefront/libs/erx"
	"carefront/libs/maps"
	"carefront/libs/pharmacy"
	"carefront/libs/svcclient"
	"carefront/libs/svcreg"
	"carefront/services/auth"
	thriftapi "carefront/thrift/api"
	"github.com/go-sql-driver/mysql"
	"github.com/samuel/go-metrics/metrics"
)

const (
	defaultMaxInMemoryPhotoMB = 2
)

type DBConfig struct {
	User     string `long:"db_user" description:"Username for accessing database"`
	Password string `long:"db_password" description:"Password for accessing database"`
	Host     string `long:"db_host" description:"Database host"`
	Port     int    `long:"db_port" description:"Database port"`
	Name     string `long:"db_name" description:"Database name"`
	CACert   string `long:"db_cacert" description:"Database TLS CA certificate path"`
	TLSCert  string `long:"db_cert" description:"Database TLS client certificate path"`
	TLSKey   string `long:"db_key" description:"Database TLS client key path"`
}

type Config struct {
	*config.BaseConfig
	ListenAddr               string    `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSListenAddr            string    `long:"tls_listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSCert                  string    `long:"tls_cert" description:"Path of SSL certificate"`
	TLSKey                   string    `long:"tls_key" description:"Path of SSL private key"`
	DB                       *DBConfig `group:"Database" toml:"database"`
	PharmacyDB               *DBConfig `group:"PharmacyDatabase" toml:"pharmacy_database"`
	MaxInMemoryForPhotoMB    int64     `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
	CaseBucket               string    `long:"case_bucket" description:"S3 Bucket name for case information"`
	PatientLayoutBucket      string    `long:"client_layout_bucket" description:"S3 Bucket name for client digestable layout for patient information intake"`
	VisualLayoutBucket       string    `long:"patient_layout_bucket" description:"S3 Bucket name for human readable layout for patient information intake"`
	DoctorVisualLayoutBucket string    `long:"doctor_visual_layout_bucket" description:"S3 Bucket name for patient overview for doctor's viewing"`
	DoctorLayoutBucket       string    `long:"doctor_layout_bucket" description:"S3 Bucket name for pre-processed patient overview for doctor's viewing"`
	Debug                    bool      `long:"debug" description:"Enable debugging"`
	DoseSpotClinicKey        string    `long:"dose_spot_clinic_key" description:"DoseSpot Clinic Key for eRX integration"`
	DoseSpotClinicId         string    `long:"dose_spot_clinic_id" description:"DoseSpot Clinic Id for eRX integration"`
	DoseSpotUserId           string    `long:"dose_spot_user_id" description:"DoseSpot UserId for eRx integration"`
}

var DefaultConfig = Config{
	BaseConfig: &config.BaseConfig{
		AppName: "restapi",
	},
	DB: &DBConfig{
		Name: "carefront",
		Host: "127.0.0.1",
		Port: 3306,
	},
	ListenAddr:            ":8080",
	TLSListenAddr:         ":8443",
	CaseBucket:            "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
}

func connectToDatabase(conf *Config, dbConf *DBConfig) (*sql.DB, error) {
	enableTLS := dbConf.CACert != "" && dbConf.TLSCert != "" && dbConf.TLSKey != ""
	if enableTLS {
		rootCertPool := x509.NewCertPool()
		pem, err := conf.ReadURI(dbConf.CACert)
		if err != nil {
			return nil, err
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return nil, fmt.Errorf("Failed to append PEM.")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		cert, err := conf.ReadURI(dbConf.TLSCert)
		if err != nil {
			return nil, err
		}
		key, err := conf.ReadURI(dbConf.TLSKey)
		if err != nil {
			return nil, err
		}
		certs, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)
		mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:      rootCertPool,
			Certificates: clientCert,
		})
	}

	tlsOpt := "?parseTime=true"
	if enableTLS {
		tlsOpt += "&tls=custom"
	}
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.Name, tlsOpt))
	if err != nil {
		return nil, err
	}
	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
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

	db, err := connectToDatabase(&conf, conf.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pharmacyDb, err := connectToDatabase(&conf, conf.PharmacyDB)
	if err != nil {
		log.Fatal(err)
	}
	defer pharmacyDb.Close()

	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}

	svcReg, err := conf.ServiceRegistry()
	if err != nil {
		log.Fatalf("Failed to create service registry: %+v", err)
	}

	var authApi thriftapi.Auth
	if conf.BaseConfig.ZookeeperHosts == "" {
		if conf.Debug {
			authApi = &auth.AuthService{DB: db}
		} else {
			log.Fatalf("No Zookeeper hosts defined and not running under debug")
		}
	} else {
		secureSvcClientBuilder, err := svcclient.NewThriftServiceClientBuilder(svcReg, svcreg.ServiceId{Environment: conf.Environment, Name: "secure"})
		if err != nil {
			log.Fatalf("Failed to create client builder for secure service: %+v", err)
		}
		secureSvcClient := svcclient.NewClient("restapi", 4, secureSvcClientBuilder, metricsRegistry.Scope("securesvc-client"))
		authApi = &thriftapi.AuthClient{Client: secureSvcClient}
	}

	dataApi := &api.DataService{DB: db}
	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	photoAnswerCloudStorageApi := api.NewCloudStorageService(awsAuth)
	authHandler := &apiservice.AuthenticationHandler{AuthApi: authApi}
	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: dataApi, MapsService: maps.GoogleMapsService(0)}
	signupPatientHandler := &apiservice.SignupPatientHandler{DataApi: dataApi, AuthApi: authApi}
	authenticateDoctorHandler := &apiservice.DoctorAuthenticationHandler{DataApi: dataApi, AuthApi: authApi}
	signupDoctorHandler := &apiservice.SignupDoctorHandler{DataApi: dataApi, AuthApi: authApi}
	patientVisitHandler := apiservice.NewPatientVisitHandler(dataApi, authApi, cloudStorageApi, photoAnswerCloudStorageApi)
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(dataApi)
	autocompleteHandler := &apiservice.AutocompleteHandler{ERxApi: erx.NewDoseSpotService(conf.DoseSpotClinicId, conf.DoseSpotClinicKey, conf.DoseSpotUserId), Role: api.PATIENT_ROLE}
	doctorTreatmentSuggestionHandler := &apiservice.AutocompleteHandler{ERxApi: erx.NewDoseSpotService(conf.DoseSpotClinicId, conf.DoseSpotClinicKey, conf.DoseSpotUserId), Role: api.DOCTOR_ROLE}
	doctorInstructionsHandler := apiservice.NewDoctorDrugInstructionsHandler(dataApi)
	doctorFollowupHandler := apiservice.NewPatientVisitFollowUpHandler(dataApi)
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{ERxApi: erx.NewDoseSpotService(conf.DoseSpotClinicId, conf.DoseSpotClinicKey, conf.DoseSpotUserId)}
	newTreatmentHandler := &apiservice.NewTreatmentHandler{ERxApi: erx.NewDoseSpotService(conf.DoseSpotClinicId, conf.DoseSpotClinicKey, conf.DoseSpotUserId)}
	medicationDispenseUnitHandler := &apiservice.MedicationDispenseUnitsHandler{DataApi: dataApi}
	treatmentsHandler := apiservice.NewTreatmentsHandler(dataApi)
	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(dataApi, photoAnswerCloudStorageApi, conf.CaseBucket, conf.AWSRegion, conf.MaxInMemoryForPhotoMB*1024*1024)
	pharmacySearchHandler := &apiservice.PharmacySearchHandler{PharmacySearchService: &pharmacy.PharmacySearchService{PharmacyDB: pharmacyDb}, MapsService: maps.GoogleMapsService(0)}
	googlePlacesPharmacySearch := &apiservice.PharmacySearchHandler{PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0), MapsService: maps.GoogleMapsService(0)}
	generateDoctorLayoutHandler := &apiservice.GenerateDoctorLayoutHandler{
		DataApi:                  dataApi,
		CloudStorageApi:          cloudStorageApi,
		DoctorLayoutBucket:       conf.DoctorLayoutBucket,
		DoctorVisualLayoutBucket: conf.DoctorVisualLayoutBucket,
		MaxInMemoryForPhoto:      conf.MaxInMemoryForPhotoMB,
		AWSRegion:                conf.AWSRegion,
		Purpose:                  api.REVIEW_PURPOSE,
	}
	generateDiagnoseLayoutHandler := &apiservice.GenerateDoctorLayoutHandler{
		DataApi:                  dataApi,
		CloudStorageApi:          cloudStorageApi,
		DoctorLayoutBucket:       conf.DoctorLayoutBucket,
		DoctorVisualLayoutBucket: conf.DoctorVisualLayoutBucket,
		MaxInMemoryForPhoto:      conf.MaxInMemoryForPhotoMB,
		AWSRegion:                conf.AWSRegion,
		Purpose:                  api.DIAGNOSE_PURPOSE,
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
	diagnosePatientHandler := apiservice.NewDiagnosePatientHandler(dataApi, authApi, cloudStorageApi)

	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(dataApi)
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(dataApi)

	mux := &apiservice.AuthServeMux{ServeMux: *http.NewServeMux(), AuthApi: authApi}

	mux.Handle("/v1/patient", signupPatientHandler)
	mux.Handle("/v1/visit", patientVisitHandler)
	mux.Handle("/v1/check_eligibility", checkElligibilityHandler)
	mux.Handle("/v1/patient_visit_review", doctorPatientVisitReviewHandler)
	mux.Handle("/v1/answer", answerIntakeHandler)
	mux.Handle("/v1/answer/photo", photoAnswerIntakeHandler)
	mux.Handle("/v1/signup", authHandler)
	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/logout", authHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/autocomplete", autocompleteHandler)
	mux.Handle("/v1/pharmacy", pharmacySearchHandler)
	mux.Handle("/v1/places/pharmacy", googlePlacesPharmacySearch)

	mux.Handle("/v1/doctor_layout", generateDoctorLayoutHandler)
	mux.Handle("/v1/diagnose_layout", generateDiagnoseLayoutHandler)
	mux.Handle("/v1/client_model", generateModelIntakeHandler)

	mux.Handle("/v1/doctor/signup", signupDoctorHandler)
	mux.Handle("/v1/doctor/authenticate", authenticateDoctorHandler)
	mux.Handle("/v1/doctor/diagnosis", diagnosePatientHandler)

	mux.Handle("/v1/doctor/treatment/medication_suggestions", doctorTreatmentSuggestionHandler)
	mux.Handle("/v1/doctor/treatment/medication_strengths", medicationStrengthSearchHandler)
	mux.Handle("/v1/doctor/treatment/medication_dispense_units", medicationDispenseUnitHandler)

	mux.Handle("/v1/doctor/treatment/new", newTreatmentHandler)
	mux.Handle("/v1/doctor/treatment/treatments", treatmentsHandler)
	mux.Handle("/v1/doctor/treatment/supplemental_instructions", doctorInstructionsHandler)

	mux.Handle("/v1/doctor/regimen/", doctorRegimenHandler)
	mux.Handle("/v1/doctor/advice/", doctorAdviceHandler)
	mux.Handle("/v1/doctor/followup/", doctorFollowupHandler)

	s := &http.Server{
		Addr:           conf.ListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if conf.TLSCert != "" && conf.TLSKey != "" {
		go func() {
			s.TLSConfig = &tls.Config{}
			if s.TLSConfig.NextProtos == nil {
				s.TLSConfig.NextProtos = []string{"http/1.1"}
			}

			cert, err := conf.ReadURI(conf.TLSCert)
			if err != nil {
				log.Fatal(err)
			}
			key, err := conf.ReadURI(conf.TLSKey)
			if err != nil {
				log.Fatal(err)
			}
			certs, err := tls.X509KeyPair(cert, key)
			if err != nil {
				log.Fatal(err)
			}

			s.TLSConfig.Certificates = []tls.Certificate{certs}

			conn, err := net.Listen("tcp", conf.TLSListenAddr)
			if err != nil {
				log.Fatal(err)
			}

			log.Fatal(s.Serve(tls.NewListener(conn, s.TLSConfig)))
		}()
	}
	log.Fatal(s.ListenAndServe())
}
