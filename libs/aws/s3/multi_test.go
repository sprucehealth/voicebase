package s3

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/aws"
)

type nullReader struct{}

func (nullReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func TestMultiOneChunk(t *testing.T) {
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

	headers := map[string][]string{"x-amz-server-side-encryption": []string{"AES256"}}
	multi, err := s3.InitMulti(bucket, key, "text/plain", Private, headers)
	if err != nil {
		t.Fatal(err)
	}

	part, err := multi.PutPartFrom(1, bytes.NewReader([]byte("testmulti")), 9)
	if err != nil {
		if err := multi.Abort(); err != nil {
			t.Error(err.Error())
		}
		t.Fatal(err)
	}

	if err := multi.Complete([]Part{part}); err != nil {
		if err := multi.Abort(); err != nil {
			t.Error(err.Error())
		}
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	data, err := s3.Get(bucket, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))

	headers, err = s3.Head(bucket, key)
	if err != nil {
		t.Errorf("HEAD failed: %+v", err)
	}
	t.Logf("%+v", headers)

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}
}

func TestMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test when running short tests only")
	}
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

	headers := map[string][]string{"x-amz-server-side-encryption": []string{"AES256"}}
	multi, err := s3.InitMulti(bucket, key, "text/plain", Private, headers)
	if err != nil {
		t.Fatal(err)
	}

	chunker := io.LimitReader(nullReader{}, multiChunkSize)

	var parts []Part

	part, err := multi.PutPartFrom(1, chunker, multiChunkSize)
	if err != nil {
		if err := multi.Abort(); err != nil {
			t.Error(err.Error())
		}
		t.Fatal(err)
	}
	parts = append(parts, part)

	part, err = multi.PutPartFrom(2, bytes.NewReader([]byte("testmulti")), 9)
	if err != nil {
		if err := multi.Abort(); err != nil {
			t.Error(err.Error())
		}
		t.Fatal(err)
	}
	parts = append(parts, part)

	if err := multi.Complete(parts); err != nil {
		if err := multi.Abort(); err != nil {
			t.Error(err.Error())
		}
		t.Fatal(err)
	}

	headers, err = s3.Head(bucket, key)
	if err != nil {
		t.Errorf("HEAD failed: %+v", err)
	}
	t.Logf("%+v", headers)

	if n, err := strconv.ParseInt(headers["Content-Length"][0], 10, 64); err != nil {
		t.Fatal(err)
	} else if size := int64(multiChunkSize + 9); n != size {
		t.Fatalf("Expected content-length %d got %d", size, n)
	}

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}
}

func TestPutMultiSmall(t *testing.T) {
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

	if err := s3.PutMultiFrom(bucket, key, bytes.NewReader([]byte("testputmulti")), "text/plain", Private, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

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

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}
}

func TestPutMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test when running short tests only")
	}
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

	size := int64(multiChunkSize + 1024)
	chunker := io.LimitReader(nullReader{}, size)
	if err := s3.PutMultiFrom(bucket, key, chunker, "text/plain", Private, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	headers, err := s3.Head(bucket, key)
	if err != nil {
		t.Errorf("HEAD failed: %+v", err)
	}
	t.Logf("%+v", headers)

	if n, err := strconv.ParseInt(headers.Get("Content-Length"), 10, 64); err != nil {
		t.Fatal(err)
	} else if n != size {
		t.Fatalf("Expected content-length %d got %d", size, n)
	}

	if err := s3.Delete(bucket, key); err != nil {
		t.Fatal(err)
	}
}
