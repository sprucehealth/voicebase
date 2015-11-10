package main

import (
	"flag"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sprucehealth/backend/libs/awsutil"
)

var (
	awsAccessKey = flag.String("aws_access_key", "", "AWS Access Key ID")
	awsSecretKey = flag.String("aws_secret_key", "", "AWS Secret Key")
	awsToken     = flag.String("aws_token", "", "AWS Access Token")
	awsRole      = flag.String("aws_role", "", "AWS Role")
	awsRegion    = flag.String("aws_region", "", "AWS Region")

	awsConfig  *aws.Config
	awsSession *session.Session
	s3Client   *s3.S3
	cwlClient  *cloudwatchlogs.CloudWatchLogs
)

func setupAWS() error {
	var creds *credentials.Credentials
	if *awsAccessKey != "" && *awsSecretKey != "" {
		creds = credentials.NewStaticCredentials(*awsAccessKey, *awsSecretKey, *awsToken)
	} else {
		creds = credentials.NewEnvCredentials()
		if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
			creds = ec2rolecreds.NewCredentials(session.New(), func(p *ec2rolecreds.EC2RoleProvider) {
				p.ExpiryWindow = time.Minute * 5
			})
		}
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
		Region:      awsRegion,
	}
	awsSession = session.New(awsConfig)
	s3Client = s3.New(awsSession)
	cwlClient = cloudwatchlogs.New(awsSession)
	return nil
}
