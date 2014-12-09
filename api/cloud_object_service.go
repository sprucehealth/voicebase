package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	goamz "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/mitchellh/goamz/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/mitchellh/goamz/s3"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws"
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

func (c *CloudStorageService) GetObjectAtLocation(bucket, key, region string) ([]byte, http.Header, error) {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(common.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)

	res, err := s3Bucket.GetResponse(key)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	return data, res.Header, nil
}

func (c *CloudStorageService) DeleteObjectAtLocation(bucket, key, region string) error {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(common.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)
	err := s3Bucket.Del(key)
	return err
}

func (c *CloudStorageService) GetSignedURLForObjectAtLocation(bucket, key, region string, duration time.Time) (string, error) {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Auth := common.AWSAuthAdapter(c.awsAuth)
	s3Access := s3.New(s3Auth, awsRegion)
	s3Bucket := s3Access.Bucket(bucket)
	return s3Bucket.SignedURL(key, duration), nil
}

func (c *CloudStorageService) PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataAPI DataAPI) (int64, string, error) {
	objectRecordID, err := dataAPI.CreateNewUploadCloudObjectRecord(bucket, key, region)
	if err != nil {
		return 0, "", err
	}

	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	s3Access := s3.New(common.AWSAuthAdapter(c.awsAuth), awsRegion)
	s3Bucket := s3Access.Bucket(bucket)
	headers := map[string][]string{
		"x-amz-server-side-encryption": {"AES256"},
		"Content-Type":                 {contentType},
	}

	err = s3Bucket.PutReaderHeader(key, bytes.NewReader(rawData), int64(len(rawData)), headers, s3.BucketOwnerFull)
	if err != nil {
		return 0, "", err
	}
	dataAPI.UpdateCloudObjectRecordToSayCompleted(objectRecordID)
	signedUrl := s3Bucket.SignedURL(key, duration)
	return objectRecordID, signedUrl, nil
}
