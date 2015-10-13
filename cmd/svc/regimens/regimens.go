package regimens

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	svc "github.com/sprucehealth/backend/svc/regimens"
)

const (
	regimenTableName                = "Regimen"
	regimenTagTableName             = "RegimenTag"
	regimenTagTableTagViewIndexName = "tag_view_count"
)

var (
	up = big.NewInt(math.MaxInt64)
)

// service contains a collections of methods that interact with amazon AWS Dynamo Db to perform the various regimen DAL actions
type service struct {
	dynamoClient dynamodbiface.DynamoDBAPI
	signer       *sig.Signer
}

// New returns an initialized instance of service
func New(d dynamodbiface.DynamoDBAPI, authSecret string) (svc.Service, error) {
	if authSecret == "" {
		return nil, errors.Trace(errors.New("An empty auth secret cannot be used"))
	}
	s := &service{dynamoClient: d}
	var err error
	s.signer, err = sig.NewSigner([][]byte{[]byte(authSecret)}, nil)
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("auth: Failed to initialize auth signer: %s", err))
	}
	return s, errors.Trace(s.verifyDynamo())
}

func (s *service) Regimen(id string) (*svc.Regimen, bool, error) {
	singleIncValue := ptr.String("1")
	updateResp, err := s.dynamoClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: ptr.String(regimenTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"regimen_id": &dynamodb.AttributeValue{
				S: ptr.String(id),
			},
		},
		UpdateExpression:          ptr.String("set view_count = view_count + :inc"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":inc": {N: singleIncValue}},
		ReturnValues:              ptr.String("ALL_NEW"),
	})

	if err != nil {
		return nil, false, errors.Trace(err)
	}

	if updateResp.Attributes == nil {
		return nil, false, errors.Trace(api.ErrNotFound(fmt.Sprintf("Unable to locate regimen with ID with id %s", id)))
	}

	r := &svc.Regimen{}
	if err := json.Unmarshal(updateResp.Attributes["regimen"].B, r); err != nil {
		return nil, false, errors.Trace(err)
	}
	published := updateResp.Attributes["published"].BOOL
	if published == nil {
		published = ptr.Bool(false)
	}

	vc, err := strconv.ParseInt(*updateResp.Attributes["view_count"].N, 10, 64)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	r.ViewCount = int(vc)

	// Asynchronoushly update the tag index table, if any updates fail who cares, log it
	regimenID := *updateResp.Attributes["regimen_id"].S
	conc.Go(func() {
		for _, tag := range r.Tags {
			_, err := s.dynamoClient.UpdateItem(&dynamodb.UpdateItemInput{
				TableName: ptr.String(regimenTagTableName),
				Key: map[string]*dynamodb.AttributeValue{
					"tag": &dynamodb.AttributeValue{
						S: ptr.String(tag),
					},
					"regimen_id": &dynamodb.AttributeValue{
						S: ptr.String(regimenID),
					},
				},
				UpdateExpression:          ptr.String("set view_count = view_count + :inc"),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":inc": {N: singleIncValue}},
			})
			if err != nil {
				golog.Errorf("Error while asynchronously incrementing tag index table view_count - tag: %s, regimen_id: %s", tag, regimenID)
			}
		}
	})

	return r, *published, nil
}

func (s *service) PutRegimen(id string, r *svc.Regimen, published bool) error {
	if r.ID != id {
		return errors.Trace(fmt.Errorf("Cannot insert a regimen with an empty or mismatch ID: expected %q, found %q", id, r.ID))
	}

	if r.URL == "" {
		return errors.Trace(fmt.Errorf("Cannot insert a regimen with an empty URL"))
	}

	regimenData, err := json.Marshal(r)
	if err != nil {
		return errors.Trace(err)
	}

	regimenWriteRequests := []*dynamodb.WriteRequest{
		&dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"regimen_id": &dynamodb.AttributeValue{
						S: ptr.String(id),
					},
					// An unpublished regimen always has a view count of 0 and a published regimen should not be mutated so always PUT with a 0 value
					"view_count": &dynamodb.AttributeValue{
						N: ptr.String("0"),
					},
					"published": &dynamodb.AttributeValue{
						BOOL: ptr.Bool(published),
					},
					"regimen": &dynamodb.AttributeValue{
						B: regimenData,
					},
				},
			},
		},
	}

	// track all the tags we're adding since we can't write duplicates to dynamo
	usedTags := make(map[string]bool)
	tagWriteRequests := make([]*dynamodb.WriteRequest, len(r.Tags))
	for i, tag := range r.Tags {
		if _, ok := usedTags[tag]; !ok {
			usedTags[tag] = true
		} else {
			continue
		}
		tagWriteRequests[i] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"tag": &dynamodb.AttributeValue{
						S: ptr.String(tag),
					},
					// An unpublished regimen always has a view count of 0 and a published regimen should not be mutated so always PUT with a 0 value
					"view_count": &dynamodb.AttributeValue{
						N: ptr.String("0"),
					},
					"regimen_id": &dynamodb.AttributeValue{
						S: ptr.String(id),
					},
				},
			},
		}
	}

	batchWriteInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			regimenTableName: regimenWriteRequests,
		},
	}
	// Only attach tag write requests if there are any
	if len(tagWriteRequests) > 0 {
		batchWriteInput.RequestItems[regimenTagTableName] = tagWriteRequests
	}
	_, err = s.dynamoClient.BatchWriteItem(batchWriteInput)
	return errors.Trace(err)
}

func (s *service) CanAccessResource(resourceID, authToken string) (bool, error) {
	sig, err := base64.StdEncoding.DecodeString(authToken)
	return s.signer.Verify([]byte(resourceID), sig), errors.Trace(err)
}

func (s *service) AuthorizeResource(resourceID string) (string, error) {
	h, err := s.hash(resourceID)
	return h, errors.Trace(err)
}

func (s *service) hash(id string) (string, error) {
	h, err := s.signer.Sign([]byte(id))
	return base64.StdEncoding.EncodeToString(h), errors.Trace(err)
}

type regimenIDViewCount struct {
	regimenID string
	viewCount int
}

type regimenIDViewCountByViewCount []*regimenIDViewCount

func (s regimenIDViewCountByViewCount) Len() int {
	return len(s)
}

func (s regimenIDViewCountByViewCount) Less(i, j int) bool {
	return s[i].viewCount < s[j].viewCount
}

func (s regimenIDViewCountByViewCount) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *service) TagQuery(tags []string) ([]*svc.Regimen, error) {
	regimenIDs := make(map[string]*regimenIDViewCount)
	for _, t := range tags {
		tagRegimenIDs, err := s.dynamoClient.Query(&dynamodb.QueryInput{
			TableName:                 ptr.String(regimenTagTableName),
			IndexName:                 ptr.String(regimenTagTableTagViewIndexName),
			KeyConditionExpression:    ptr.String("tag = :tag"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":tag": {S: ptr.String(t)}},
			// Only return a maximum of 100 records
			Limit: ptr.Int64(100),
			// Order by view count desc
			ScanIndexForward: ptr.Bool(false),
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		// Merge in the result of each query
		for _, v := range tagRegimenIDs.Items {
			vc, err := strconv.ParseInt(*v["view_count"].N, 10, 64)
			if err != nil {
				return nil, errors.Trace(err)
			}
			regimenIDs[*v["regimen_id"].S] = &regimenIDViewCount{
				regimenID: *v["regimen_id"].S,
				viewCount: int(vc),
			}
		}
	}

	if len(regimenIDs) == 0 {
		return nil, nil
	}

	regimenIDVCs := make([]*regimenIDViewCount, len(regimenIDs))
	var i int
	for _, rIDVC := range regimenIDs {
		regimenIDVCs[i] = rIDVC
		i++
	}

	// Sort the ID's so we can take a top set before the fetch
	sort.Sort(sort.Reverse(regimenIDViewCountByViewCount(regimenIDVCs)))
	if len(regimenIDVCs) > 100 {
		regimenIDVCs = regimenIDVCs[:100]
	}
	regimenIDRequests := make([]map[string]*dynamodb.AttributeValue, len(regimenIDVCs))
	for i, rIDVC := range regimenIDVCs {
		regimenIDRequests[i] = map[string]*dynamodb.AttributeValue{
			"regimen_id": {S: ptr.String(rIDVC.regimenID)},
		}
	}

	regimensResp, err := s.dynamoClient.BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			regimenTableName: {Keys: regimenIDRequests},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Only do capacity here since we might have unpublished regimens we need to skip
	rs := make([]*svc.Regimen, 0, len(regimensResp.Responses[regimenTableName]))
	for _, regimen := range regimensResp.Responses[regimenTableName] {
		// skip any unpublished  regimens
		if !(*regimen["published"].BOOL) {
			continue
		}

		r := &svc.Regimen{}
		if err := json.Unmarshal(regimen["regimen"].B, r); err != nil {
			return nil, errors.Trace(err)
		}

		vc, err := strconv.ParseInt(*regimen["view_count"].N, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		r.ViewCount = int(vc)
		rs = append(rs, r)
	}
	sort.Sort(sort.Reverse(svc.ByViewCount(rs)))
	return rs, nil
}

func (s *service) verifyDynamo() error {
	_, err := s.dynamoClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: ptr.String(regimenTableName),
	})

	if err != nil {
		golog.Infof(err.Error())
	}

	if awserr, ok := err.(awserr.Error); ok {
		if awserr.Code() == "ResourceNotFoundException" {
			if err := s.bootstrapDynamo(); err != nil {
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

func (s *service) bootstrapDynamo() error {
	golog.Infof("Bootstrapping dynamo tables...")
	// Create the svc table that maps ids to svc indexed by the ID and view count
	if _, err := s.dynamoClient.CreateTable(&dynamodb.CreateTableInput{
		TableName: ptr.String(regimenTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			&dynamodb.AttributeDefinition{
				AttributeName: ptr.String("regimen_id"),
				AttributeType: ptr.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: ptr.String("regimen_id"),
				KeyType:       ptr.String("HASH"),
			},
		},

		// TODO: Learn about and tune this
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}); err != nil {
		return errors.Trace(err)
	}
	if err := waitForStatus(&dynamoTable{tableName: regimenTableName, client: s.dynamoClient}, awsStatus(`ACTIVE`), time.Second, time.Minute); err != nil {
		return errors.Trace(err)
	}

	// Create the tags table that maps and is indexed by tags to regimen id's
	if _, err := s.dynamoClient.CreateTable(&dynamodb.CreateTableInput{
		TableName: ptr.String(regimenTagTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			&dynamodb.AttributeDefinition{
				AttributeName: ptr.String("tag"),
				AttributeType: ptr.String("S"),
			},
			&dynamodb.AttributeDefinition{
				AttributeName: ptr.String("regimen_id"),
				AttributeType: ptr.String("S"),
			},
			&dynamodb.AttributeDefinition{
				AttributeName: ptr.String("view_count"),
				AttributeType: ptr.String("N"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: ptr.String("tag"),
				KeyType:       ptr.String("HASH"),
			},
			&dynamodb.KeySchemaElement{
				AttributeName: ptr.String("regimen_id"),
				KeyType:       ptr.String("RANGE"),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			&dynamodb.GlobalSecondaryIndex{
				IndexName: ptr.String(regimenTagTableTagViewIndexName),
				KeySchema: []*dynamodb.KeySchemaElement{
					&dynamodb.KeySchemaElement{
						AttributeName: ptr.String("tag"),
						KeyType:       ptr.String("HASH"),
					},
					&dynamodb.KeySchemaElement{
						AttributeName: ptr.String("view_count"),
						KeyType:       ptr.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: ptr.String("ALL"),
				},
				// TODO: Learn about and tune this
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  ptr.Int64(10),
					WriteCapacityUnits: ptr.Int64(10),
				},
			},
		},
		// TODO: Learn about and tune this also
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  ptr.Int64(10),
			WriteCapacityUnits: ptr.Int64(10),
		},
	}); err != nil {
		return errors.Trace(err)
	}
	if err := waitForStatus(&dynamoTable{tableName: regimenTagTableName, client: s.dynamoClient}, awsStatus(`ACTIVE`), time.Second, time.Minute); err != nil {
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
