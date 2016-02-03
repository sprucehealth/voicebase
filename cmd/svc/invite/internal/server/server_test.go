package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	branchmock "github.com/sprucehealth/backend/libs/branch/mock"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/test"
)

func init() {
	unitTesting = true
}

func TestAttribution(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := New(dl, nil, nil, nil, nil, "")

	values := []*invite.AttributionValue{
		{Key: "abc", Value: "123"},
	}
	valueMap := make(map[string]string, len(values))
	for _, v := range values {
		valueMap[v.Key] = v.Value
	}
	dl.Expect(mock.NewExpectation(dl.SetAttributionData, "dev", valueMap).WithReturns(nil))
	setRes, err := srv.SetAttributionData(nil, &invite.SetAttributionDataRequest{
		DeviceID: "dev",
		Values:   values,
	})
	test.OK(t, err)
	test.Equals(t, &invite.SetAttributionDataResponse{}, setRes)

	dl.Expect(mock.NewExpectation(dl.AttributionData, "dev").WithReturns(valueMap, nil))
	getRes, err := srv.AttributionData(nil, &invite.AttributionDataRequest{DeviceID: "dev"})
	test.OK(t, err)
	test.Equals(t, &invite.AttributionDataResponse{Values: values}, getRes)
}

func TestInviteColleague(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	dir := dirmock.New(t)
	defer dir.Finish()
	branch := branchmock.New(t)
	defer branch.Finish()
	sg := newSGMock(t)
	defer sg.Finish()
	clk := clock.NewManaged(time.Unix(10000000, 0))
	srv := New(dl, clk, dir, branch, sg, "from@example.com")

	// Lookup organization
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "org",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: "org", Type: directory.EntityType_ORGANIZATION, Info: &directory.EntityInfo{DisplayName: "Org"}},
		},
	}, nil))

	// Lookup inviter
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "ent",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: "ent", Type: directory.EntityType_INTERNAL, Info: &directory.EntityInfo{DisplayName: "Inviter"}},
		},
	}, nil))

	// Generate branch URL
	clientData := map[string]interface{}{
		"token":       "thetoken",
		"client_data": `{"organization_invite":{"popover":{"title":"Welcome to Spruce!","message":"Inviter has invited you to join them on Spruce.","button_text":"Get Started"},"org_id":"org"}}`,
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                "thetoken",
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Type:                 models.ColleagueInvite,
		Email:                "someone@example.com",
		PhoneNumber:          "+15555551212",
		URL:                  "https://example.com/invite",
		Created:              clk.Now(),
	}).WithReturns(nil))

	// Send invite email
	sg.Expect(mock.NewExpectation(sg.Send, &sendgrid.SGMail{
		To:      []string{"someone@example.com"},
		Subject: fmt.Sprintf("Invite to join %s", "Org"),
		Text: fmt.Sprintf(
			"I would like you to join my organization %s\n%s\n\nBest,\n%s",
			"Org", "https://example.com/invite", "Inviter"),
		From:     "from@example.com",
		FromName: "Inviter",
		SMTPAPIHeader: smtpapi.SMTPAPIHeader{
			UniqueArgs: map[string]string{
				"invite_token": "thetoken",
			},
		},
	}).WithReturns(nil))

	ires, err := srv.InviteColleagues(nil, &invite.InviteColleaguesRequest{
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Colleagues: []*invite.Colleague{
			{Email: "someone@example.com", PhoneNumber: "+15555551212"},
		},
	})
	test.OK(t, err)
	test.Equals(t, &invite.InviteColleaguesResponse{}, ires)
}

func TestLookupInvite(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := New(dl, nil, nil, nil, nil, "")

	dl.Expect(mock.NewExpectation(dl.InviteForToken, "testtoken").WithReturns(
		&models.Invite{
			Type:                 models.ColleagueInvite,
			Token:                "testtoken",
			OrganizationEntityID: "org",
			InviterEntityID:      "ent",
			Email:                "someone@example.com",
			PhoneNumber:          "+15555551212",
			Created:              time.Now(),
		}, nil))
	res, err := srv.LookupInvite(nil, &invite.LookupInviteRequest{Token: "testtoken"})
	test.OK(t, err)
	test.Equals(t, &invite.LookupInviteResponse{
		Type: invite.LookupInviteResponse_COLLEAGUE,
		Invite: &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				OrganizationEntityID: "org",
				InviterEntityID:      "ent",
				Colleague: &invite.Colleague{
					Email:       "someone@example.com",
					PhoneNumber: "+15555551212",
				},
			},
		},
	}, res)
}
