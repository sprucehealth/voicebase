package dal

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"golang.org/x/net/context"
)

const (
	attribValuesKey         = "AttributionValues"
	createdTimestampKey     = "CreatedTimestamp"
	deviceIDKey             = "DeviceID"
	emailKey                = "Email"
	inviterEntityIDKey      = "InviterEntityID"
	inviteTokenKey          = "InviteToken"
	organizationEntityIDKey = "OrganizationEntityID"
	phoneNumberKey          = "PhoneNumber"
	typeKey                 = "Type"
	urlKey                  = "URL"
)

// ErrNotFound is the error when an object is missing
var ErrNotFound = errors.New("invite/dal: not found")

// ErrDuplicateInviteToken is returned when trying to insert an invite with a token that is already used
var ErrDuplicateInviteToken = errors.New("invite/dal: an invite with the provided token already exists")

// DAL is the interface implemented by a data access layer for the invite service
type DAL interface {
	AttributionData(ctx context.Context, deviceID string) (map[string]string, error)
	SetAttributionData(ctx context.Context, deviceID string, values map[string]string) error
	InsertInvite(ctx context.Context, invite *models.Invite) error
	InviteForToken(ctx context.Context, token string) (*models.Invite, error)
}

type dal struct {
	db               dynamodbiface.DynamoDBAPI
	attributionTable string
	inviteTable      string
}

// New returns a new DAL using DynamoDB for storage
func New(db dynamodbiface.DynamoDBAPI, env string) DAL {
	return &dal{
		db:               db,
		attributionTable: env + "-invite-attribution",
		inviteTable:      env + "-invite",
	}
}

func (d *dal) AttributionData(ctx context.Context, deviceID string) (map[string]string, error) {
	res, err := d.db.GetItem(&dynamodb.GetItemInput{
		ConsistentRead: ptr.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			deviceIDKey: {S: &deviceID},
		},
		TableName:            &d.attributionTable,
		ProjectionExpression: ptr.String(attribValuesKey),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	itemVals := res.Item[attribValuesKey]
	if itemVals == nil {
		return nil, errors.Trace(ErrNotFound)
	}
	attrVals := make(map[string]string, len(itemVals.M))
	for name, val := range itemVals.M {
		attrVals[name] = *val.S
	}
	return attrVals, nil
}

func (d *dal) SetAttributionData(ctx context.Context, deviceID string, values map[string]string) error {
	if deviceID == "" {
		return errors.Trace(errors.New("deviceID required"))
	}
	itemVals := make(map[string]*dynamodb.AttributeValue, len(values))
	for name, val := range values {
		itemVals[name] = &dynamodb.AttributeValue{S: ptr.String(val)}
	}
	_, err := d.db.PutItem(&dynamodb.PutItemInput{
		TableName: &d.attributionTable,
		Item: map[string]*dynamodb.AttributeValue{
			deviceIDKey:     {S: &deviceID},
			attribValuesKey: {M: itemVals},
		},
	})
	return errors.Trace(err)
}

func (d *dal) InsertInvite(ctx context.Context, invite *models.Invite) error {
	if invite.Token == "" {
		return errors.Trace(errors.New("Token required"))
	}
	if invite.Type == "" {
		return errors.Trace(errors.New("Type required"))
	}
	if invite.InviterEntityID == "" {
		return errors.Trace(errors.New("InviterEntityID required"))
	}
	if invite.OrganizationEntityID == "" {
		return errors.Trace(errors.New("OrganizationEntityID required"))
	}
	if invite.Email == "" {
		return errors.Trace(errors.New("Email required"))
	}
	if invite.PhoneNumber == "" {
		return errors.Trace(errors.New("PhoneNumber required"))
	}
	if invite.URL == "" {
		return errors.Trace(errors.New("URL required"))
	}
	if invite.Created.IsZero() {
		return errors.Trace(errors.New("Created required"))
	}
	_, err := d.db.PutItem(&dynamodb.PutItemInput{
		TableName:           &d.inviteTable,
		ConditionExpression: ptr.String("attribute_not_exists(" + inviteTokenKey + ")"),
		Item: map[string]*dynamodb.AttributeValue{
			inviteTokenKey:          {S: &invite.Token},
			typeKey:                 {S: ptr.String(string(invite.Type))},
			organizationEntityIDKey: {S: &invite.OrganizationEntityID},
			inviterEntityIDKey:      {S: &invite.InviterEntityID},
			emailKey:                {S: &invite.Email},
			phoneNumberKey:          {S: &invite.PhoneNumber},
			urlKey:                  {S: &invite.URL},
			createdTimestampKey:     {N: ptr.String(strconv.FormatInt(invite.Created.UnixNano(), 10))},
		},
	})
	if err != nil {
		if e, ok := err.(awserr.RequestFailure); ok && e.Code() == "ConditionalCheckFailedException" {
			return errors.Trace(ErrDuplicateInviteToken)
		}
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) InviteForToken(ctx context.Context, token string) (*models.Invite, error) {
	res, err := d.db.GetItem(&dynamodb.GetItemInput{
		ConsistentRead: ptr.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			inviteTokenKey: {S: &token},
		},
		TableName: &d.inviteTable,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(res.Item) == 0 {
		return nil, errors.Trace(ErrNotFound)
	}
	ct, err := strconv.ParseInt(*res.Item[createdTimestampKey].N, 10, 64)
	if err != nil {
		golog.Errorf("Invalid created time in invite for token %s", token)
	}
	return &models.Invite{
		Token:                token,
		Type:                 models.InviteType(*res.Item[typeKey].S),
		OrganizationEntityID: *res.Item[organizationEntityIDKey].S,
		InviterEntityID:      *res.Item[inviterEntityIDKey].S,
		Email:                *res.Item[emailKey].S,
		PhoneNumber:          *res.Item[phoneNumberKey].S,
		URL:                  *res.Item[urlKey].S,
		Created:              time.Unix(ct/1e9, ct%1e9),
	}, nil
}
