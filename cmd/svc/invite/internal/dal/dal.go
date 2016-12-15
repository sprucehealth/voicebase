package dal

import (
	"fmt"
	"strconv"
	"time"

	"context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	attribValuesKey            = "AttributionValues"
	createdTimestampKey        = "CreatedTimestamp"
	deviceIDKey                = "DeviceID"
	emailKey                   = "Email"
	entityIDKey                = "EntityID"
	inviterEntityIDKey         = "InviterEntityID"
	inviteTokenKey             = "InviteToken"
	isInviteKey                = "IsInvite"
	organizationEntityIDKey    = "OrganizationEntityID"
	parkedEntityIDKey          = "ParkedEntityID"
	phoneNumberKey             = "PhoneNumber"
	tagsKey                    = "Tags"
	typeKey                    = "Type"
	urlKey                     = "URL"
	valuesKey                  = "Values"
	verificationRequirementKey = "VerificationRequirement"
)

// ErrNotFound is the error when an object is missing
var ErrNotFound = errors.New("invite/dal: not found")

// ErrDuplicateInviteToken is returned when trying to insert an invite with a token that is already used
var ErrDuplicateInviteToken = errors.New("invite/dal: an invite with the provided token already exists")

// DAL is the interface implemented by a data access layer for the invite service
type DAL interface {
	AttributionData(ctx context.Context, deviceID string) (map[string]string, error)
	DeleteInvite(ctx context.Context, token string) error
	InsertEntityToken(ctx context.Context, entityID, token string) error
	InsertInvite(ctx context.Context, invite *models.Invite) error
	InviteForToken(ctx context.Context, token string) (*models.Invite, error)
	InvitesForParkedEntityID(ctx context.Context, parkedEntityID string) ([]*models.Invite, error)
	SetAttributionData(ctx context.Context, deviceID string, values map[string]string) error
	TokensForEntity(ctx context.Context, entityID string) ([]string, error)
	UpdateInvite(ctx context.Context, token string, update *models.InviteUpdate) (*models.Invite, error)
}

type dal struct {
	db                  dynamodbiface.DynamoDBAPI
	attributionTable    string
	inviteTable         string
	parkedEntityIDIndex string
	entityTokenTable    string
}

// New returns a new DAL using DynamoDB for storage
func New(db dynamodbiface.DynamoDBAPI, env string) DAL {
	return &dal{
		db:                  db,
		attributionTable:    env + "-invite-attribution",
		inviteTable:         env + "-invite",
		entityTokenTable:    env + "-entity-token",
		parkedEntityIDIndex: env + "-parked_entity_id-index",
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
		return nil, ErrNotFound
	}
	attrVals := make(map[string]string, len(itemVals.M))
	for name, val := range itemVals.M {
		attrVals[name] = *val.S
	}
	return attrVals, nil
}

func (d *dal) SetAttributionData(ctx context.Context, deviceID string, values map[string]string) error {
	if deviceID == "" {
		return errors.Errorf("deviceID required")
	}
	itemVals := make(map[string]*dynamodb.AttributeValue, len(values))
	for name, val := range values {
		itemVals[name] = &dynamodb.AttributeValue{S: ptr.String(val)}
	}
	isInvite := values["invite_token"] != ""
	in := &dynamodb.PutItemInput{
		TableName: &d.attributionTable,
		Item: map[string]*dynamodb.AttributeValue{
			deviceIDKey:     {S: &deviceID},
			isInviteKey:     {BOOL: &isInvite},
			attribValuesKey: {M: itemVals},
		},
	}
	// TODO: for now giving priority to invites. we'll eventually want a more complex
	//       behavior here (perhaps keeping all unique attribution data) and changing
	//       invite to not use attribution for tracking sign up flow.
	if !isInvite {
		// For non-invite data make sure not to overwrite any existing invite data
		// Using NOT isInvite = true rather than isInvite = false to handle existing items
		// that don't have the value at all. Can switch it later to simplify.
		in.ConditionExpression = ptr.String("NOT " + isInviteKey + " = :true")
		in.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":true": {BOOL: ptr.Bool(true)},
		}
	}
	_, err := d.db.PutItem(in)
	if err != nil {
		if e, ok := err.(awserr.RequestFailure); ok && e.Code() == "ConditionalCheckFailedException" {
			// The caller likely doesn't care about this and can't really do anything with the inforamtion.
			return nil
		}
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) InsertInvite(ctx context.Context, invite *models.Invite) error {
	item := make(map[string]*dynamodb.AttributeValue)
	if invite.Token == "" {
		return errors.Errorf("Token required")
	}
	item[inviteTokenKey] = &dynamodb.AttributeValue{S: &invite.Token}
	if invite.Type == "" {
		return errors.Errorf("Type required")
	}
	item[typeKey] = &dynamodb.AttributeValue{S: ptr.String(string(invite.Type))}
	if invite.OrganizationEntityID == "" {
		return errors.Errorf("OrganizationEntityID required")
	}
	item[organizationEntityIDKey] = &dynamodb.AttributeValue{S: &invite.OrganizationEntityID}
	switch invite.Type {
	case models.ColleagueInvite, models.PatientInvite:
		if invite.Type == models.PatientInvite {
			if invite.ParkedEntityID == "" {
				return errors.Errorf("ParkedEntityID required")
			}
			item[parkedEntityIDKey] = &dynamodb.AttributeValue{S: &invite.ParkedEntityID}
		}

		if invite.InviterEntityID != "" {
			item[inviterEntityIDKey] = &dynamodb.AttributeValue{S: &invite.InviterEntityID}
		}

		if invite.Email == "" {
			return errors.Errorf("Email required")
		}
		item[emailKey] = &dynamodb.AttributeValue{S: &invite.Email}
		if invite.PhoneNumber == "" {
			return errors.Errorf("PhoneNumber required")
		}
		item[phoneNumberKey] = &dynamodb.AttributeValue{S: &invite.PhoneNumber}
	}
	if invite.URL == "" {
		return errors.Errorf("URL required")
	}
	item[urlKey] = &dynamodb.AttributeValue{S: &invite.URL}
	if invite.Created.IsZero() {
		return errors.Errorf("Created required")
	}
	item[createdTimestampKey] = &dynamodb.AttributeValue{N: ptr.String(strconv.FormatInt(invite.Created.UnixNano(), 10))}

	if invite.VerificationRequirement != "" {
		item[verificationRequirementKey] = &dynamodb.AttributeValue{S: ptr.String(string(invite.VerificationRequirement))}
	}

	if len(invite.Values) > 0 {
		valuesAttr := make(map[string]*dynamodb.AttributeValue, len(invite.Values))
		for k, v := range invite.Values {
			valuesAttr[k] = &dynamodb.AttributeValue{S: ptr.String(v)}
		}
		item[valuesKey] = &dynamodb.AttributeValue{M: valuesAttr}
	}

	// Cannot have an empty set in dynamodb
	if len(invite.Tags) > 0 {
		item[tagsKey] = &dynamodb.AttributeValue{SS: ptr.Strings(invite.Tags)}
	}

	_, err := d.db.PutItem(&dynamodb.PutItemInput{
		TableName:           &d.inviteTable,
		ConditionExpression: ptr.String("attribute_not_exists(" + inviteTokenKey + ")"),
		Item:                item,
	})
	if err != nil {
		if e, ok := err.(awserr.RequestFailure); ok && e.Code() == "ConditionalCheckFailedException" {
			return ErrDuplicateInviteToken
		}
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) UpdateInvite(ctx context.Context, token string, update *models.InviteUpdate) (*models.Invite, error) {
	if token == "" {
		return nil, errors.Errorf("Token required")
	}

	updateItemInput := &dynamodb.UpdateItemInput{
		TableName:    ptr.String(d.inviteTable),
		Key:          map[string]*dynamodb.AttributeValue{inviteTokenKey: &dynamodb.AttributeValue{S: &token}},
		ReturnValues: ptr.String(dynamodb.ReturnValueAllNew),
	}

	if len(update.Tags) == 0 {
		updateItemInput.UpdateExpression = ptr.String(fmt.Sprintf("REMOVE %s", tagsKey))
	} else {
		updateItemInput.UpdateExpression = ptr.String(fmt.Sprintf("SET %s = :tags", tagsKey))
		updateItemInput.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
			`:tags`: {SS: ptr.Strings(update.Tags)},
		}
	}

	res, err := d.db.UpdateItem(updateItemInput)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(res.Attributes) == 0 {
		return nil, ErrNotFound
	}
	return inviteFromAttributes(res.Attributes), nil
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
		return nil, ErrNotFound
	}

	return inviteFromAttributes(res.Item), nil
}

func (d *dal) InvitesForParkedEntityID(ctx context.Context, parkedEntityID string) ([]*models.Invite, error) {
	res, err := d.db.Query(&dynamodb.QueryInput{
		TableName:              &d.inviteTable,
		IndexName:              &d.parkedEntityIDIndex,
		KeyConditionExpression: ptr.String("ParkedEntityID = :parkedEntityID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			`:parkedEntityID`: {S: &parkedEntityID},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	invites := make([]*models.Invite, len(res.Items))
	for i, ri := range res.Items {
		invites[i] = inviteFromAttributes(ri)
	}

	return invites, nil
}

func inviteFromAttributes(attributes map[string]*dynamodb.AttributeValue) *models.Invite {
	ct, err := strconv.ParseInt(*attributes[createdTimestampKey].N, 10, 64)
	if err != nil {
		golog.Errorf("Invalid created time in invite %s", *attributes[inviteTokenKey])
	}
	// Not all invites have associated parked entity ID's
	var parkedEntityID string
	if peID, ok := attributes[parkedEntityIDKey]; ok {
		parkedEntityID = *peID.S
	}
	var inviterEntityID string
	if ieID, ok := attributes[inviterEntityIDKey]; ok {
		inviterEntityID = *ieID.S
	}
	var email string
	if em, ok := attributes[emailKey]; ok {
		email = *em.S
	}
	var phoneNumber string
	if pn, ok := attributes[phoneNumberKey]; ok {
		phoneNumber = *pn.S
	}
	var verificationRequirement models.VerificationRequirement
	if attributes[verificationRequirementKey] != nil {
		verificationRequirement = models.VerificationRequirement(*attributes[verificationRequirementKey].S)
	}
	inv := &models.Invite{
		Token:                   *attributes[inviteTokenKey].S,
		Type:                    models.InviteType(*attributes[typeKey].S),
		OrganizationEntityID:    *attributes[organizationEntityIDKey].S,
		InviterEntityID:         inviterEntityID,
		Email:                   email,
		PhoneNumber:             phoneNumber,
		URL:                     *attributes[urlKey].S,
		ParkedEntityID:          parkedEntityID,
		Created:                 time.Unix(ct/1e9, ct%1e9),
		VerificationRequirement: verificationRequirement,
	}
	valuesAttr := attributes[valuesKey].M
	inv.Values = make(map[string]string, len(valuesAttr))
	for k, v := range valuesAttr {
		inv.Values[k] = *v.S
	}
	tags := attributes[tagsKey]
	if tags != nil {
		inv.Tags = make([]string, len(tags.SS))
		for i, t := range tags.SS {
			inv.Tags[i] = *t
		}
	}

	return inv
}

func (d *dal) DeleteInvite(ctx context.Context, token string) error {
	_, err := d.db.DeleteItem(&dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			inviteTokenKey: {S: &token},
		},
		TableName: &d.inviteTable,
	})
	return errors.Trace(err)
}

func (d *dal) InsertEntityToken(ctx context.Context, entityID, token string) error {
	if entityID == "" {
		return errors.Errorf("Entity ID required")
	}
	if token == "" {
		return errors.Errorf("Token required")
	}

	// Get our existing set of tokens to append to
	tokens, err := d.TokensForEntity(ctx, entityID)
	if err != nil && err != ErrNotFound {
		return errors.Trace(err)
	}
	tokens = append(tokens, token)
	item := map[string]*dynamodb.AttributeValue{
		entityIDKey:         {S: &entityID},
		inviteTokenKey:      {SS: ptr.Strings(tokens)},
		createdTimestampKey: {N: ptr.String(strconv.FormatInt(time.Now().UnixNano(), 10))},
	}
	if _, err := d.db.PutItem(&dynamodb.PutItemInput{
		TableName: &d.entityTokenTable,
		Item:      item,
	}); err != nil {
		if e, ok := err.(awserr.RequestFailure); ok && e.Code() == "ConditionalCheckFailedException" {
			return ErrDuplicateInviteToken
		}
		return errors.Trace(err)
	}
	return nil
}

func (d *dal) TokensForEntity(ctx context.Context, entityID string) ([]string, error) {
	res, err := d.db.GetItem(&dynamodb.GetItemInput{
		ConsistentRead: ptr.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			entityIDKey: {S: &entityID},
		},
		TableName: &d.entityTokenTable,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(res.Item) == 0 {
		return nil, ErrNotFound
	}

	// Map the old single token model into the new one
	var tokens []string
	token, ok := res.Item[inviteTokenKey]
	if !ok {
		return nil, ErrNotFound
	}
	if token.S != nil {
		tokens = []string{*token.S}
	} else {
		tokens = make([]string, len(token.SS))
		for i, ps := range token.SS {
			tokens[i] = *ps
		}
	}
	return tokens, nil
}
