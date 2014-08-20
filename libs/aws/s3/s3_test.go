package s3

import (
	"os"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/aws"
)

func TestS3(t *testing.T) {
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("Skipping aws.s3 tests. AWS keys not found in environment.")
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("Skipping aws.s3 tests. TEST_S3_BUCKET environment variable not set.")
	}

	key := "test-object-1"

	cli := &aws.Client{
		Auth: keys,
	}
	s3 := &S3{
		Region: aws.USEast,
		Client: cli,
	}

	if err := s3.Put(bucket, key, []byte("test1"), "text/plain", Private, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	res, err := s3.ListBucket(bucket, &ListBucketParams{MaxKeys: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Contents) < 1 {
		t.Error("Expected at least 1 item in the bucket from ListBucket")
	}

	data, err := s3.Get(bucket, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))

	headers, err := s3.Head(bucket, key)
	if err != nil {
		t.Errorf("HEAD failed: %+v", err)
	}
	t.Logf("%+v", headers)

	_, err = s3.Head(bucket, "non-existant")
	if err == nil {
		t.Errorf("HEAD should have failed")
	}

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}

}
