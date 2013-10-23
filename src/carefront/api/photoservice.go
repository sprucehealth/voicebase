package api

import (
	"bytes"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"time"
)

type PhotoService struct {
	AWSAccessKey string
	AWSSecretKey string
}

func (p *PhotoService) Upload(data []byte, contentType string, key string, bucket string, duration time.Time) (string, error) {
	auth := aws.Auth{p.AWSAccessKey, p.AWSSecretKey}

	s3Access := s3.New(auth, aws.USWest)
	s3Bucket := s3Access.Bucket(bucket)

	additionalHeaders := map[string][]string{
		"x-amz-server-side-encryption": {"AES256"},
	}

	if err := s3Bucket.Put(key, data, contentType, s3.BucketOwnerFull, additionalHeaders); err != nil {
		return "", err
	}

	return s3Bucket.SignedURL(key, duration), nil
}

func (p *PhotoService) GenerateSignedUrlsForKeysInBucket(bucket, prefix string, duration time.Time) ([]string, error) {
	auth := aws.Auth{p.AWSAccessKey, p.AWSSecretKey}
	s3Access := s3.New(auth, aws.USWest)
	s3Bucket := s3Access.Bucket(bucket)

	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString("/")
	listBucketResult, err := s3Bucket.List(buffer.String(), "/", "", 100)
	if err != nil {
		return nil, err
	}

	signedUrls := make([]string, len(listBucketResult.Contents))
	for i, v := range listBucketResult.Contents {
		signedUrls[i] = s3Bucket.SignedURL(v.Key, duration)
	}

	return signedUrls, nil
}
