package awsutil

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

type dynamoTableDescriber interface {
	DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error)
}

type dynamoTableCreator interface {
	dynamoTableDescriber
	CreateTable(input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error)
}

type dynamoTable struct {
	client    dynamoTableDescriber
	tableName string
}

func (dt *dynamoTable) Status() (awsStatus, error) {
	describeResp, err := dt.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: ptr.String(dt.tableName),
	})
	return awsStatus(*describeResp.Table.TableStatus), err
}

// CreateDynamoDBTable creates the described table and waits for it's creation to complete. If the table exists already then nothing is created
// TODO: In the future it would be nice if this not only checked the table for existance but also that the schema matches the expected before continueing
func CreateDynamoDBTable(tc dynamoTableCreator, input *dynamodb.CreateTableInput) error {
	_, err := tc.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: input.TableName,
	})
	if err != nil {
		golog.Infof(err.Error())
	}

	if awserr, ok := err.(awserr.Error); ok {
		if awserr.Code() == "ResourceNotFoundException" {
			if err := createTable(tc, input); err != nil {
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

func createTable(tc dynamoTableCreator, input *dynamodb.CreateTableInput) error {
	golog.Infof("Creating DynamoTable %s...", *input.TableName)
	if _, err := tc.CreateTable(input); err != nil {
		return errors.Trace(err)
	}
	if err := waitForStatus(&dynamoTable{tableName: *input.TableName, client: tc}, awsStatus(`ACTIVE`), time.Second, time.Minute); err != nil {
		return errors.Trace(err)
	}
	return nil
}
