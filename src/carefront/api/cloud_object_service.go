package api

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type CloudObjectService struct {
	awsAccessKey string
	awsSecretKey string
}

func NewService(accessKey, secretKey string) *CloudObjectService {
	return &CloudObjectService{accessKey, secretKey}
}
func (c *CloudObjectService) GetObjectAtLocation(bucket, key, region string) (rawData []byte, err error) {
	auth := aws.Auth{c.awsAccessKey, c.awsSecretKey}
	var awsRegion aws.Region
	switch region {
	case "us-east-1":
		awsRegion = aws.USEast
	case "us-west-1":
		awsRegion = aws.USWest
	default:
		awsRegion = aws.USEast
	}

	s3Access := s3.New(auth, awsRegion)
	s3Bucket := s3Access.Bucket(bucket)

	rawData, err = s3Bucket.Get(key)
	if err != nil {
		return nil, err
	}
	return rawData, nil
}
