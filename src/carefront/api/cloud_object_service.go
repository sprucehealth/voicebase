package api

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"time"
)

// TODO Need a better way of decentralizing access to different buckets
// such that the cloud service does not have access to all buckets,
// resulting in a security concern
type CloudStorageService struct {
	awsAccessKey string
	awsSecretKey string
}

func NewCloudStorageService(accessKey, secretKey string) *CloudStorageService {
	return &CloudStorageService{accessKey, secretKey}
}

func (c *CloudStorageService) GetObjectAtLocation(bucket, key, region string) (rawData []byte, err error) {
	auth := aws.Auth{c.awsAccessKey, c.awsSecretKey}
	awsRegion, ok := aws.Regions[region]
	if !ok {
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

func (c *CloudStorageService) PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error) {
	objectRecordId, err := dataApi.CreateNewUploadCloudObjectRecord(bucket, key, region)
	if err != nil {
		return 0, "", err
	}

	auth := aws.Auth{c.awsAccessKey, c.awsSecretKey}
	awsRegion, ok := aws.Regions[region]
	if !ok {
		awsRegion = aws.USEast
	}

	s3Access := s3.New(auth, awsRegion)
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
