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
	awsConf := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
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
	id, err := storage.Put("test-1", data, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := storage.Delete(id); err != nil {
			t.Fatal(err)
		}
	}()
	t.Logf("ID: %s", id)
	out, headers, err := storage.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Headers: %+v", headers)
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}

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
