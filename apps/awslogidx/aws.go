package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/sprucehealth/backend/libs/awsutil"
)

var (
	awsAccessKey = flag.String("aws_access_key", "", "AWS Access Key ID")
	awsSecretKey = flag.String("aws_secret_key", "", "AWS Secret Key")
	awsRole      = flag.String("aws_role", "", "AWS Role")
	awsRegion    = flag.String("aws_region", "", "AWS Region")

	awsConfig *aws.Config
	s3Client  *s3.S3
	cwlClient *cloudwatchlogs.CloudWatchLogs
)

func setupAWS() error {
	var awsConfig *aws.Config

	var creds *credentials.Credentials
	if *awsRole == "" {
		*awsRole = os.Getenv("AWS_ROLE")
	}
	if *awsRole != "" || *awsRole == "*" {
		creds = credentials.NewEC2RoleCredentials(http.DefaultClient, "", time.Minute*10)
	} else if *awsAccessKey != "" && *awsSecretKey != "" {
		creds = credentials.NewStaticCredentials(*awsAccessKey, *awsSecretKey, "")
	} else {
		creds = credentials.NewEnvCredentials()
	}
	if *awsRegion == "" {
		az, err := awsutil.GetMetadata(awsutil.MetadataAvailabilityZone)
		if err != nil {
			return err
		}
		// Remove the last letter of the az to get the region (e.g. us-east-1a -> us-east-1)
		*awsRegion = az[:len(az)-1]
	}

	awsConfig = &aws.Config{
		Credentials: creds,
		Region:      *awsRegion,
	}
	s3Client = s3.New(awsConfig)
	cwlClient = cloudwatchlogs.New(awsConfig)
	return nil
}
