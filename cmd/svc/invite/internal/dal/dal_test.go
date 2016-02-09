package dal

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func testDB(t *testing.T) dynamodbiface.DynamoDBAPI {
	dbEndpoint := os.Getenv("TEST_DYNAMODB_ENDPOINT")
	if dbEndpoint == "" {
		t.Skip("TEST_DYNAMODB_ENDPOINT not set")
	}
	dynamoConfig := &aws.Config{
		Region:      ptr.String("us-east-1"),
		DisableSSL:  ptr.Bool(true),
		Endpoint:    &dbEndpoint,
		Credentials: credentials.NewEnvCredentials(),
	}
	return dynamodb.New(session.New(dynamoConfig))
}

func TestAttribution(t *testing.T) {
	db := testDB(t)
	dal := New(db, "local")
	ctx := context.Background()

	_, err := dal.AttributionData(ctx, "nope")
	test.Equals(t, ErrNotFound, errors.Cause(err))

	deviceID := randomID()
	expVals := map[string]string{
		"token": "abc",
		"foo":   "bar",
	}
	test.OK(t, dal.SetAttributionData(ctx, deviceID, expVals))
	vals, err := dal.AttributionData(ctx, deviceID)
	test.OK(t, err)
	test.Equals(t, expVals, vals)

	// Make sure updating values overwrites everything (doesn't leave any old values even if key isn't reused)
	expVals = map[string]string{
		"abc": "123",
	}
	test.OK(t, dal.SetAttributionData(ctx, deviceID, expVals))
	vals, err = dal.AttributionData(ctx, deviceID)
	test.OK(t, err)
	test.Equals(t, expVals, vals)
}

func TestInviteColleague(t *testing.T) {
	db := testDB(t)
	dal := New(db, "local")
	ctx := context.Background()

	_, err := dal.InviteForToken(ctx, "nope")
	test.Equals(t, ErrNotFound, errors.Cause(err))

	invite := &models.Invite{
		Type:                 models.ColleagueInvite,
		OrganizationEntityID: "e_1",
		InviterEntityID:      "e_2",
		Token:                randomID(),
		Email:                "someone@somewhere.com",
		PhoneNumber:          "+15551112222",
		URL:                  "https://example.com",
		Created:              time.Unix(123, 0),
	}
	test.OK(t, dal.InsertInvite(ctx, invite))

	// Trying to insert the same token twice should fail
	test.Equals(t, ErrDuplicateInviteToken, errors.Cause(dal.InsertInvite(ctx, invite)))

	in, err := dal.InviteForToken(ctx, invite.Token)
	test.OK(t, err)
	test.Equals(t, invite, in)
}

func randomID() string {
	return strconv.FormatInt(rand.Int63(), 10)
}