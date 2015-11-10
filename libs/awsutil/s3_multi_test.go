package awsutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type nullReader struct{}

func (nullReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func s3Client() *s3.S3 {
	creds := credentials.NewEnvCredentials()
	if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
		creds = ec2rolecreds.NewCredentials(session.New(), func(p *ec2rolecreds.EC2RoleProvider) {
			p.ExpiryWindow = time.Minute * 5
		})
	}
	sess := session.New(&aws.Config{Region: aws.String("us-east-1"), Credentials: creds})
	return s3.New(sess)
}

func TestPutMultiSmall(t *testing.T) {
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("Skipping aws.s3 tests. TEST_S3_BUCKET environment variable not set.")
	}
	s3c := s3Client()

	key := "test-object-1"

	if err := PutMultiFrom(s3c, bucket, key, bytes.NewReader([]byte("testputmulti")), "text/plain", "", ACLPrivate, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	res, err := s3c.GetObject(&s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))

	if _, err := s3c.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key}); err != nil {
		t.Fatal(err)
	}
}

func TestPutMulti(t *testing.T) {
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("Skipping aws.s3 tests. TEST_S3_BUCKET environment variable not set.")
	}
	s3c := s3Client()

	key := "test-object-1"

	size := int64(multiChunkSize + 1024)
	chunker := io.LimitReader(nullReader{}, size)
	if err := PutMultiFrom(s3c, bucket, key, chunker, "text/plain", "", ACLPrivate, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	res, err := s3c.HeadObject(&s3.HeadObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		t.Errorf("HEAD failed: %+v", err)
	}
	t.Logf("%+v", res)

	if n := *res.ContentLength; n != size {
		t.Fatalf("Expected content-length %d got %d", size, n)
	}

	if _, err := s3c.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key}); err != nil {
		t.Fatal(err)
	}
}
