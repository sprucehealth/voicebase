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
	_, _, err := storage.Get("s3://us-east-1/" + bucket + "/storage-test/ofiu3j2n90f32u09fnmeuw9")
	if err != ErrNoObject {
		t.Fatalf("Expected ErrNoObject got %T %+v", err, err)
	}
	_, _, err = storage.GetReader("s3://us-east-1/" + bucket + "/storage-test/ofiu3j2n90f32u09fnmeuw9")
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
}
