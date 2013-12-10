package kinesis

type Action string

const (
	CreateStream     Action = "CreateStream"
	DeleteStream     Action = "DeleteStream"
	DescribeStream   Action = "DescribeStream"
	GetNextRecords   Action = "GetNextRecords"
	GetShardIterator Action = "GetShardIterator"
	ListStreams      Action = "ListStreams"
	MergeShards      Action = "MergeShards"
	PutRecord        Action = "PutRecord"
	SplitShard       Action = "SplitShard"
)

type CreateStreamRequest struct {
	ShardCount int
	StreamName string
}

type DeleteStreamRequest struct {
	StreamName string
}

type DescribeStreamRequest struct {
	ExclusiveStartShardId string `json:",omitempty"`
	Limit                 int    `json:",omitempty"`
	StreamName            string
}

type DescribeStreamResponse struct {
	StreamDescription *StreamDescription
}

type GetNextRecordsRequest struct {
	Limit         int `json:",omitempty"`
	ShardIterator string
}

type GetNextRecordsResponse struct {
	NextShardIterator string
	Records           []*Record
}

type GetShardIteratorRequest struct {
	ShardId                string
	ShardIteratorType      ShardIteratorType
	StartingSequenceNumber string `json:",omitempty"`
	StreamName             string
}

type GetShardIteratorResponse struct {
	ShardIterator string
}

type ListStreamsRequest struct {
	ExclusiveStartStreamName string `json:",omitempty"`
	Limit                    int    `json:",omitempty"`
}

type ListStreamsResponse struct {
	IsMoreDataAvailable bool
	StreamNames         []string
}

type MergeShardsRequest struct {
	AdjacentShardToMerge string
	ShardToMerge         string
	StreamName           string
}

type PutRecordRequest struct {
	Data                           []byte
	StreamName                     string
	PartitionKey                   string `json:",omitempty"`
	ExplicitHashKey                string `json:",omitempty"`
	ExclusiveMinimumSequenceNumber string `json:",omitempty"`
}

type PutRecordResponse struct {
	SequenceNumber string
	ShardId        string
}

type SplitShardRequest struct {
	NewStartingHashKey string
	ShardToSplit       string
	StreamName         string
}
