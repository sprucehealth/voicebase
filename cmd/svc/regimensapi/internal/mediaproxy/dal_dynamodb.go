package mediaproxy

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	dynamoTableNameFormat = "%s_mediaproxy"
	dynamoIDColumn        = "id"
	dynamoDataColumn      = "data"
)

// DynamoDBDAL uses DynamoDB to store the media metadata
type DynamoDBDAL struct {
	db        dynamodbiface.DynamoDBAPI
	tableName string
}

// NewDynamoDBDAL returns a DAL that uses DynamoDB for storage
func NewDynamoDBDAL(db dynamodbiface.DynamoDBAPI, env string, metricsRegistry metrics.Registry) (*DynamoDBDAL, error) {
	d := &DynamoDBDAL{
		db:        db,
		tableName: fmt.Sprintf(dynamoTableNameFormat, env),
	}
	return d, errors.Trace(awsutil.CreateDynamoDBTable(db, &dynamodb.CreateTableInput{
		TableName: &d.tableName,
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: ptr.String(dynamoIDColumn),
				AttributeType: ptr.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: ptr.String(dynamoIDColumn),
				KeyType:       ptr.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}))
}

// Get implements DAL.Get
func (d *DynamoDBDAL) Get(ids []string) ([]*Media, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]map[string]*dynamodb.AttributeValue, len(ids))
	for i, id := range ids {
		keys[i] = map[string]*dynamodb.AttributeValue{
			dynamoIDColumn: {S: ptr.String(id)},
		}
	}
	res, err := d.db.BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			d.tableName: {
				AttributesToGet: []*string{ptr.String(dynamoDataColumn)},
				ConsistentRead:  ptr.Bool(true),
				Keys:            keys,
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	// TODO: record consumed capacity
	ms := make([]*Media, 0, len(ids))
	for _, r := range res.Responses[d.tableName] {
		m := &Media{}
		if err := json.Unmarshal(r[dynamoDataColumn].B, m); err != nil {
			// TODO: could return partial results which would be a softer failure
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, nil
}

// Put implements DAL.Put
func (d *DynamoDBDAL) Put(ms []*Media) error {
	req := make([]*dynamodb.WriteRequest, len(ms))
	for i, m := range ms {
		b, err := json.Marshal(m)
		if err != nil {
			return errors.Trace(err)
		}
		req[i] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					dynamoIDColumn:   {S: &m.ID},
					dynamoDataColumn: {B: b},
				},
			},
		}
	}
	_, err := d.db.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			d.tableName: req,
		},
	})
	if err != nil {
		return errors.Trace(err)
	}
	// TODO: record consumed capacity
	return nil

}
