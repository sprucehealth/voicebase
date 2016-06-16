package regimens

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	rsvc "github.com/sprucehealth/backend/svc/regimens"
	"github.com/sprucehealth/backend/test"
)

const (
	testEnv    = "test"
	testSecret = "seekrit"
)

func expectRegimensTableExists(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
	}))
	// The return value here doesn't matter as the table is only created if this throws and error
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
}

func expectRegimensCreateTable(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
	m.DescribeTableErrs = append(m.DescribeTableErrs, awsutil.ErrAWS{CodeF: "ResourceNotFoundException", MessageF: "Initial Describe Regimen"})
	m.Expect(mock.NewExpectation(m.CreateTable, &dynamodb.CreateTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: ptr.String(regimenIDAN),
				AttributeType: ptr.String("S"),
			},
			{
				AttributeName: ptr.String(sourceRegimenIDAN),
				AttributeType: ptr.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: ptr.String(regimenIDAN),
				KeyType:       ptr.String("HASH"),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: regimenTableSourceRegimenRegimenIndexName,
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: ptr.String(sourceRegimenIDAN),
						KeyType:       ptr.String("HASH"),
					},
					{
						AttributeName: ptr.String(regimenIDAN),
						KeyType:       ptr.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: ptr.String("INCLUDE"),
					NonKeyAttributes: []*string{
						ptr.String(publishedAN),
						ptr.String(viewCountAN),
						ptr.String(regimenAN),
					},
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  ptr.Int64(10),
					WriteCapacityUnits: ptr.Int64(10),
				},
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}))
	m.CreateTableOutputs = append(m.CreateTableOutputs, nil)

	// Assert that we wait for the created table
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("CREATING")},
	})
	m.DescribeTableErrs = append(m.DescribeTableErrs, nil)
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("ACTIVE")},
	})
	m.DescribeTableErrs = append(m.DescribeTableErrs, nil)
}

func expectRegimensTagTableExists(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
	}))
	// The return value here doesn't matter as the table is only created if this throws and error
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
}

func expectRegimensTagCreateTable(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
	m.DescribeTableErrs = append(m.DescribeTableErrs, awsutil.ErrAWS{CodeF: "ResourceNotFoundException", MessageF: "Initial Describe Tag"})
	m.Expect(mock.NewExpectation(m.CreateTable, &dynamodb.CreateTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: ptr.String(tagAN),
				AttributeType: ptr.String("S"),
			},
			{
				AttributeName: ptr.String(regimenIDAN),
				AttributeType: ptr.String("S"),
			},
			{
				AttributeName: ptr.String(viewCountAN),
				AttributeType: ptr.String("N"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: ptr.String(tagAN),
				KeyType:       ptr.String("HASH"),
			},
			{
				AttributeName: ptr.String(regimenIDAN),
				KeyType:       ptr.String("RANGE"),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: regimenTagTableTagViewIndexName,
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: ptr.String(tagAN),
						KeyType:       ptr.String("HASH"),
					},
					{
						AttributeName: ptr.String(viewCountAN),
						KeyType:       ptr.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: ptr.String("ALL"),
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  ptr.Int64(10),
					WriteCapacityUnits: ptr.Int64(10),
				},
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}))
	m.CreateTableOutputs = append(m.CreateTableOutputs, nil)

	// Assert that we wait for the created table
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("CREATING")},
	})
	m.DescribeTableErrs = append(m.DescribeTableErrs, nil)
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("ACTIVE")},
	})
	m.DescribeTableErrs = append(m.DescribeTableErrs, nil)
}

func TestRegimensServiceTableCreation(t *testing.T) {
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensCreateTable(kvs)
	expectRegimensTagCreateTable(kvs)
	_, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)
	mock.FinishAll(publisher, kvs)
}

func TestRegimensServiceRegimen(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)

	// We expect to fetch the item and then do some formatting and return
	regimen := &rsvc.Regimen{}
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.GetItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.GetItemOutputs = append(kvs.GetItemOutputs, &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			regimenAN:   {B: data},
			publishedAN: {BOOL: ptr.Bool(true)},
			viewCountAN: {N: ptr.String("99")},
		},
	})

	reg, published, err := svc.Regimen(id)
	test.OK(t, err)
	regimen.ViewCount = 99
	test.Equals(t, regimen, reg)
	test.Equals(t, true, published)
	mock.FinishAll(publisher, kvs)
}

func TestRegimensServiceRegimenUnknownError(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.GetItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.GetItemOutputs = append(kvs.GetItemOutputs, nil)
	kvs.GetItemErrs = append(kvs.GetItemErrs, errors.New("Random error"))

	reg, published, err := svc.Regimen(id)
	test.Equals(t, (*rsvc.Regimen)(nil), reg)
	test.Equals(t, false, published)
	test.Assert(t, err != nil, "Expected unhandled error")
	mock.FinishAll(publisher, kvs)
}

func TestRegimensServiceRegimenNilPublishedDefault(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)

	regimen := &rsvc.Regimen{}
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.GetItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.GetItemOutputs = append(kvs.GetItemOutputs, &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			regimenAN:   {B: data},
			publishedAN: {BOOL: nil},
			viewCountAN: {N: ptr.String("99")},
		},
	})

	_, published, err := svc.Regimen(id)
	test.OK(t, err)
	test.Equals(t, false, published)
	mock.FinishAll(publisher, kvs)
}

func TestRegimensIncrementViewCount(t *testing.T) {
	id := "myRegimenID"
	tags := []string{"Tag1", "Tag2"}
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)

	regimen := &rsvc.Regimen{Tags: tags}
	data, err := json.Marshal(regimen)
	test.OK(t, err)

	// We expect to update the view count on the regimens table then the tag index
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
		ReturnConsumedCapacity:    indexesRCC,
		ReturnValues:              allNewRV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, &dynamodb.UpdateItemOutput{
		Attributes: map[string]*dynamodb.AttributeValue{
			regimenAN: {B: data},
		},
	})
	// Tag updates, nil responses since they aren't considered
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			tagAN: {
				S: ptr.String(strings.ToLower(tags[0])),
			},
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, nil)
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			tagAN: {
				S: ptr.String(strings.ToLower(tags[1])),
			},
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, nil)
	test.OK(t, svc.IncrementViewCount(id))
	mock.FinishAll(publisher, kvs)
}

func TestRegimensIncrementViewCountIgnoreIndexUpdateErrors(t *testing.T) {
	id := "myRegimenID"
	tags := []string{"Tag1", "Tag2"}
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)

	regimen := &rsvc.Regimen{Tags: tags}
	data, err := json.Marshal(regimen)
	test.OK(t, err)

	// We expect to update the view count on the regimens table then the tag index
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
		ReturnConsumedCapacity:    indexesRCC,
		ReturnValues:              allNewRV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, &dynamodb.UpdateItemOutput{
		Attributes: map[string]*dynamodb.AttributeValue{
			regimenAN: {B: data},
		},
	})
	kvs.UpdateItemErrs = append(kvs.UpdateItemErrs, nil)
	// Tag updates, nil responses since they aren't considered
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			tagAN: {
				S: ptr.String(strings.ToLower(tags[0])),
			},
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, nil)
	kvs.UpdateItemErrs = append(kvs.UpdateItemErrs, errors.New("ExpectedErr: Failed to update index"))
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			tagAN: {
				S: ptr.String(strings.ToLower(tags[1])),
			},
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, nil)
	kvs.UpdateItemErrs = append(kvs.UpdateItemErrs, errors.New("ExpectedErr: Failed to update index"))
	test.OK(t, svc.IncrementViewCount(id))
	mock.FinishAll(publisher, kvs)
}

func TestRegimensIncrementViewCountNoRegimen(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)

	// We expect to update the view count on the regimens table then the tag index
	kvs.Expect(mock.NewExpectation(kvs.UpdateItem, &dynamodb.UpdateItemInput{
		TableName: ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
		ReturnConsumedCapacity:    indexesRCC,
		ReturnValues:              allNewRV,
	}))
	kvs.UpdateItemOutputs = append(kvs.UpdateItemOutputs, &dynamodb.UpdateItemOutput{})
	test.Assert(t, api.IsErrNotFound(svc.IncrementViewCount(id)), "Expected ErrNotFound")
	mock.FinishAll(publisher, kvs)
}

func TestRegimensPutRegimenUnpublished(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	published := false
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)
	regimen := &rsvc.Regimen{ID: id + "wrong"}
	test.Assert(t, svc.PutRegimen(id, regimen, published) != nil, "Expected an error with mismatch IDs")
	regimen.ID = id
	test.Assert(t, svc.PutRegimen(id, regimen, published) != nil, "Expected an error with empty URL")
	regimen.URL = "myURL"
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.BatchWriteItem, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			fmt.Sprintf(regimenTableNameFormatString, testEnv): {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							regimenIDAN: {
								S: ptr.String(id),
							},
							viewCountAN: {
								N: zeroAV,
							},
							publishedAN: {
								BOOL: ptr.Bool(published),
							},
							regimenAN: {
								B: data,
							},
						},
					},
				},
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.BatchWriteItemOutputs = append(kvs.BatchWriteItemOutputs, nil)
	test.OK(t, svc.PutRegimen(id, regimen, published))
	mock.FinishAll(publisher, kvs)
}

func TestRegimensPutRegimenPublished(t *testing.T) {
	id := "myRegimenID"
	// We should ignore the duplicate tag
	tags := []string{"Tag1", "Tag2", "Tag2"}
	publisher := &mock.Publisher{}
	published := true
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)
	regimen := &rsvc.Regimen{ID: id, URL: "myURL", Tags: tags}
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.BatchWriteItem, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			fmt.Sprintf(regimenTableNameFormatString, testEnv): {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							regimenIDAN: {
								S: ptr.String(id),
							},
							viewCountAN: {
								N: zeroAV,
							},
							publishedAN: {
								BOOL: ptr.Bool(published),
							},
							regimenAN: {
								B: data,
							},
						},
					},
				},
			},
			fmt.Sprintf(regimenTagTableNameFormatString, testEnv): {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							tagAN: {
								S: ptr.String(strings.ToLower(tags[0])),
							},
							// An unpublished regimen always has a view count of 0 and a published regimen should not be mutated so always PUT with a 0 value
							viewCountAN: {
								N: zeroAV,
							},
							regimenIDAN: {
								S: ptr.String(id),
							},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							tagAN: {
								S: ptr.String(strings.ToLower(tags[1])),
							},
							// An unpublished regimen always has a view count of 0 and a published regimen should not be mutated so always PUT with a 0 value
							viewCountAN: {
								N: zeroAV,
							},
							regimenIDAN: {
								S: ptr.String(id),
							},
						},
					},
				},
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.BatchWriteItemOutputs = append(kvs.BatchWriteItemOutputs, nil)
	test.OK(t, svc.PutRegimen(id, regimen, published))
	mock.FinishAll(publisher, kvs)
}

func TestRegimensResourceAuthorization(t *testing.T) {
	id := "myRegimenID"
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	token, err := svc.AuthorizeResource(id)
	test.OK(t, err)
	ok, err := svc.CanAccessResource(id, token)
	test.OK(t, err)
	test.Assert(t, ok, "Expected resource to be authorized")
	mock.FinishAll(publisher, kvs)
}

func TestRegimensTagQuery(t *testing.T) {
	id := "myRegimenID"
	id2 := "myRegimenID2"
	// We should ignore the duplicate tag
	tags := []string{"Tag1", "Tag2"}
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	test.OK(t, err)
	regimen := &rsvc.Regimen{ID: id, URL: "myURL", Tags: tags}
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	regimen2 := &rsvc.Regimen{ID: id2, URL: "myURL", Tags: tags}
	data2, err := json.Marshal(regimen2)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.Query, &dynamodb.QueryInput{
		TableName:                 ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		IndexName:                 regimenTagTableTagViewIndexName,
		KeyConditionExpression:    tagEqualsTagKCE,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":tag": {S: ptr.String(strings.ToLower(tags[0]))}},
		Limit:                  limitValue,
		ScanIndexForward:       tagQueryScanIndexDirection,
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.QueryOutputs = append(kvs.QueryOutputs, &dynamodb.QueryOutput{
		Items: []map[string]*dynamodb.AttributeValue{
			{viewCountAN: {N: ptr.String("49")}, regimenIDAN: {S: ptr.String(id)}},
		},
	})
	kvs.Expect(mock.NewExpectation(kvs.Query, &dynamodb.QueryInput{
		TableName:                 ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, testEnv)),
		IndexName:                 regimenTagTableTagViewIndexName,
		KeyConditionExpression:    tagEqualsTagKCE,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":tag": {S: ptr.String(strings.ToLower(tags[1]))}},
		Limit:                  limitValue,
		ScanIndexForward:       tagQueryScanIndexDirection,
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.QueryOutputs = append(kvs.QueryOutputs, &dynamodb.QueryOutput{
		Items: []map[string]*dynamodb.AttributeValue{
			{viewCountAN: {N: ptr.String("50")}, regimenIDAN: {S: ptr.String(id2)}},
		},
	})
	kvs.Expect(mock.NewExpectation(kvs.BatchGetItem, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			fmt.Sprintf(regimenTableNameFormatString, testEnv): {
				Keys: []map[string]*dynamodb.AttributeValue{
					{regimenIDAN: {S: ptr.String(id2)}},
					{regimenIDAN: {S: ptr.String(id)}},
				},
			},
		},
	}))
	kvs.BatchGetItemOutputs = append(kvs.BatchGetItemOutputs, &dynamodb.BatchGetItemOutput{
		Responses: map[string][]map[string]*dynamodb.AttributeValue{
			fmt.Sprintf(regimenTableNameFormatString, testEnv): {
				{
					regimenAN:   {B: data},
					viewCountAN: {N: ptr.String("49")},
				},
				{
					regimenAN:   {B: data2},
					viewCountAN: {N: ptr.String("50")},
				},
			},
		},
	})

	// Asser the ordering of our results
	regs, err := svc.TagQuery(tags)
	test.OK(t, err)
	test.Equals(t, 2, len(regs))
	regimen2.ViewCount = 50
	regimen.ViewCount = 49
	test.Equals(t, []*rsvc.Regimen{regimen2, regimen}, regs)
	mock.FinishAll(publisher, kvs)
}

func TestRegimensFoundationOf(t *testing.T) {
	id := "myRegimenID"
	id2 := "myRegimenID2"
	limit := 5
	publisher := &mock.Publisher{}
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectRegimensTableExists(kvs)
	expectRegimensTagTableExists(kvs)
	svc, err := New(kvs, publisher, testEnv, testSecret)
	regimen := &rsvc.Regimen{ID: id, URL: "myURL"}
	data, err := json.Marshal(regimen)
	test.OK(t, err)
	regimen2 := &rsvc.Regimen{ID: id2, URL: "myURL"}
	data2, err := json.Marshal(regimen2)
	test.OK(t, err)
	kvs.Expect(mock.NewExpectation(kvs.Query, &dynamodb.QueryInput{
		TableName:                 ptr.String(fmt.Sprintf(regimenTableNameFormatString, testEnv)),
		IndexName:                 regimenTableSourceRegimenRegimenIndexName,
		KeyConditionExpression:    sourceRegimenIDEqualsSourceRegimenIDKCE,
		FilterExpression:          ptr.String("published = :published"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":source_regimen_id": {S: ptr.String(id)}, ":published": {BOOL: publishedFilter}},
		Limit: limitValue,
		ReturnConsumedCapacity: indexesRCC,
	}))
	kvs.QueryOutputs = append(kvs.QueryOutputs, &dynamodb.QueryOutput{
		Items: []map[string]*dynamodb.AttributeValue{
			{viewCountAN: {N: ptr.String("49")}, regimenAN: {B: data}, regimenIDAN: {S: ptr.String(id)}},
			{viewCountAN: {N: ptr.String("50")}, regimenAN: {B: data2}, regimenIDAN: {S: ptr.String(id2)}},
		},
	})

	// Assert the ordering of our results
	regs, err := svc.FoundationOf(id, limit)
	test.OK(t, err)
	test.Equals(t, 2, len(regs))
	regimen2.ViewCount = 50
	regimen.ViewCount = 49
	test.Equals(t, []*rsvc.Regimen{regimen2, regimen}, regs)
	mock.FinishAll(publisher, kvs)
}
