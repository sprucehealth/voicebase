package kinesis

type StreamStatus string

const (
	Creating StreamStatus = "CREATING"
	Deleting StreamStatus = "DELETING"
	Active   StreamStatus = "ACTIVE"
	Updating StreamStatus = "UPDATING"
)

type ShardIteratorType string

const (
	AtSequenceNumber    ShardIteratorType = "AT_SEQUENCE_NUMBER"    // Start reading exactly from the position denoted by a specific sequence number.
	AfterSequenceNumber ShardIteratorType = "AFTER_SEQUENCE_NUMBER" // Start reading right after the position denoted by a specific sequence number.
	TrimHorizon         ShardIteratorType = "TRIM_HORIZON"          // Start reading at the last untrimmed record in the shard in the system, which is the oldest data record in the shard.
	Latest              ShardIteratorType = "LATEST"                // Start reading just after the most recent record in the shard, so that you always read the most recent data in the shard.
)

type HashKeyRange struct {
	EndingHashKey   string
	StartingHashKey string
}

type SequenceNumberRange struct {
	EndingSequenceNumber   string `json:",omitempty"`
	StartingSequenceNumber string
}

type Shard struct {
	AdjacentParentShardId string `json:",omitempty"`
	HashKeyRange          HashKeyRange
	ParentShardId         string `json:",omitempty"`
	SequenceNumberRange   SequenceNumberRange
	ShardId               string
}

type StreamDescription struct {
	IsMoreDataAvailable bool
	Shards              []*Shard
	StreamARN           string
	StreamName          string
	StreamStatus        StreamStatus
}

type Record struct {
	Data           []byte
	PartitionKey   string
	SequenceNumber string
}
