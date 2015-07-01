package storage

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
)

func TestS3(t *testing.T) {
	var creds *credentials.Credentials
	creds = credentials.NewEnvCredentials()
	if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
		creds = credentials.NewEC2RoleCredentials(&http.Client{Timeout: 2 * time.Second}, "", time.Minute*5)
	}
	awsConf := &aws.Config{
		Credentials: creds,
		Region:      "us-east-1",
	}
	if _, err := awsConf.Credentials.Get(); err != nil {
		t.Skip(err.Error())
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("TEST_S3_BUCKET environment variable not set.")
	}

	data := []byte("foo")

	storage := NewS3(awsConf, bucket, "/storage-test")

	// Test not existant object
	_, _, err := storage.Get("s3://us-east-1/" + bucket + "/storage-test/ofiu3j2n90f32u09fnmeuw9")
	if err != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}

	// Test put
	id, err := storage.Put("test-1", data, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %s", id)
	defer func() {
		if err := storage.Delete(id); err != nil {
			t.Fatal(err)
		}
	}()

	// Test get on existing object
	out, headers, err := storage.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Headers: %+v", headers)
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}

	// Test signed URLs
	url, err := storage.SignedURL(id, time.Minute*10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("URL: %s", url)
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}
}
