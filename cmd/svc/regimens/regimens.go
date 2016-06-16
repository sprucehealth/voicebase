package regimens

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	svc "github.com/sprucehealth/backend/svc/regimens"
)

const (
	regimenTableNameFormatString    = "%s_regimen"
	regimenTagTableNameFormatString = "%s_regimen_tag"

	// AN represents "Attribute Name"
	regimenIDAN       = "regimen_id"
	sourceRegimenIDAN = "source_regimen_id"
	publishedAN       = "published"
	viewCountAN       = "view_count"
	regimenAN         = "regimen"
	tagAN             = "tag"
)

var (
	// Preallocate strings and pointers to common objects used in the dynamo tables
	// tag_view_count is the GSI used to sort tag results by view count
	regimenTagTableTagViewIndexName = ptr.String("tag_view_count")
	// source_regimen_regimen is the GSI used to sort tag results by view count
	regimenTableSourceRegimenRegimenIndexName = ptr.String("source_regimen_regimen")
	// A false parameter for Scan Index Direction returns descending order
	tagQueryScanIndexDirection = ptr.Bool(false)
	publishedFilter            = ptr.Bool(true)
	// Limit queries to a 100 results
	limitValue = ptr.Int64(100)
	// AV represents an AttributeValue
	oneAV  = ptr.String("1")
	zeroAV = ptr.String("0")
	// RV represents a ReturnValues option
	allNewRV = ptr.String("ALL_NEW")
	// UE represents an UpdateExpression
	incrementViewCountUE = ptr.String("set view_count = view_count + :inc")
	// UEAV represents an UpdateExpressionAttributeValues
	incrementSingleValueUEAV = map[string]*dynamodb.AttributeValue{":inc": {N: oneAV}}
	// KCE represents a KeyConditionExpression
	tagEqualsTagKCE                         = ptr.String("tag = :tag")
	sourceRegimenIDEqualsSourceRegimenIDKCE = ptr.String("source_regimen_id = :source_regimen_id")
	// RCC represents a ReturnConsumedCapacity value
	indexesRCC = ptr.String("INDEXES")
)

// TODO: Make this more generic if we move away from dynamo
type kvs interface {
	BatchGetItem(*dynamodb.BatchGetItemInput) (*dynamodb.BatchGetItemOutput, error)
	BatchWriteItem(*dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
	CreateTable(*dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error)
	DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error)
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// service contains a collections of methods that interact with amazon AWS Dynamo Db to perform the various regimen DAL actions
type service struct {
	publisher           dispatch.Publisher
	kvs                 kvs
	signer              *sig.Signer
	regimenTableName    *string
	regimenTagTableName *string
}

// New returns an initialized instance of service
func New(kvs kvs, publisher dispatch.Publisher, env, authSecret string) (svc.Service, error) {
	if authSecret == "" {
		return nil, errors.Trace(errors.New("An empty auth secret cannot be used"))
	}
	signer, err := sig.NewSigner([][]byte{[]byte(authSecret)}, nil)
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("auth: Failed to initialize auth signer: %s", err))
	}

	s := &service{
		publisher:           publisher,
		kvs:                 kvs,
		signer:              signer,
		regimenTableName:    ptr.String(fmt.Sprintf(regimenTableNameFormatString, env)),
		regimenTagTableName: ptr.String(fmt.Sprintf(regimenTagTableNameFormatString, env)),
	}
	return s, errors.Trace(s.bootstrapDynamo())
}

func (s *service) Regimen(id string) (*svc.Regimen, bool, error) {
	getResp, operr := s.kvs.GetItem(&dynamodb.GetItemInput{
		TableName: s.regimenTableName,
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		ReturnConsumedCapacity: indexesRCC,
	})
	if operr != nil {
		return nil, false, errors.Trace(operr)
	}
	if getResp.Item == nil {
		return nil, false, errors.Trace(api.ErrNotFound(fmt.Sprintf("Unable to locate regimen with ID %s", id)))
	}

	r := &svc.Regimen{}
	if err := json.Unmarshal(getResp.Item[regimenAN].B, r); err != nil {
		return nil, false, errors.Trace(err)
	}
	published := getResp.Item[publishedAN].BOOL
	if published == nil {
		published = ptr.Bool(false)
	}
	vc, err := strconv.ParseInt(*getResp.Item[viewCountAN].N, 10, 64)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	r.ViewCount = int(vc)

	return r, *published, nil
}

func (s *service) IncrementViewCount(id string) error {
	updateResp, operr := s.kvs.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: s.regimenTableName,
		Key: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},
		},
		UpdateExpression:          incrementViewCountUE,
		ExpressionAttributeValues: incrementSingleValueUEAV,
		ReturnConsumedCapacity:    indexesRCC,
		ReturnValues:              allNewRV,
	})
	if operr != nil {
		return errors.Trace(operr)
	}

	if updateResp.Attributes == nil {
		return errors.Trace(api.ErrNotFound(fmt.Sprintf("Unable to locate regimen with ID with id %s", id)))
	}

	r := &svc.Regimen{}
	if err := json.Unmarshal(updateResp.Attributes[regimenAN].B, r); err != nil {
		return errors.Trace(err)
	}

	// Update the tag index table, if any updates fail who cares, log it
	regimenID := ptr.String(id)
	for i, tag := range r.Tags {
		tag = strings.ToLower(tag)

		if i != 0 {
			// Throttle the index updates to 4 a second for a maximum search delay of 6 seconds
			time.Sleep(250 * time.Millisecond)
		}
		_, err := s.kvs.UpdateItem(&dynamodb.UpdateItemInput{
			TableName: s.regimenTagTableName,
			Key: map[string]*dynamodb.AttributeValue{
				tagAN: {
					S: ptr.String(tag),
				},
				regimenIDAN: {
					S: regimenID,
				},
			},
			UpdateExpression:          incrementViewCountUE,
			ExpressionAttributeValues: incrementSingleValueUEAV,
		})
		if err != nil {
			golog.Errorf("Error while incrementing tag index table view_count - tag: %s, regimen_id: %s - %s", tag, id, err)
		}
	}

	return nil
}

type putRegimenAnalytics struct {
	Err              error                        `json:"error,omitempty"`
	ConsumedCapacity []*dynamodb.ConsumedCapacity `json:"consumed_capacity"`
	RegimenID        string                       `json:"regimen_id"`
	Published        bool                         `json:"published"`
	Tags             []string                     `json:"tags"`
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

	putRequest := &dynamodb.PutRequest{
		Item: map[string]*dynamodb.AttributeValue{
			regimenIDAN: {
				S: ptr.String(id),
			},

			// An unpublished regimen always has a view count of 0 and a published regimen should not be mutated so always PUT with a 0 value
			viewCountAN: {
				N: zeroAV,
			},
			publishedAN: {
				BOOL: ptr.Bool(published),
			},
			regimenAN: {
				B: regimenData,
			},
		},
	}
	if r.SourceRegimenID != "" {
		putRequest.Item[sourceRegimenIDAN] = &dynamodb.AttributeValue{
			S: ptr.String(r.SourceRegimenID),
		}
	}
	regimenWriteRequests := []*dynamodb.WriteRequest{
		{
			PutRequest: putRequest,
		},
	}

	batchWriteInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			*s.regimenTableName: regimenWriteRequests,
		},
		ReturnConsumedCapacity: indexesRCC,
	}

	// Only map a regimen into the tag set if it is being published
	if published {
		// track all the tags we're adding since we can't write duplicates to dynamo
		usedTags := make(map[string]bool)
		tagWriteRequests := make([]*dynamodb.WriteRequest, 0, len(r.Tags))
		for _, tag := range r.Tags {
			tag = strings.ToLower(tag)
			if _, ok := usedTags[tag]; !ok {
				usedTags[tag] = true
			} else {
				continue
			}
			tagWriteRequests = append(tagWriteRequests, &dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						tagAN: {
							S: ptr.String(tag),
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
			})
		}

		// Only attach tag write requests if there are any
		if len(tagWriteRequests) > 0 {
			batchWriteInput.RequestItems[*s.regimenTagTableName] = tagWriteRequests
		}
	}
	batchResponse, operr := s.kvs.BatchWriteItem(batchWriteInput)

	// Emit our operation metrics
	conc.Go(func() {
		opAnalytics := &putRegimenAnalytics{
			RegimenID: id,
			Err:       operr,
			Tags:      r.Tags,
			Published: published,
		}
		if batchResponse != nil {
			opAnalytics.ConsumedCapacity = batchResponse.ConsumedCapacity
		}
		s.publisher.PublishAsync(&analytics.ServerEvent{
			Event:     "put_regimen:batch_write_item",
			Timestamp: analytics.Time(time.Now()),
			ExtraJSON: analytics.JSONString(opAnalytics),
		})
	})

	return errors.Trace(operr)
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

type tagQueryAnalytics struct {
	Err              error                      `json:"error"`
	ConsumedCapacity *dynamodb.ConsumedCapacity `json:"consumed_capacity"`
	Tag              string                     `json:"tag"`
}

func (s *service) TagQuery(tags []string) ([]*svc.Regimen, error) {
	regimenIDs := make(map[string]*regimenIDViewCount)
	for _, t := range tags {
		t = strings.ToLower(t)
		tagRegimenIDs, operr := s.kvs.Query(&dynamodb.QueryInput{
			TableName:                 s.regimenTagTableName,
			IndexName:                 regimenTagTableTagViewIndexName,
			KeyConditionExpression:    tagEqualsTagKCE,
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":tag": {S: ptr.String(t)}},
			// Only return a maximum of 100 records
			Limit: limitValue,
			// Order by view count desc
			ScanIndexForward:       tagQueryScanIndexDirection,
			ReturnConsumedCapacity: indexesRCC,
		})

		// Emit our operation metrics
		conc.Go(func() {
			opAnalytics := &tagQueryAnalytics{
				Err: operr,
				Tag: t,
			}
			if tagRegimenIDs != nil {
				opAnalytics.ConsumedCapacity = tagRegimenIDs.ConsumedCapacity
			}
			data, err := json.Marshal(opAnalytics)
			if err != nil {
				golog.Errorf("Error while marshaling analytics section for TagQuery:Query tag %s: %s", t, err)
			}
			s.publisher.PublishAsync(&analytics.ServerEvent{
				Event:     "tag_query:query",
				Timestamp: analytics.Time(time.Now()),
				ExtraJSON: string(data),
			})
		})

		if operr != nil {
			return nil, errors.Trace(operr)
		}

		// Merge in the result of each query
		for _, v := range tagRegimenIDs.Items {
			vc, err := strconv.ParseInt(*v[viewCountAN].N, 10, 64)
			if err != nil {
				return nil, errors.Trace(err)
			}
			regimenIDs[*v[regimenIDAN].S] = &regimenIDViewCount{
				regimenID: *v[regimenIDAN].S,
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
			regimenIDAN: {S: ptr.String(rIDVC.regimenID)},
		}
	}

	regimensResp, err := s.kvs.BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			*s.regimenTableName: {Keys: regimenIDRequests},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Only do capacity here since we might have unpublished regimens we need to skip
	// TODO: Remove legacy unpublished regimens from the tag space table as they are no longer allowed and move this to actual size preallocation
	rs := make([]*svc.Regimen, 0, len(regimensResp.Responses[*s.regimenTableName]))
	for _, regimen := range regimensResp.Responses[*s.regimenTableName] {
		r := &svc.Regimen{}
		if err := json.Unmarshal(regimen[regimenAN].B, r); err != nil {
			return nil, errors.Trace(err)
		}

		vc, err := strconv.ParseInt(*regimen[viewCountAN].N, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		r.ViewCount = int(vc)
		rs = append(rs, r)
	}
	sort.Sort(sort.Reverse(svc.ByViewCount(rs)))
	return rs, nil
}

func (s *service) FoundationOf(id string, maxResults int) ([]*svc.Regimen, error) {
	if maxResults > int(*limitValue) || maxResults == 0 {
		maxResults = int(*limitValue)
	}
	rs := make([]*svc.Regimen, 0, *limitValue)
	regimenResult, operr := s.kvs.Query(&dynamodb.QueryInput{
		TableName:                 s.regimenTableName,
		IndexName:                 regimenTableSourceRegimenRegimenIndexName,
		KeyConditionExpression:    sourceRegimenIDEqualsSourceRegimenIDKCE,
		FilterExpression:          ptr.String("published = :published"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":source_regimen_id": {S: ptr.String(id)}, ":published": {BOOL: publishedFilter}},
		// Only return a maximum of 100 records to sort between, if a regimen is the foundation of a TON then we may miss some :(
		Limit: limitValue,
		ReturnConsumedCapacity: indexesRCC,
	})
	if operr != nil {
		return nil, errors.Trace(operr)
	}

	// Merge in the result of each query
	for _, v := range regimenResult.Items {
		if v[viewCountAN].N == nil {
			golog.Errorf("Encountered a nil view count for regimen %s when doing foundation query, moving on", v[regimenIDAN].String())
			continue
		}

		vc, err := strconv.ParseInt(*v[viewCountAN].N, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		regimen := &svc.Regimen{}
		if err := json.Unmarshal(v[regimenAN].B, regimen); err != nil {
			golog.Errorf("Unable to deserialize regimen %s when doing foundation query, moving on: %s", v[regimenIDAN].String(), err)
			continue
		}
		regimen.ViewCount = int(vc)
		rs = append(rs, regimen)
	}

	sort.Sort(sort.Reverse(svc.ByViewCount(rs)))
	if len(rs) > maxResults {
		rs = rs[:maxResults]
	}

	return rs, nil
}

func (s *service) bootstrapDynamo() error {
	// Create the svc table that maps ids to svc indexed by the ID and view count
	if err := awsutil.CreateDynamoDBTable(s.kvs, &dynamodb.CreateTableInput{
		TableName: s.regimenTableName,
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
	}); err != nil {
		return errors.Trace(err)
	}

	// Create the tags table that maps and is indexed by tags to regimen id's
	return errors.Trace(awsutil.CreateDynamoDBTable(s.kvs, &dynamodb.CreateTableInput{
		TableName: s.regimenTagTableName,
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
}
