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
	"carefront/app_worker"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/aws"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"carefront/libs/maps"
	"carefront/libs/pharmacy"
	"carefront/libs/svcclient"
	"carefront/libs/svcreg"
	"carefront/services/auth"
	thriftapi "carefront/thrift/api"
	"github.com/SpruceHealth/go-proxy-protocol/proxyproto"
	"github.com/go-sql-driver/mysql"
	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
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

type TwilioConfig struct {
	AccountSid string `long:"twilio_account_sid" description:"Twilio AccountSid"`
	AuthToken  string `long:"twilio_auth_token" description:"Twilio AuthToken"`
	FromNumber string `long:"twilio_from_number" description:"Twilio From Number for Messages"`
}

type Config struct {
	*config.BaseConfig
	ProxyProtocol            bool          `long:"proxy_protocol" description:"Enable if behind a proxy that uses the PROXY protocol"`
	ListenAddr               string        `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSListenAddr            string        `long:"tls_listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSCert                  string        `long:"tls_cert" description:"Path of SSL certificate"`
	TLSKey                   string        `long:"tls_key" description:"Path of SSL private key"`
	DB                       *DBConfig     `group:"Database" toml:"database"`
	PharmacyDB               *DBConfig     `group:"PharmacyDatabase" toml:"pharmacy_database"`
	MaxInMemoryForPhotoMB    int64         `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
	ContentBucket            string        `long:"content_bucket" description:"S3 Bucket name for all static content"`
	CaseBucket               string        `long:"case_bucket" description:"S3 Bucket name for case information"`
	PatientLayoutBucket      string        `long:"client_layout_bucket" description:"S3 Bucket name for client digestable layout for patient information intake"`
	VisualLayoutBucket       string        `long:"patient_layout_bucket" description:"S3 Bucket name for human readable layout for patient information intake"`
	DoctorVisualLayoutBucket string        `long:"doctor_visual_layout_bucket" description:"S3 Bucket name for patient overview for doctor's viewing"`
	DoctorLayoutBucket       string        `long:"doctor_layout_bucket" description:"S3 Bucket name for pre-processed patient overview for doctor's viewing"`
	Debug                    bool          `long:"debug" description:"Enable debugging"`
	IOSDeeplinkScheme        string        `long:"ios_deeplink_scheme" description:"Scheme for iOS deep-links (e.g. spruce://)"`
	DoseSpotClinicKey        string        `long:"dose_spot_clinic_key" description:"DoseSpot Clinic Key for eRX integration"`
	DoseSpotClinicId         string        `long:"dose_spot_clinic_id" description:"DoseSpot Clinic Id for eRX integration"`
	DoseSpotUserId           string        `long:"dose_spot_user_id" description:"DoseSpot UserId for eRx integration"`
	NoServices               bool          `long:"noservices" description:"Disable connecting to remote services"`
	ERxRouting               bool          `long:"erx_routing" description:"Disable sending of prescriptions electronically"`
	ERxQueue                 string        `long:"erx_queue" description:"Erx queue name"`
	AuthTokenExpiration      int           `long:"auth_token_expire" description:"Expiration time in seconds for the auth token"`
	AuthTokenRenew           int           `long:"auth_token_renew" description:"Time left below which to renew the auth token"`
	StaticContentBaseUrl     string        `long:"static_content_base_url" description:"URL from which to serve static content"`
	Twilio                   *TwilioConfig `group:"Twilio" toml:"twilio"`
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
	Twilio:                &TwilioConfig{},
	ListenAddr:            ":8080",
	TLSListenAddr:         ":8443",
	CaseBucket:            "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
	AuthTokenExpiration:   60 * 60 * 24 * 2,
	AuthTokenRenew:        60 * 60 * 36,
	IOSDeeplinkScheme:     "spruce",
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
	if conf.NoServices || conf.BaseConfig.ZookeeperHosts == "" {
		if conf.NoServices || conf.Debug {
			authApi = &auth.AuthService{
				DB:             db,
				ExpireDuration: time.Duration(conf.AuthTokenExpiration) * time.Second,
				RenewDuration:  time.Duration(conf.AuthTokenRenew) * time.Second,
			}
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

	var twilioCli *twilio.Client
	if conf.Twilio != nil && conf.Twilio.AccountSid != "" && conf.Twilio.AuthToken != "" {
		twilioCli = twilio.NewClient(conf.Twilio.AccountSid, conf.Twilio.AuthToken, nil)
	}

	mapsService := maps.NewGoogleMapsService(metricsRegistry.Scope("google_maps_api"))
	doseSpotService := erx.NewDoseSpotService(conf.DoseSpotClinicId, conf.DoseSpotClinicKey, conf.DoseSpotUserId, metricsRegistry.Scope("dosespot_api"))

	dataApi := &api.DataService{DB: db}
	cloudStorageApi := api.NewCloudStorageService(awsAuth)
	photoAnswerCloudStorageApi := api.NewCloudStorageService(awsAuth)
	authHandler := &apiservice.AuthenticationHandler{AuthApi: authApi, DataApi: dataApi, PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0), StaticContentBaseUrl: conf.StaticContentBaseUrl}
	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: dataApi, MapsService: mapsService, StaticContentUrl: conf.StaticContentBaseUrl}
	signupPatientHandler := &apiservice.SignupPatientHandler{DataApi: dataApi, AuthApi: authApi, MapsApi: mapsService}
	updatePatientBillingAddress := &apiservice.UpdatePatientAddressHandler{DataApi: dataApi, AddressType: apiservice.BILLING_ADDRESS_TYPE}
	updatePatientPharmacyHandler := &apiservice.UpdatePatientPharmacyHandler{DataApi: dataApi, PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0)}
	authenticateDoctorHandler := &apiservice.DoctorAuthenticationHandler{DataApi: dataApi, AuthApi: authApi}
	signupDoctorHandler := &apiservice.SignupDoctorHandler{DataApi: dataApi, AuthApi: authApi}
	patientVisitHandler := apiservice.NewPatientVisitHandler(dataApi, authApi, cloudStorageApi, photoAnswerCloudStorageApi, twilioCli, conf.Twilio.FromNumber)
	patientVisitReviewHandler := &apiservice.PatientVisitReviewHandler{DataApi: dataApi}
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(dataApi)
	autocompleteHandler := &apiservice.AutocompleteHandler{ERxApi: doseSpotService, Role: api.PATIENT_ROLE}
	doctorTreatmentSuggestionHandler := &apiservice.AutocompleteHandler{ERxApi: doseSpotService, Role: api.DOCTOR_ROLE}
	doctorInstructionsHandler := apiservice.NewDoctorDrugInstructionsHandler(dataApi)
	doctorFollowupHandler := apiservice.NewPatientVisitFollowUpHandler(dataApi)
	doctorFavoriteTreatmentsHandler := &apiservice.DoctorFavoriteTreatmentsHandler{DataApi: dataApi}
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{ERxApi: doseSpotService}
	newTreatmentHandler := &apiservice.NewTreatmentHandler{ERxApi: doseSpotService}
	medicationDispenseUnitHandler := &apiservice.MedicationDispenseUnitsHandler{DataApi: dataApi}
	treatmentsHandler := apiservice.NewTreatmentsHandler(dataApi)
	photoAnswerIntakeHandler := apiservice.NewPhotoAnswerIntakeHandler(dataApi, photoAnswerCloudStorageApi, conf.CaseBucket, conf.AWSRegion, conf.MaxInMemoryForPhotoMB*1024*1024)
	pharmacySearchHandler := &apiservice.PharmacyTextSearchHandler{PharmacySearchService: pharmacy.GooglePlacesPharmacySearchService(0), DataApi: dataApi, MapsService: mapsService}
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
		PharmacySearchService:      pharmacy.GooglePlacesPharmacySearchService(0),
		PatientPhotoStorageService: photoAnswerCloudStorageApi,
	}
	staticContentHandler := &apiservice.StaticContentHandler{
		DataApi:               dataApi,
		ContentStorageService: cloudStorageApi,
		BucketLocation:        conf.ContentBucket,
		Region:                conf.AWSRegion,
	}
	doctorPrescriptionsHandler := &apiservice.DoctorPrescriptionsHandler{
		DataApi: dataApi,
	}
	doctorPrescriptionsNotificationsHandler := &apiservice.DoctorPrescriptionsNotificationsHandler{
		DataApi: dataApi,
		ErxApi:  doseSpotService,
	}
	doctorPrescriptionErrorsHandler := &apiservice.DoctorPrescriptionsErrorsHandler{
		DataApi: dataApi,
		ErxApi:  doseSpotService,
	}

	erxStatusQueue, err := common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxQueue)
	if err != nil {
		log.Fatal("Unable to get erx queue for sending prescriptions to: " + err.Error())
	}
	doctorSubmitPatientVisitHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{DataApi: dataApi,
		ERxApi:            doseSpotService,
		TwilioFromNumber:  conf.Twilio.FromNumber,
		TwilioCli:         twilioCli,
		IOSDeeplinkScheme: conf.IOSDeeplinkScheme,
		ErxStatusQueue:    erxStatusQueue,
		ERxRouting:        conf.ERxRouting}

	diagnosePatientHandler := apiservice.NewDiagnosePatientHandler(dataApi, authApi, cloudStorageApi)
	diagnosisSummaryHandler := &apiservice.DiagnosisSummaryHandler{DataApi: dataApi}
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(dataApi)
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(dataApi)
	doctorQueueHandler := &apiservice.DoctorQueueHandler{DataApi: dataApi}
	mux := apiservice.NewAuthServeMux(authApi, metricsRegistry.Scope("restapi"))

	mux.Handle("/v1/content", staticContentHandler)
	mux.Handle("/v1/patient", signupPatientHandler)
	mux.Handle("/v1/patient/address/billing", updatePatientBillingAddress)
	mux.Handle("/v1/patient/pharmacy", updatePatientPharmacyHandler)
	mux.Handle("/v1/visit", patientVisitHandler)
	mux.Handle("/v1/visit/review", patientVisitReviewHandler)
	mux.Handle("/v1/check_eligibility", checkElligibilityHandler)
	mux.Handle("/v1/answer", answerIntakeHandler)
	mux.Handle("/v1/answer/photo", photoAnswerIntakeHandler)
	mux.Handle("/v1/signup", authHandler)
	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/isauthenticated", authHandler)
	mux.Handle("/v1/logout", authHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/autocomplete", autocompleteHandler)
	mux.Handle("/v1/pharmacy", pharmacySearchHandler)
	mux.Handle("/v1/doctor_layout", generateDoctorLayoutHandler)
	mux.Handle("/v1/diagnose_layout", generateDiagnoseLayoutHandler)
	mux.Handle("/v1/client_model", generateModelIntakeHandler)

	mux.Handle("/v1/doctor/signup", signupDoctorHandler)
	mux.Handle("/v1/doctor/authenticate", authenticateDoctorHandler)
	mux.Handle("/v1/doctor/queue", doctorQueueHandler)
	mux.Handle("/v1/doctor/treatment/favorite", doctorFavoriteTreatmentsHandler)
	mux.Handle("/v1/doctor/rx", doctorPrescriptionsHandler)
	mux.Handle("/v1/doctor/rx/notifications", doctorPrescriptionsNotificationsHandler)
	mux.Handle("/v1/doctor/rx/errors", doctorPrescriptionErrorsHandler)

	mux.Handle("/v1/doctor/visit/review", doctorPatientVisitReviewHandler)
	mux.Handle("/v1/doctor/visit/diagnosis", diagnosePatientHandler)
	mux.Handle("/v1/doctor/visit/diagnosis/summary", diagnosisSummaryHandler)
	mux.Handle("/v1/doctor/visit/treatment/new", newTreatmentHandler)
	mux.Handle("/v1/doctor/visit/treatment/treatments", treatmentsHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_suggestions", doctorTreatmentSuggestionHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_strengths", medicationStrengthSearchHandler)
	mux.Handle("/v1/doctor/visit/treatment/medication_dispense_units", medicationDispenseUnitHandler)
	mux.Handle("/v1/doctor/visit/treatment/supplemental_instructions", doctorInstructionsHandler)
	mux.Handle("/v1/doctor/visit/regimen", doctorRegimenHandler)
	mux.Handle("/v1/doctor/visit/advice", doctorAdviceHandler)
	mux.Handle("/v1/doctor/visit/followup", doctorFollowupHandler)
	mux.Handle("/v1/doctor/visit/submit", doctorSubmitPatientVisitHandler)

	app_worker.StartWorkerToUpdatePrescriptionStatusForPatient(dataApi, doseSpotService, erxStatusQueue, metricsRegistry.Scope("check_erx_status"))

	s := &http.Server{
		Addr:           conf.ListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if conf.TLSCert != "" && conf.TLSKey != "" {
		go func() {
			s.TLSConfig = &tls.Config{
				MinVersion:               tls.VersionTLS10,
				PreferServerCipherSuites: true,
				CipherSuites: []uint16{
					// Do not include RC4 or 3DES
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
			}
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

			if conf.ProxyProtocol {
				conn = &proxyproto.Listener{Listener: conn}
			}

			ln := tls.NewListener(conn, s.TLSConfig)

			golog.Infof("Starting SSL server on %s...", conf.TLSListenAddr)
			log.Fatal(s.Serve(ln))
		}()
	}
	golog.Infof("Starting server on %s...", conf.ListenAddr)

	log.Fatal(s.ListenAndServe())
}
