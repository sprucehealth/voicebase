package aws

// import (
// 	"testing"
// 	"time"
// )

// func TestS3(t *testing.T) {
// 	keys := KeysFromEnvironment()
// 	if keys.AccessKey == "" || keys.SecretKey == "" {
// 		t.Logf("Skipping aws.s3 tests. AWS keys not found in environment")
// 		return
// 	}

// 	bucket := "carefront-config-useast1"
// 	key := "test-object-1"

// 	cli := &Client{
// 		Auth: keys,
// 	}
// 	s3 := &S3{
// 		Region: USEast,
// 		Client: cli,
// 	}

// 	if err := s3.Put(bucket, key, []byte("test1"), "text/plain", Private, nil); err != nil {
// 		t.Fatal(err)
// 	}

// 	time.Sleep(time.Millisecond * 100)

// 	data, err := s3.Get(bucket, key)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Log(string(data))

// 	if err := s3.Delete(bucket, key); err != nil {
// 		t.Fatal(err)
// 	}
// }
