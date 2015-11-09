package rxguide

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	rxGuideTableNameFormatString = "%s_rx_guide"
)

var (
	// Limit queries to a 100 results
	limitValue = ptr.Int64(100)

	// Common attribute names
	rxGuidesAN = ptr.String("rx_guides")
	drugNameAN = ptr.String("drug_name")
	rxGuideAN  = ptr.String("rx_guide")

	// KCE represents a KeyConditionExpression
	drugNameBeginsWithKCE = ptr.String("rx_guides = :rx_guides and begins_with(drug_name, :drug_name_prefix)")

	// ErrNoGuidesFound represents that a guide couldn't be found with the provided information
	ErrNoGuidesFound = errors.New("No guides found")
)

// Service describes the methods required to interact with the RXGuide back end
type Service interface {
	PutRXGuide(r *responses.RXGuide) error
	QueryRXGuides(prefix string, limit int) (map[string]*responses.RXGuide, error)
	RXGuide(id string) (*responses.RXGuide, error)
}

// kvs describes the key value store operations that the RXguide service uses
// TODO: Make this interface more generic if we move away from dynamo, currently just used for testing
type kvs interface {
	BatchWriteItem(*dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
	CreateTable(*dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error)
	DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error)
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
}

type service struct {
	rxGuideTableName *string
	kvs              kvs
}

// New returns an initialized instance of service
func New(kvs kvs, env string) (Service, error) {
	s := &service{
		rxGuideTableName: ptr.String(fmt.Sprintf(rxGuideTableNameFormatString, env)),
		kvs:              kvs,
	}
	// Bootstrap the table if it doesn't exist
	return s, errors.Trace(awsutil.CreateDynamoDBTable(kvs, &dynamodb.CreateTableInput{
		TableName: s.rxGuideTableName,
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
		// Note: This table should essentially never recieve a write except when loading. Only invest in reads.
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(50),
			WriteCapacityUnits: ptr.Int64(5),
		},
	}))
}

// QueryRXGuide returns the RX guides that match the provided prefix
func (s *service) RXGuide(drugName string) (*responses.RXGuide, error) {
	getResp, err := s.kvs.GetItem(&dynamodb.GetItemInput{
		TableName: s.rxGuideTableName,
		Key: map[string]*dynamodb.AttributeValue{
			*rxGuidesAN: &dynamodb.AttributeValue{
				S: rxGuidesAN,
			},
			*drugNameAN: &dynamodb.AttributeValue{
				S: ptr.String(strings.ToLower(strings.TrimSpace(drugName))),
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if getResp.Item == nil {
		return nil, ErrNoGuidesFound
	}

	guideData := getResp.Item[*rxGuideAN].B
	if guideData == nil {
		return nil, errors.Trace(fmt.Errorf("No guide data found for matched name %s", drugName))
	}
	guide := &responses.RXGuide{}
	return guide, errors.Trace(json.Unmarshal(guideData, guide))
}

// QueryRXGuide returns the RX guides that match the provided prefix
func (s *service) QueryRXGuides(prefix string, lim int) (map[string]*responses.RXGuide, error) {
	limit := limitValue
	if int64(lim) < *limitValue && lim != 0 {
		limit = ptr.Int64(int64(lim))
	}
	queryResp, err := s.kvs.Query(&dynamodb.QueryInput{
		TableName:              s.rxGuideTableName,
		KeyConditionExpression: drugNameBeginsWithKCE,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rx_guides":        {S: rxGuidesAN},
			":drug_name_prefix": {S: ptr.String(strings.ToLower(strings.TrimSpace(prefix)))},
		},
		Limit: limit,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(queryResp.Items) == 0 {
		return nil, ErrNoGuidesFound
	}

	guides := make(map[string]*responses.RXGuide, len(queryResp.Items))
	for _, guideRecord := range queryResp.Items {
		guide := &responses.RXGuide{}
		if err := json.Unmarshal(guideRecord[*rxGuideAN].B, guide); err != nil {
			return nil, errors.Trace(err)
		}

		// Attempt to find the name of the product in it's correct casing in the guide
		name := *guideRecord[*drugNameAN].S
		if strings.EqualFold(name, guide.GenericName) {
			name = guide.GenericName
		} else {
			for _, brandName := range guide.BrandNames {
				if strings.EqualFold(name, brandName) {
					name = brandName
					break
				}
			}
		}
		guides[name] = guide
	}

	return guides, nil
}

// PutRXGuide maps all the drug variants into the rx_guide table
func (s *service) PutRXGuide(r *responses.RXGuide) error {
	guideData, err := json.Marshal(r)
	if err != nil {
		return errors.Trace(err)
	}

	// Map the guide to all brand and generic forms
	writeRequests := make([]*dynamodb.WriteRequest, len(r.BrandNames)+1)
	for i, brandName := range r.BrandNames {
		writeRequests[i] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					*rxGuidesAN: &dynamodb.AttributeValue{
						S: rxGuidesAN,
					},
					*drugNameAN: &dynamodb.AttributeValue{
						S: ptr.String(strings.ToLower(strings.TrimSpace(brandName))),
					},
					*rxGuideAN: &dynamodb.AttributeValue{
						B: guideData,
					},
				},
			},
		}
	}
	writeRequests[len(r.BrandNames)] = &dynamodb.WriteRequest{
		PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				*rxGuidesAN: &dynamodb.AttributeValue{
					S: rxGuidesAN,
				},
				*drugNameAN: &dynamodb.AttributeValue{
					S: ptr.String(strings.ToLower(strings.TrimSpace(r.GenericName))),
				},
				*rxGuideAN: &dynamodb.AttributeValue{
					B: guideData,
				},
			},
		},
	}

	batchWriteInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			*s.rxGuideTableName: writeRequests,
		},
	}
	_, err = s.kvs.BatchWriteItem(batchWriteInput)
	return errors.Trace(err)
}
