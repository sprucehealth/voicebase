package awsutil

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Config returns an AWS config using either ht peovided credentials, the environment, or ec2 role depending on what's available.
func Config(region, accessKey, secretKey, token string) (*aws.Config, error) {
	var cred *credentials.Credentials
	if accessKey != "" && secretKey != "" {
		cred = credentials.NewStaticCredentials(accessKey, secretKey, "")
	} else {
		cred = credentials.NewEnvCredentials()
		if v, err := cred.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
			cred = ec2rolecreds.NewCredentials(session.New(), func(p *ec2rolecreds.EC2RoleProvider) {
				p.ExpiryWindow = time.Minute * 5
			})
		}
	}
	if region == "" {
		az, err := GetMetadata(MetadataAvailabilityZone)
		if err != nil {
			return nil, fmt.Errorf("no region provided and failed to get from instance metadata: %s", err)
		}
		region = az[:len(az)-1]
	}
	return &aws.Config{
		Credentials: cred,
		Region:      aws.String(region),
	}, nil
}
