package main

import (
	"errors"
	"flag"
	"os"

	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/cloudwatchlogs"
	"github.com/sprucehealth/backend/libs/aws/s3"
)

var (
	awsAccessKey = flag.String("aws_access_key", "", "AWS Access Key ID")
	awsSecretKey = flag.String("aws_secret_key", "", "AWS Secret Key")
	awsRole      = flag.String("aws_role", "", "AWS Role")
	awsRegion    = flag.String("aws_region", "", "AWS Region")

	region    aws.Region
	awsClient *aws.Client
	s3Client  *s3.S3
	cwlClient *cloudwatchlogs.Client
)

func setupAWS() error {
	var auth aws.Auth

	if *awsRole == "" {
		*awsRole = os.Getenv("AWS_ROLE")
	}
	if *awsRole != "" || *awsRole == "*" {
		var err error
		auth, err = aws.CredentialsForRole(*awsRole)
		if err != nil {
			return err
		}
	} else {
		keys := aws.Keys{
			AccessKey: *awsAccessKey,
			SecretKey: *awsSecretKey,
		}
		if keys.AccessKey == "" || keys.SecretKey == "" {
			keys = aws.KeysFromEnvironment()
		}
		if keys.AccessKey == "" || keys.SecretKey == "" {
			return errors.New("No AWS credentials or role set")
		}
		auth = keys
	}

	if *awsRegion == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			return err
		}
		*awsRegion = az[:len(az)-1]
	}

	var ok bool
	region, ok = aws.Regions[*awsRegion]
	if !ok {
		return errors.New("Unknown region " + *awsRegion)
	}

	awsClient = &aws.Client{
		Auth: auth,
	}

	s3Client = &s3.S3{
		Region: region,
		Client: awsClient,
	}

	cwlClient = &cloudwatchlogs.Client{
		Region: aws.USEast,
		Client: awsClient,
	}

	return nil
}
