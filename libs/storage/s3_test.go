package storage

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sprucehealth/backend/libs/errors"
)

func TestS3(t *testing.T) {
	sess := session.New()
	var creds *credentials.Credentials
	creds = credentials.NewEnvCredentials()
	if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
		creds = ec2rolecreds.NewCredentials(sess, func(p *ec2rolecreds.EC2RoleProvider) {
			p.ExpiryWindow = time.Minute * 5
		})
	}
	awsConf := &aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-1"),
	}
	if _, err := awsConf.Credentials.Get(); err != nil {
		t.Skip(err.Error())
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("TEST_S3_BUCKET environment variable not set.")
	}

	data := []byte("foo")

	sess = sess.Copy(awsConf)
	storage := NewS3(sess, bucket, "/storage-test")

	// Test not existant object
	nonExistantID := "ofiu3j2n90f32u09fnmeuw9"
	_, _, err := storage.Get(nonExistantID)
	if errors.Cause(err) != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}
	_, _, err = storage.GetReader(nonExistantID)
	if errors.Cause(err) != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}

	// Test put
	url, err := storage.Put("test-1", data, "image/tiff", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := storage.Delete("test-1"); err != nil {
			t.Error(err)
		}
	}()

	// Test get on existing object
	out, headers, err := storage.Get("test-1")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Headers: %+v", headers)
	if headers.Get("Content-Type") != "image/tiff" {
		t.Errorf("Expected content-type of image/tiff, got %s", headers.Get("Content-Type"))
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}
	// by URL
	out, headers, err = storage.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Headers: %+v", headers)
	if headers.Get("Content-Type") != "image/tiff" {
		t.Errorf("Expected content-type of image/tiff, got %s", headers.Get("Content-Type"))
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}

	// Copy non-existant object
	if err := storage.Copy("thecopy", nonExistantID); errors.Cause(err) != ErrNoObject {
		t.Errorf("Expected error ErrNoObject got %s", err)
	}

	// Copy object
	dstID := "thecopy"
	if err := storage.Copy(dstID, "test-1"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := storage.Delete(dstID); err != nil {
			t.Error(err)
		}
	}()
	out, headers, err = storage.Get(dstID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Headers: %+v", headers)
	if headers.Get("Content-Type") != "image/tiff" {
		t.Errorf("Expected content-type of image/tiff, got %s", headers.Get("Content-Type"))
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("get %+v but expected %+v", out, data)
	}
}
