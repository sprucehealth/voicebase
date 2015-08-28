package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/s3"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	awsAccessKey        = flag.String("aws_access_key", "", "AWS Access Key ID")
	awsSecretKey        = flag.String("aws_secret_key", "", "AWS Secret Key")
	awsRegion           = flag.String("aws_region", "us-east-1", "AWS Region")
	pharmacyDBHost      = flag.String("db_host", "127.0.0.1", "Pharmacy DB Host")
	pharmacyDBUsername  = flag.String("db_username", "", "Pharmacy DB Username")
	pharmacyDBName      = flag.String("db_name", "", "Pharmacy DB Name")
	pharmacyDBPassword  = flag.String("db_password", "", "Pharmacy DB Password")
	libratoUsername     = flag.String("librato.username", "", "Librato username for analytics")
	libratoToken        = flag.String("librato.token", "", "Librato token for analytics")
	libratoSource       = flag.String("librato.source", "", "Librato source for analytics")
	migrationBucketName = flag.String("bucket_name", "", "Pharmacy migration files bucketname")
	arcGISClientID      = flag.String("arcgis_client_id", "", "Client ID for Geocoding using ArcGIS Geocoding Service")
	arcGISClientSecret  = flag.String("arcgis_client_secret", "", "Client Secret for Geocoding using ArcGIS Geocoding Service")
	consulAddress       = flag.String("consul", "127.0.0.1:8500", "Consul HTTP API host:port")
	consulServiceID     = flag.String("consul_service_id", "", "Service ID for Consul. Only needed when running more than one instance on a host")
	sslRequired         = flag.Bool("ssl_required", true, "Require SSL connection to pharmacy DB")
	pharmacyDBPort      = flag.Int("db_port", 5432, "Pharmacy DB Port")
)

var (
	statGeocodingFailed           = metrics.NewCounter()
	statGeocodingSuccessful       = metrics.NewCounter()
	statPharmacyUpdateFailed      = metrics.NewCounter()
	statPharmacyUpdatedSuccessful = metrics.NewCounter()
	statsRegistry                 = metrics.NewRegistry().Scope("pharmacydb")
)

func init() {
	statsRegistry.Add("geocoding/failed", statGeocodingFailed)
	statsRegistry.Add("geocoding/success", statGeocodingSuccessful)
	statsRegistry.Add("pharmacydb/failed", statPharmacyUpdateFailed)
	statsRegistry.Add("pharmacydb/success", statPharmacyUpdatedSuccessful)
}

func main() {
	flag.Parse()

	sslParam := "require"
	if !(*sslRequired) {
		sslParam = "disable"
	}

	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		*pharmacyDBUsername, *pharmacyDBPassword, *pharmacyDBHost, *pharmacyDBPort, *pharmacyDBName, sslParam))
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		golog.Fatalf(err.Error())
	}

	var creds *credentials.Credentials
	if *awsAccessKey != "" && *awsSecretKey != "" {
		creds = credentials.NewStaticCredentials(*awsAccessKey, *awsSecretKey, "")
	} else {
		creds = credentials.NewEnvCredentials()
		if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
			creds = ec2rolecreds.NewCredentials(ec2metadata.New(&ec2metadata.Config{
				HTTPClient: &http.Client{Timeout: 2 * time.Second},
			}), time.Minute*10)
		}
	}
	if *awsRegion == "" {
		az, err := awsutil.GetMetadata(awsutil.MetadataAvailabilityZone)
		if err != nil {
			log.Fatalf("no region specified and failed to get from instance metadata: %+v", err)
		}
		*awsRegion = az[:len(az)-1]
	}

	awsConfig := &aws.Config{
		Credentials: creds,
		Region:      awsRegion,
	}
	s3Client := s3.New(awsConfig)

	consulClient, err := consulapi.NewClient(&consulapi.Config{
		Address:    *consulAddress,
		HttpClient: http.DefaultClient,
	})
	if err != nil {
		golog.Fatalf("Unable to instantiate new consul client: %s", err)
	}

	svc, err := consul.RegisterService(consulClient, *consulServiceID, "pharmacydb", nil, 0)
	if err != nil {
		log.Fatalf("Failed to register service with Consul: %s", err.Error())
	}
	defer svc.Deregister()

	if err := setupLibrato(); err != nil {
		golog.Fatalf(err.Error())
	}

	// start the pharmacy update worker
	(&pharmacyUpdateWorker{
		db:            db,
		s3Client:      s3Client,
		bucketName:    *migrationBucketName,
		consulService: svc,
	}).start()

	// start the geocoding job
	(&geocodingWorker{
		db:            db,
		clientID:      *arcGISClientID,
		clientSecret:  *arcGISClientSecret,
		consulService: svc,
	}).start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}
}

func setupLibrato() error {
	if *libratoUsername == "" || *libratoToken == "" {
		return nil
	}

	source := *libratoSource
	if source == "" {
		var err error
		source, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	statsReporter := reporter.NewLibratoReporter(
		statsRegistry, time.Minute, true, *libratoUsername, *libratoToken, source)
	statsReporter.Start()
	return nil
}
