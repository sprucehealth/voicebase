package s3

import (
	"os"
	"testing"
	"time"

	"carefront/libs/aws"
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

	data, err := s3.Get(bucket, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}
}
