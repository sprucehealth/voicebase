package aws

import (
	"os"
	"testing"
	"time"
)

func TestS3(t *testing.T) {
	keys := KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("Skipping aws.s3 tests. AWS keys not found in environment.")
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("Skipping aws.s3 tests. TEST_S3_BUCKET environment variable not set.")
	}

	key := "test-object-1"

	cli := &Client{
		Auth: keys,
	}
	s3 := &S3{
		Region: USEast,
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
