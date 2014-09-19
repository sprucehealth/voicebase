package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/s3"
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
	migrationBucketName = flag.String("bucket_name", "", "Pharmacy migration files bucketname")
	arcGISClientID      = flag.String("arcgis_client_id", "", "Client ID for Geocoding using ArcGIS Geocoding Service")
	arcGISClientSecret  = flag.String("arcgis_client_secret", "", "Client Secret for Geocoding using ArcGIS Geocoding Service")
	sslRequired         = flag.Bool("ssl_required", true, "Require SSL connection to pharmacy DB")
	pharmacyDBPort      = flag.Int("db_port", 3305, "Pharmacy DB Port")
)

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

	s3Client := &s3.S3{
		Region: aws.Regions[*awsRegion],
		Client: &aws.Client{
			Auth: aws.Keys{
				AccessKey: *awsAccessKey,
				SecretKey: *awsSecretKey,
			},
		},
	}

	// start the pharmacy update worker
	(&pharmacyUpdateWorker{
		db:         db,
		s3Client:   s3Client,
		bucketName: *migrationBucketName,
	}).start()

	// start the geocoding job
	(&geocodingWorker{
		db:           db,
		clientID:     *arcGISClientID,
		clientSecret: *arcGISClientSecret,
	}).start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}

	// TODO: Ensure to stop the job in the event that there is a failed job

	// TODO: Use consul to acquire a service lock

	// TODO: Run job in staging
}
