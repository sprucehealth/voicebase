package rxguide

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/test"
)

const testEnv = "test"

func expectTableExists(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
	}))
	// The return value here doesn't matter as the table is only created if this throws and error
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
}

func expectCreateTable(m *mock.DynamoDB) {
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, nil)
	m.DescribeTableErrs = append(m.DescribeTableErrs, awsutil.ErrAWS{CodeF: "ResourceNotFoundException"})
	m.Expect(mock.NewExpectation(m.CreateTable, &dynamodb.CreateTableInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			&dynamodb.AttributeDefinition{
				AttributeName: rxGuidesAN,
				AttributeType: ptr.String("S"),
			},
			&dynamodb.AttributeDefinition{
				AttributeName: drugNameAN,
				AttributeType: ptr.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: rxGuidesAN,
				KeyType:       ptr.String("HASH"),
			},
			&dynamodb.KeySchemaElement{
				AttributeName: drugNameAN,
				KeyType:       ptr.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(50),
			WriteCapacityUnits: ptr.Int64(5),
		},
	}))
	m.CreateTableOutputs = append(m.CreateTableOutputs, nil)

	// Assert that we wait for the created table
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("CREATING")},
	})
	m.Expect(mock.NewExpectation(m.DescribeTable, &dynamodb.DescribeTableInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
	}))
	m.DescribeTableOutputs = append(m.DescribeTableOutputs, &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{TableStatus: ptr.String("ACTIVE")},
	})
}

func TestRXGuideServiceRXGuide(t *testing.T) {
	drugName := "testDrugName"
	rxGuide := &responses.RXGuide{GenericName: drugName}
	data, err := json.Marshal(rxGuide)
	test.OK(t, err)

	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectCreateTable(kvs)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.GetItemInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			*rxGuidesAN: &dynamodb.AttributeValue{
				S: rxGuidesAN,
			},
			*drugNameAN: &dynamodb.AttributeValue{
				S: ptr.String(strings.ToLower(strings.TrimSpace(drugName))),
			},
		},
	}))
	kvs.GetItemOutputs = []*dynamodb.GetItemOutput{
		&dynamodb.GetItemOutput{
			Item: map[string]*dynamodb.AttributeValue{
				*rxGuideAN: {B: data},
			},
		},
	}

	svc, err := New(kvs, testEnv)
	test.OK(t, err)
	guide, err := svc.RXGuide(drugName)
	test.OK(t, err)
	test.Equals(t, rxGuide, guide)
	kvs.Finish()
}

func TestRXGuideServiceRXGuideNoGuidesErr(t *testing.T) {
	drugName := "testDrugName"
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectTableExists(kvs)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.GetItemInput{
		TableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
		Key: map[string]*dynamodb.AttributeValue{
			*rxGuidesAN: &dynamodb.AttributeValue{
				S: rxGuidesAN,
			},
			*drugNameAN: &dynamodb.AttributeValue{
				S: ptr.String(strings.ToLower(strings.TrimSpace(drugName))),
			},
		},
	}))
	kvs.GetItemOutputs = []*dynamodb.GetItemOutput{
		&dynamodb.GetItemOutput{},
	}

	svc, err := New(kvs, testEnv)
	test.OK(t, err)
	_, err = svc.RXGuide(drugName)
	test.Equals(t, ErrNoGuidesFound, err)
	kvs.Finish()
}

func TestRXGuideServiceQueryRXGuides(t *testing.T) {
	drugPrefix := "testDrugName"
	limit := 100
	rxGuide := &responses.RXGuide{GenericName: drugPrefix}
	rxGuide2 := &responses.RXGuide{GenericName: drugPrefix + "2"}
	data, err := json.Marshal(rxGuide)
	test.OK(t, err)
	data2, err := json.Marshal(rxGuide2)
	test.OK(t, err)

	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectTableExists(kvs)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.QueryInput{
		TableName:              ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
		KeyConditionExpression: drugNameBeginsWithKCE,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rx_guides":        {S: rxGuidesAN},
			":drug_name_prefix": {S: ptr.String(strings.ToLower(strings.TrimSpace(drugPrefix)))},
		},
		Limit: ptr.Int64(int64(limit)),
	}))
	kvs.QueryOutputs = []*dynamodb.QueryOutput{
		&dynamodb.QueryOutput{
			Items: []map[string]*dynamodb.AttributeValue{
				map[string]*dynamodb.AttributeValue{
					*rxGuideAN: {B: data},
				},
				map[string]*dynamodb.AttributeValue{
					*rxGuideAN: {B: data2},
				},
			},
		},
	}

	svc, err := New(kvs, testEnv)
	test.OK(t, err)
	guides, err := svc.QueryRXGuides(drugPrefix, limit)
	test.OK(t, err)
	test.Equals(t, []*responses.RXGuide{rxGuide, rxGuide2}, guides)
	kvs.Finish()
}

func TestRXGuideServiceQueryRXGuidesNoGuidesErr(t *testing.T) {
	drugPrefix := "testDrugName"
	limit := 100
	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectTableExists(kvs)
	kvs.Expect(mock.NewExpectation(kvs.GetItem, &dynamodb.QueryInput{
		TableName:              ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, testEnv)),
		KeyConditionExpression: drugNameBeginsWithKCE,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rx_guides":        {S: rxGuidesAN},
			":drug_name_prefix": {S: ptr.String(strings.ToLower(strings.TrimSpace(drugPrefix)))},
		},
		Limit: ptr.Int64(int64(limit)),
	}))
	kvs.QueryOutputs = []*dynamodb.QueryOutput{
		&dynamodb.QueryOutput{
			Items: []map[string]*dynamodb.AttributeValue{},
		},
	}

	svc, err := New(kvs, testEnv)
	test.OK(t, err)
	_, err = svc.QueryRXGuides(drugPrefix, limit)
	test.Equals(t, ErrNoGuidesFound, err)
	kvs.Finish()
}

func TestRXGuideServicePutRXGuide(t *testing.T) {
	genericName := "genericName"
	brandNames := []string{"brandName1", "brandName2"}
	rxGuide := &responses.RXGuide{GenericName: genericName, BrandNames: brandNames}
	data, err := json.Marshal(rxGuide)
	test.OK(t, err)

	kvs := &mock.DynamoDB{Expector: &mock.Expector{T: t}}
	expectTableExists(kvs)
	kvs.Expect(mock.NewExpectation(kvs.BatchWriteItem, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			fmt.Sprintf(rxGuideTableNameFormatString, testEnv): []*dynamodb.WriteRequest{
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							*rxGuidesAN: &dynamodb.AttributeValue{
								S: rxGuidesAN,
							},
							*drugNameAN: &dynamodb.AttributeValue{
								S: ptr.String(strings.ToLower(strings.TrimSpace(brandNames[0]))),
							},
							*rxGuideAN: &dynamodb.AttributeValue{
								B: data,
							},
						},
					},
				},
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							*rxGuidesAN: &dynamodb.AttributeValue{
								S: rxGuidesAN,
							},
							*drugNameAN: &dynamodb.AttributeValue{
								S: ptr.String(strings.ToLower(strings.TrimSpace(brandNames[1]))),
							},
							*rxGuideAN: &dynamodb.AttributeValue{
								B: data,
							},
						},
					},
				},
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							*rxGuidesAN: &dynamodb.AttributeValue{
								S: rxGuidesAN,
							},
							*drugNameAN: &dynamodb.AttributeValue{
								S: ptr.String(strings.ToLower(strings.TrimSpace(genericName))),
							},
							*rxGuideAN: &dynamodb.AttributeValue{
								B: data,
							},
						},
					},
				},
			},
		}}))
	kvs.BatchWriteItemOutputs = []*dynamodb.BatchWriteItemOutput{nil}
	svc, err := New(kvs, testEnv)
	test.OK(t, err)
	test.OK(t, svc.PutRXGuide(rxGuide))
	kvs.Finish()
}
