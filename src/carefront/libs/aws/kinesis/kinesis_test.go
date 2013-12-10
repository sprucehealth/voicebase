package kinesis

import (
	"os"
	"testing"

	"carefront/libs/aws"
)

func TestKinesis(t *testing.T) {
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("Skipping aws.kinesis tests. AWS keys not found in environment.")
	}
	stream := os.Getenv("TEST_KINESIS_STREAM")
	if stream == "" {
		t.Skip("Skipping aws.kinesis tests. TEST_KINESIS_STREAM environment variable not set.")
	}

	cli := &aws.Client{
		Auth: keys,
	}
	kin := &Kinesis{
		Region: aws.USEast,
		Client: cli,
	}

	listReq := &ListStreamsRequest{}
	listRes := &ListStreamsResponse{}
	if err := kin.Request(ListStreams, listReq, listRes); err != nil {
		t.Fatal(err)
	}
	if len(listRes.StreamNames) == 0 {
		t.Fatalf("ListStreams returned 0 streams: %+v", listRes)
	}
	t.Logf("ListStreams %+v", listRes)

	partKey := "partKey"
	data := []byte("foo")

	putReq := &PutRecordRequest{
		StreamName:   stream,
		PartitionKey: partKey,
		Data:         data,
	}
	putRes := &PutRecordResponse{}
	if err := kin.Request(PutRecord, putReq, putRes); err != nil {
		t.Fatal(err)
	}
	t.Logf("PutRecord %+v", putRes)
}
