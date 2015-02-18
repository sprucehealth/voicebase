package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/misc"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
)

func buildRESTAPI(conf *Config, dataAPI api.DataAPI, authAPI api.AuthAPI, diagnosisAPI diagnosis.API, smsAPI api.SMSAPI, eRxAPI erx.ERxAPI,
	dispatcher *dispatch.Dispatcher, consulService *consul.Service, signer *sig.Signer, stores storage.StoreMap,
	rateLimiters ratelimit.KeyedRateLimiters, alog analytics.Logger, metricsRegistry metrics.Registry,
) http.Handler {
	awsAuth, err := conf.AWSAuth()
	if err != nil {
		log.Fatalf("Failed to get AWS auth: %+v", err)
	}

	emailService := email.NewService(dataAPI, conf.Email, metricsRegistry.Scope("email"))
	surescriptsPharmacySearch, err := pharmacy.NewSurescriptsPharmacySearch(conf.PharmacyDB)
	if err != nil {
		if conf.Debug {
			log.Printf("Unable to initialize pharmacy search: %s", err)
		} else {
			log.Fatalf("Unable to initialize pharmacy search: %s", err)
		}
	}

	var erxStatusQueue *common.SQSQueue
	if conf.ERxStatusQueue != "" {
		var err error
		erxStatusQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxStatusQueue)
		if err != nil {
			log.Fatalf("Unable to get erx queue for sending prescriptions to: %s", err.Error())
		}
	} else if conf.Debug {
		erxStatusQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueURL:     "ERxStatusQueue",
		}
	} else if conf.ERxRouting {
		log.Fatal("ERxStatusQueue not configured but ERxRouting is enabled")
	}

	var erxRoutingQueue *common.SQSQueue
	if conf.ERxRoutingQueue != "" {
		var err error
		erxRoutingQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.ERxRoutingQueue)
		if err != nil {
			log.Fatalf("Unable to get erx queue for sending prescriptions to: %s", err.Error())
		}
	} else if conf.Debug {
		erxRoutingQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueURL:     "ERXRoutingQueue",
		}
	} else if conf.ERxRouting {
		log.Fatal("ERxRoutingQueue not configured but ERxRouting is enabled")
	}

	var medicalRecordQueue *common.SQSQueue
	if conf.MedicalRecordQueue != "" {
		medicalRecordQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.MedicalRecordQueue)
		if err != nil {
			log.Fatalf("Failed to get queue for medical record requests: %s", err.Error())
		}
	} else if !conf.Debug {
		log.Fatal("MedicalRecordQueue not configured")
	} else {
		medicalRecordQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueURL:     "MedicalRecord",
		}
	}

	var visitQueue *common.SQSQueue
	if conf.VisitQueue != "" {
		visitQueue, err = common.NewQueue(awsAuth, aws.Regions[conf.AWSRegion], conf.VisitQueue)
		if err != nil {
			log.Fatalf("Failed to get queue for charging visits: %s", err.Error())
		}
	} else if !conf.Debug {
		log.Fatal("VisitQueue not configured")
	} else {
		visitQueue = &common.SQSQueue{
			QueueService: &sqs.Mock{},
			QueueURL:     "Visit",
		}
	}

	snsClient := &sns.SNS{
		Region: aws.Regions[conf.AWSRegion],
		Client: &aws.Client{
			Auth: awsAuth,
		},
	}
	smartyStreetsService := &address.SmartyStreetsService{
		AuthID:    conf.SmartyStreets.AuthID,
		AuthToken: conf.SmartyStreets.AuthToken,
	}

	notificationManager := notify.NewManager(dataAPI, authAPI, snsClient, smsAPI, emailService,
		conf.Twilio.FromNumber, conf.NotifiyConfigs, metricsRegistry.Scope("notify"))

	stripeService := &stripe.StripeService{}
	if conf.TestStripe != nil && conf.TestStripe.SecretKey != "" {
		if conf.Environment == "prod" {
			golog.Warningf("Using test stripe key in production for patient")
		}
		stripeService.SecretKey = conf.TestStripe.SecretKey
	} else {
		stripeService.SecretKey = conf.Stripe.SecretKey
	}

	mediaStore := media.NewStore("https://"+conf.APIDomain+apipaths.MediaURLPath, signer, stores.MustGet("media"))

	url, err := mediaStore.SignedURL(4965, time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", url)

	mux := router.New(&router.Config{
		DataAPI:                  dataAPI,
		AuthAPI:                  authAPI,
		Dispatcher:               dispatcher,
		AuthTokenExpiration:      time.Duration(conf.RegularAuth.ExpireDuration) * time.Second,
		AddressValidationAPI:     smartyStreetsService,
		PharmacySearchAPI:        surescriptsPharmacySearch,
		DiagnosisAPI:             diagnosisAPI,
		SNSClient:                snsClient,
		PaymentAPI:               stripeService,
		NotifyConfigs:            conf.NotifiyConfigs,
		MinimumAppVersionConfigs: conf.MinimumAppVersionConfigs,
		DosespotConfig:           conf.DoseSpot,
		NotificationManager:      notificationManager,
		ERxRoutingQueue:          erxRoutingQueue,
		ERxStatusQueue:           erxStatusQueue,
		ERxAPI:                   eRxAPI,
		VisitQueue:               visitQueue,
		MedicalRecordQueue:       medicalRecordQueue,
		EmailService:             emailService,
		MetricsRegistry:          metricsRegistry,
		SMSAPI:                   smsAPI,
		Stores:                   stores,
		MediaStore:               mediaStore,
		RateLimiters:             rateLimiters,
		MaxCachedItems:           2000,
		ERxRouting:               conf.ERxRouting,
		NumDoctorSelection:       conf.NumDoctorSelection,
		JBCQMinutesThreshold:     conf.JBCQMinutesThreshold,
		CustomerSupportEmail:     conf.Support.CustomerSupportEmail,
		TechnicalSupportEmail:    conf.Support.TechnicalSupportEmail,
		APIDomain:                conf.APIDomain,
		WebDomain:                conf.WebDomain,
		APICDNDomain:             conf.APICDNDomain,
		StaticContentURL:         conf.StaticContentBaseURL,
		StaticResourceURL:        conf.StaticResourceURL,
		AWSRegion:                conf.AWSRegion,
		AnalyticsLogger:          alog,
		TwoFactorExpiration:      conf.TwoFactorExpiration,
		SMSFromNumber:            conf.Twilio.FromNumber,
	})

	if !environment.IsProd() {
		demo.NewWorker(
			dataAPI,
			newConsulLock("service/restapi/training_cases", consulService, conf.Debug),
			conf.APIDomain,
			conf.AWSRegion,
		).Start()
	}

	notifyDoctorLock := newConsulLock("service/restapi/notify_doctor", consulService, conf.Debug)
	refillRequestCheckLock := newConsulLock("service/restapi/check_refill_request", consulService, conf.Debug)
	checkRxErrorsLock := newConsulLock("service/restapi/check_rx_error", consulService, conf.Debug)

	// Start worker to check for expired items in the global case queue
	doctor_queue.StartClaimedItemsExpirationChecker(dataAPI, alog, metricsRegistry.Scope("doctor_queue"))
	if conf.ERxRouting {
		app_worker.NewERxStatusWorker(
			dataAPI,
			eRxAPI,
			dispatcher,
			erxStatusQueue,
			metricsRegistry.Scope("check_erx_status"),
		).Start()
		app_worker.NewRefillRequestWorker(
			dataAPI,
			eRxAPI,
			refillRequestCheckLock,
			dispatcher,
			metricsRegistry.Scope("check_rx_refill_requests"),
		).Start()
		app_worker.NewERxErrorWorker(
			dataAPI,
			eRxAPI,
			checkRxErrorsLock,
			metricsRegistry.Scope("check_rx_errors"),
		).Start()
		doctor_treatment_plan.StartWorker(dataAPI, eRxAPI, dispatcher, erxRoutingQueue, erxStatusQueue, 0, metricsRegistry.Scope("erx_route"))
	}

	medrecord.NewWorker(
		dataAPI,
		medicalRecordQueue,
		emailService,
		conf.Support.CustomerSupportEmail,
		conf.APIDomain,
		conf.WebDomain,
		signer,
		stores.MustGet("medicalrecords"),
		mediaStore,
		time.Duration(conf.RegularAuth.ExpireDuration)*time.Second,
	).Start()

	schedmsg.StartWorker(dataAPI, authAPI, dispatcher, emailService, metricsRegistry.Scope("sched_msg"), 0)
	misc.StartWorker(dataAPI, metricsRegistry)

	cost.NewWorker(
		dataAPI,
		alog,
		dispatcher,
		stripeService,
		emailService,
		visitQueue,
		metricsRegistry.Scope("visit_queue"),
		conf.VisitWorkerTimePeriodSeconds,
		conf.Support.CustomerSupportEmail,
	).Start()

	doctor_queue.NewWorker(
		dataAPI,
		authAPI,
		notifyDoctorLock,
		notificationManager,
		metricsRegistry.Scope("notify_doctors"),
	).Start()

	// seeding random number generator based on time the main function runs
	rand.Seed(time.Now().UTC().UnixNano())

	return httputil.CompressResponse(httputil.DecompressRequest(mux))
}
