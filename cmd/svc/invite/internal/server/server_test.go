package server

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	branchmock "github.com/sprucehealth/backend/libs/branch/mock"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	excommsmock "github.com/sprucehealth/backend/svc/excomms/mock"
	"github.com/sprucehealth/backend/svc/invite"
)

type sTokenGenerator struct{}

func (t *sTokenGenerator) GenerateToken() (string, error) {
	return "simpleToken", nil
}

type cTokenGenerator struct{}

func (t *cTokenGenerator) GenerateToken() (string, error) {
	return "complexToken", nil
}

func init() {
	simpleTokenGenerator = &sTokenGenerator{}
	complexTokenGenerator = &cTokenGenerator{}
	conc.Testing = true
}

func TestAttribution(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()

	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	srv := New(dl, nil, nil, nil, snsC, nil, nil, "", "", "", "")

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

func TestInviteColleagues(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	dir := dirmock.New(t)
	defer dir.Finish()
	branch := branchmock.New(t)
	defer branch.Finish()
	sg := newSGMock(t)
	defer sg.Finish()
	clk := clock.NewManaged(time.Unix(10000000, 0))
	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	excommsC := excommsmock.New(t)
	defer excommsC.Finish()
	srv := New(dl, clk, dir, excommsC, snsC, branch, sg, "from@example.com", "+1234567890", "eventsTopic", "https://app.sprucehealth.com/signup?some=other")

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
			{ID: "org", Type: directory.EntityType_ORGANIZATION, Info: &directory.EntityInfo{DisplayName: "Orgo"}},
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
	values := map[string]string{
		"invite_token": "complexToken",
		"client_data":  `{"organization_invite":{"popover":{"title":"Welcome to Spruce!","message":"Inviter has invited you to join them on Spruce.","button_text":"Okay"},"org_id":"org","org_name":"Orgo"}}`,
		"$desktop_url": "https://app.sprucehealth.com/signup?invite=complexToken&some=other",
	}
	clientData := make(map[string]interface{}, len(values))
	for k, v := range values {
		clientData[k] = v
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                "complexToken",
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Type:                 models.ColleagueInvite,
		Email:                "someone@example.com",
		PhoneNumber:          "+15555551212",
		URL:                  "https://example.com/invite",
		Created:              clk.Now(),
		Values:               values,
	}).WithReturns(nil))

	// Send invite email
	sg.Expect(mock.NewExpectation(sg.Send, &sendgrid.SGMail{
		To:      []string{"someone@example.com"},
		Subject: fmt.Sprintf("Invite to join %s on Spruce", "Orgo"),
		Text: fmt.Sprintf(
			"Spruce is a communication and digital care app. By joining %s on Spruce, you'll be able to collaborate with colleagues around your patients' care, securely and efficiently.\n\nClick this link to get started:\n%s\n\nOnce you've created your account, you're all set to start catching up on the latest conversation.\n\nIf you have any troubles, we're here to help - simply reply to this email!\n\nThanks,\nThe Team at Spruce\n\nP.S.: Learn more about Spruce here: https://www.sprucehealth.com",
			"Orgo", "https://example.com/invite"),
		From:     "from@example.com",
		FromName: "Inviter",
		SMTPAPIHeader: smtpapi.SMTPAPIHeader{
			UniqueArgs: map[string]string{
				"invite_token": "complexToken",
			},
		},
	}).WithReturns(nil))

	eventData, err := events.MarshalEnvelope(events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_COLLEAGUES,
		Details: &invite.Event_InvitedColleagues{
			InvitedColleagues: &invite.InvitedColleagues{
				OrganizationEntityID: "org",
				InviterEntityID:      "ent",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

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

func TestInvitePatients(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	dir := dirmock.New(t)
	defer dir.Finish()
	branch := branchmock.New(t)
	defer branch.Finish()
	sg := newSGMock(t)
	defer sg.Finish()
	clk := clock.NewManaged(time.Unix(10000000, 0))
	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	excommsC := excommsmock.New(t)
	defer excommsC.Finish()
	srv := New(dl, clk, dir, excommsC, snsC, branch, sg, "from@example.com", "+1234567890", "eventsTopic", "https://app.sprucehealth.com/signup?some=other")

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
			{ID: "org", Type: directory.EntityType_ORGANIZATION, Info: &directory.EntityInfo{DisplayName: "Batman Inc"}},
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
			{ID: "ent", Type: directory.EntityType_INTERNAL, Info: &directory.EntityInfo{DisplayName: "Batman"}},
		},
	}, nil))

	// Generate branch URL
	values := map[string]string{
		"invite_token": "simpleToken",
		"client_data":  `{"patient_invite":{"greeting":{"title":"Welcome Alfred!","message":"Let's create your account so you can start securely messaging with Batman Inc.","button_text":"Get Started"},"org_id":"org","org_name":"Batman Inc"}}`,
		"$desktop_url": "https://app.sprucehealth.com/signup?invite=simpleToken&some=other",
	}
	clientData := make(map[string]interface{}, len(values))
	for k, v := range values {
		clientData[k] = v
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                "simpleToken",
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Type:                 models.PatientInvite,
		PhoneNumber:          phiAttributeText,
		Email:                phiAttributeText,
		URL:                  "https://example.com/invite",
		ParkedEntityID:       "parkedEntityID",
		Created:              clk.Now(),
		Values:               values,
	}).WithReturns(nil))

	// Send invite sms
	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            "Alfred - Batman Inc has invited you to use Spruce for secure messaging and digital care.",
				FromPhoneNumber: "+1234567890",
				ToPhoneNumber:   "+15555551212",
			},
		},
	}).WithReturns(&excomms.SendMessageResponse{}, nil))

	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            "Get the Spruce app now and join them. https://example.com/invite [simpleToken]",
				FromPhoneNumber: "+1234567890",
				ToPhoneNumber:   "+15555551212",
			},
		},
	}).WithReturns(&excomms.SendMessageResponse{}, nil))

	eventData, err := events.MarshalEnvelope(events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_PATIENTS,
		Details: &invite.Event_InvitedPatients{
			InvitedPatients: &invite.InvitedPatients{
				OrganizationEntityID: "org",
				InviterEntityID:      "ent",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

	ires, err := srv.InvitePatients(nil, &invite.InvitePatientsRequest{
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Patients: []*invite.Patient{
			{FirstName: "Alfred", PhoneNumber: "+15555551212", ParkedEntityID: "parkedEntityID"},
		},
	})
	test.OK(t, err)
	test.Equals(t, &invite.InvitePatientsResponse{}, ires)
}

func TestInvitePatientsNoFirstName(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	dir := dirmock.New(t)
	defer dir.Finish()
	branch := branchmock.New(t)
	defer branch.Finish()
	sg := newSGMock(t)
	defer sg.Finish()
	clk := clock.NewManaged(time.Unix(10000000, 0))
	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	excommsC := excommsmock.New(t)
	defer excommsC.Finish()
	srv := New(dl, clk, dir, excommsC, snsC, branch, sg, "from@example.com", "+1234567890", "eventsTopic", "https://app.sprucehealth.com/signup?some=other")

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
			{ID: "org", Type: directory.EntityType_ORGANIZATION, Info: &directory.EntityInfo{DisplayName: "Batman Inc"}},
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
			{ID: "ent", Type: directory.EntityType_INTERNAL, Info: &directory.EntityInfo{DisplayName: "Batman"}},
		},
	}, nil))

	// Generate branch URL
	values := map[string]string{
		"invite_token": "simpleToken",
		"client_data":  `{"patient_invite":{"greeting":{"title":"Welcome!","message":"Let's create your account so you can start securely messaging with Batman Inc.","button_text":"Get Started"},"org_id":"org","org_name":"Batman Inc"}}`,
		"$desktop_url": "https://app.sprucehealth.com/signup?invite=simpleToken&some=other",
	}
	clientData := make(map[string]interface{}, len(values))
	for k, v := range values {
		clientData[k] = v
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                "simpleToken",
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Type:                 models.PatientInvite,
		PhoneNumber:          phiAttributeText,
		Email:                phiAttributeText,
		URL:                  "https://example.com/invite",
		ParkedEntityID:       "parkedEntityID",
		Created:              clk.Now(),
		Values:               values,
	}).WithReturns(nil))

	// Send invite sms
	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            "Batman Inc has invited you to use Spruce for secure messaging and digital care.",
				FromPhoneNumber: "+1234567890",
				ToPhoneNumber:   "+15555551212",
			},
		},
	}).WithReturns(&excomms.SendMessageResponse{}, nil))

	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				Text:            "Get the Spruce app now and join them. https://example.com/invite [simpleToken]",
				FromPhoneNumber: "+1234567890",
				ToPhoneNumber:   "+15555551212",
			},
		},
	}).WithReturns(&excomms.SendMessageResponse{}, nil))

	eventData, err := events.MarshalEnvelope(events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_PATIENTS,
		Details: &invite.Event_InvitedPatients{
			InvitedPatients: &invite.InvitedPatients{
				OrganizationEntityID: "org",
				InviterEntityID:      "ent",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

	ires, err := srv.InvitePatients(nil, &invite.InvitePatientsRequest{
		OrganizationEntityID: "org",
		InviterEntityID:      "ent",
		Patients: []*invite.Patient{
			{FirstName: "", PhoneNumber: "+15555551212", ParkedEntityID: "parkedEntityID"},
		},
	})
	test.OK(t, err)
	test.Equals(t, &invite.InvitePatientsResponse{}, ires)
}

func TestLookupInvite(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	srv := New(dl, nil, nil, nil, snsC, nil, nil, "", "", "", "")

	dl.Expect(mock.NewExpectation(dl.InviteForToken, "testtoken").WithReturns(
		&models.Invite{
			Type:                 models.ColleagueInvite,
			Token:                "testtoken",
			OrganizationEntityID: "org",
			InviterEntityID:      "ent",
			Email:                "someone@example.com",
			PhoneNumber:          "+15555551212",
			Created:              time.Now(),
			Values: map[string]string{
				"foo": "bar",
			},
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
		Values: []*invite.AttributionValue{{Key: "foo", Value: "bar"}},
	}, res)
}

func TestMarkInviteConsumed(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	snsC := mock.NewSNSAPI(t)
	defer snsC.Finish()
	srv := New(dl, nil, nil, nil, snsC, nil, nil, "", "", "", "")

	dl.Expect(mock.NewExpectation(dl.DeleteInvite, "testtoken").WithReturns(nil))
	res, err := srv.MarkInviteConsumed(nil, &invite.MarkInviteConsumedRequest{Token: "testtoken"})
	test.OK(t, err)
	test.Equals(t, &invite.MarkInviteConsumedResponse{}, res)
}
