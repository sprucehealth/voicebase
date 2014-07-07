package storage

import (
	"bytes"
	"os"
	"testing"

	"github.com/sprucehealth/backend/libs/aws"
)

func TestS3(t *testing.T) {
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("AWS keys not found in environment.")
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("TEST_S3_BUCKET environment variable not set.")
	}

	data := []byte("foo")

	storage := NewS3(keys, "us-east-1", bucket, "/storage-test")
	id, err := storage.Put("test-1", data, nil)
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
}
