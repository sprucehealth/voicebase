package api

import (
	"time"

	"carefront/libs/aws"
	"carefront/util"
	goamz "launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

// TODO Need a better way of decentralizing access to different buckets
// such that the cloud service does not have access to all buckets,
// resulting in a security concern
type CloudStorageService struct {
	awsAuth aws.Auth
}

func NewCloudStorageService(awsAuth aws.Auth) *CloudStorageService {
	return &CloudStorageService{awsAuth: awsAuth}
}

func (c *CloudStorageService) GetObjectAtLocation(bucket, key, region string) (rawData []byte, err error) {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(util.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)

	rawData, err = s3Bucket.Get(key)
	if err != nil {
		return nil, err
	}
	return rawData, nil
}

func (c *CloudStorageService) GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error) {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(util.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)
	url = s3Bucket.SignedURL(key, duration)
	return
}

func (c *CloudStorageService) PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error) {
	objectRecordId, err := dataApi.CreateNewUploadCloudObjectRecord(bucket, key, region)
	if err != nil {
		return 0, "", err
	}

	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(util.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)
	additionalHeaders := map[string][]string{
		"x-amz-server-side-encryption": {"AES256"},
	}

	err = s3Bucket.Put(key, rawData, contentType, s3.BucketOwnerFull, additionalHeaders)
	if err != nil {
		return 0, "", err
	}

	dataApi.UpdateCloudObjectRecordToSayCompleted(objectRecordId)
	signedUrl := s3Bucket.SignedURL(key, duration)
	return objectRecordId, signedUrl, nil
}
