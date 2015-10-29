package mediaproxy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
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
	return d, d.verifyDynamo()
}

// Get implements DAL.Get
func (d *DynamoDBDAL) Get(ids []string) ([]*Media, error) {
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

func (d *DynamoDBDAL) verifyDynamo() error {
	_, err := d.db.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: &d.tableName,
	})

	if err != nil {
		golog.Infof(err.Error())
	}

	if awserr, ok := err.(awserr.Error); ok {
		if awserr.Code() == "ResourceNotFoundException" {
			if err := d.bootstrapDynamo(); err != nil {
				return errors.Trace(err)
			}
		} else {
			return errors.Trace(awserr.OrigErr())
		}
	} else if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *DynamoDBDAL) bootstrapDynamo() error {
	golog.Infof("Bootstrapping mediaproxy dynamo tables...")
	// Create the svc table that maps ids to svc indexed by the ID and view count
	if _, err := d.db.CreateTable(&dynamodb.CreateTableInput{
		TableName: &d.tableName,
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			&dynamodb.AttributeDefinition{
				AttributeName: ptr.String(dynamoIDColumn),
				AttributeType: ptr.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: ptr.String(dynamoIDColumn),
				KeyType:       ptr.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}); err != nil {
		return errors.Trace(err)
	}
	if err := waitForStatus(&dynamoTable{tableName: d.tableName, client: d.db}, awsStatus(`ACTIVE`), time.Second, time.Minute); err != nil {
		return errors.Trace(err)
	}
	return nil
}

type awsStatus string

type awsStatusProvider interface {
	Status() (awsStatus, error)
}

type dynamoTable struct {
	client    dynamodbiface.DynamoDBAPI
	tableName string
}

func (dt *dynamoTable) Status() (awsStatus, error) {
	describeResp, err := dt.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: ptr.String(dt.tableName),
	})
	return awsStatus(*describeResp.Table.TableStatus), err
}

func waitForStatus(provider awsStatusProvider, status awsStatus, delay, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		tStatus, err := provider.Status()
		if err != nil {
			return errors.Trace(err)
		}
		if tStatus == status {
			return nil
		}
		time.Sleep(delay)
	}
	return errors.Trace(fmt.Errorf("Status %s was never reached after waiting %v", status, timeout))
}
