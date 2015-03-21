package kinesis

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/aws"
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

	descReq := &DescribeStreamRequest{
		StreamName: stream,
	}
	descRes := &DescribeStreamResponse{}
	if err := kin.Request(DescribeStream, descReq, descRes); err != nil {
		t.Fatal(err)
	}
	t.Logf("DescribeStream %+v", descRes.StreamDescription)
	shard := descRes.StreamDescription.Shards[0]
	t.Logf("\tShard: %+v", shard)

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

	iterReq := &GetShardIteratorRequest{
		StreamName: stream,
		ShardID:    putRes.ShardID,
		// ShardIteratorType: TrimHorizon,
		ShardIteratorType:      AtSequenceNumber,
		StartingSequenceNumber: putRes.SequenceNumber,
	}
	iterRes := &GetShardIteratorResponse{}
	if err := kin.Request(GetShardIterator, &iterReq, &iterRes); err != nil {
		t.Fatal(err)
	}
	t.Logf("GetShardIterator %+v", iterRes)

	recReq := &GetNextRecordsRequest{
		ShardIterator: iterRes.ShardIterator,
		Limit:         10000,
	}
	recRes := &GetNextRecordsResponse{}
	for i := 0; i < 5; i++ {
		if err := kin.Request(GetNextRecords, &recReq, &recRes); err != nil {
			t.Fatal(err)
		}
		if len(recRes.Records) != 0 {
			break
		}
		recReq.ShardIterator = recRes.NextShardIterator
		time.Sleep(time.Millisecond * 100)
	}
	t.Logf("GetNextRecords %+v", recRes)
	if len(recRes.Records) == 0 {
		t.Fatal("GetNextRecords returned 0 records")
	}
	rec := recRes.Records[0]
	if bytes.Compare(rec.Data, data) != 0 {
		t.Fatalf("Record data did not match. Expected %+v got %+v", data, rec.Data)
	}
}
