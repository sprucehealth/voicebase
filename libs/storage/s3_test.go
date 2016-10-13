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
	nonExistantID := "s3://us-east-1/" + bucket + "/storage-test/ofiu3j2n90f32u09fnmeuw9"
	_, _, err := storage.Get(nonExistantID)
	if err != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}
	_, _, err = storage.GetReader(nonExistantID)
	if err != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}

	// Test put
	id, err := storage.Put("test-1", data, "image/tiff", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %s", id)
	defer func() {
		if err := storage.Delete(id); err != nil {
			t.Error(err)
		}
	}()

	// Test get on existing object
	out, headers, err := storage.Get(id)
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
	dstID := storage.IDFromName("thecopy")
	if err := storage.Copy(dstID, nonExistantID); err != ErrNoObject {
		t.Errorf("Expected error ErrNoObject got %s", err)
	}

	// Copy object
	if err := storage.Copy(dstID, id); err != nil {
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
