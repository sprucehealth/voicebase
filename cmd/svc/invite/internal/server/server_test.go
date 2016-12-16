package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	branchmock "github.com/sprucehealth/backend/libs/branch/mock"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	excommsmock "github.com/sprucehealth/backend/svc/excomms/mock"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/invite/clientdata"
	"github.com/sprucehealth/backend/svc/settings"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	snsC := mock.NewSNSAPI(t)
	defer mock.FinishAll(dl, snsC)
	srv := New(dl, nil, nil, nil, nil, snsC, nil, "", "", "", "", "", "")

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
	dir := dirmock.New(t)
	branch := branchmock.New(t)
	snsC := mock.NewSNSAPI(t)
	excommsC := excommsmock.New(t)
	defer mock.FinishAll(dl, dir, branch, snsC, excommsC)
	clk := clock.NewManaged(time.Unix(10000000, 0))

	srv := New(dl, clk, dir, excommsC, nil, snsC, branch, "from@example.com", "+1234567890", "eventsTopic", "https://app.sprucehealth.com/signup?some=other", "templateID", "patientInviteTemplateID")

	// Lookup organization
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Invite to join Orgo on Spruce",
				FromName:         "Spruce",
				FromEmailAddress: "from@example.com",
				Body:             "Your invite link is https://example.com/invite [simpleToken]",
				ToEmailAddress:   "someone@example.com",
				TemplateID:       "templateID",
				Transactional:    true,
				TemplateSubstitutions: []*excomms.EmailMessage_Substitution{
					{Key: "{orgname}", Value: "Orgo"},
					{Key: "{inviteurl}", Value: "https://example.com/invite"},
					{Key: "{invitername}", Value: "Inviter"},
					{Key: "{invitecode}", Value: "simpleToken"},
				},
			},
		},
	}))

	// Generate branch URL
	values := map[string]string{
		"invite_token": "simpleToken",
		"client_data":  `{"organization_invite":{"popover":{"title":"Welcome to Spruce!","message":"Inviter has invited you to join them on Spruce.","button_text":"Okay"},"org_id":"org","org_name":"Orgo"}}`,
		"$desktop_url": "https://app.sprucehealth.com/signup?invite=simpleToken&some=other",
		"invite_type":  "COLLEAGUE",
	}
	clientData := make(map[string]interface{}, len(values))
	for k, v := range values {
		clientData[k] = v
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                   "simpleToken",
		OrganizationEntityID:    "org",
		InviterEntityID:         "ent",
		Type:                    models.ColleagueInvite,
		Email:                   "someone@example.com",
		PhoneNumber:             "+15555551212",
		URL:                     "https://example.com/invite",
		Created:                 clk.Now(),
		Values:                  values,
		VerificationRequirement: models.PhoneMatchRequired,
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
	dir := dirmock.New(t)
	branch := branchmock.New(t)
	snsC := mock.NewSNSAPI(t)
	excommsC := excommsmock.New(t)
	settingsC := settingsmock.New(t)
	defer mock.FinishAll(dl, dir, branch, snsC, excommsC, settingsC)
	clk := clock.NewManaged(time.Unix(10000000, 0))
	srv := New(dl, clk, dir, excommsC, settingsC, snsC, branch, "from@example.com", "+1234567890", "eventsTopic", "https://app.sprucehealth.com/signup?some=other", "", "patientInviteTemplateID")

	// Lookup organization
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "org",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: "org", Type: directory.EntityType_ORGANIZATION, Info: &directory.EntityInfo{DisplayName: "Batman Inc."}},
		},
	}, nil))

	// Lookup inviter
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	// Lookup settings
	settingsC.Expect(mock.NewExpectation(settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: "org",
		Keys: []*settings.ConfigKey{
			{
				Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
			},
			{
				Key: invite.ConfigKeyPatientInviteChannelPreference,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: invite.PatientInviteChannelPreferenceEmail,
						},
					},
				},
			},
		},
	}, nil))

	// Generate branch URL
	values := map[string]string{
		"invite_token": "simpleToken",
		"client_data":  `{"patient_invite":{"greeting":{"title":"Welcome to Spruce!","message":"Let's create your account so you can start securely messaging with Batman Inc.","button_text":"Get Started"},"org_id":"org","org_name":"Batman Inc."}}`,
		"$desktop_url": "https://app.sprucehealth.com/signup?invite=simpleToken&some=other",
		"invite_type":  "PATIENT",
	}
	clientData := make(map[string]interface{}, len(values))
	for k, v := range values {
		clientData[k] = v
	}
	branch.Expect(mock.NewExpectation(branch.URL, clientData).WithReturns("https://example.com/invite", nil))

	// Insert invite
	dl.Expect(mock.NewExpectation(dl.InsertInvite, &models.Invite{
		Token:                   "simpleToken",
		OrganizationEntityID:    "org",
		InviterEntityID:         "ent",
		Type:                    models.PatientInvite,
		PhoneNumber:             phiAttributeText,
		Email:                   phiAttributeText,
		URL:                     "https://example.com/invite",
		ParkedEntityID:          "parkedEntityID",
		Created:                 clk.Now(),
		Values:                  values,
		VerificationRequirement: models.PhoneMatchRequired,
	}).WithReturns(nil))

	excommsC.Expect(mock.NewExpectation(excommsC.SendMessage, &excomms.SendMessageRequest{
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Please join Batman Inc. on Spruce",
				FromName:         "Spruce",
				FromEmailAddress: "from@example.com",
				Body:             fmt.Sprintf("Your invite link is %s [%s]", "https://example.com/invite", "simpleToken"),
				ToEmailAddress:   "patient@example.com",
				Transactional:    true,
				TemplateID:       "patientInviteTemplateID",
				TemplateSubstitutions: []*excomms.EmailMessage_Substitution{
					{Key: "{orgname}", Value: "Batman Inc."},
					{Key: "{inviteurl}", Value: "https://example.com/invite"},
					{Key: "{invitecode}", Value: "simpleToken"},
				},
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
			{FirstName: "Alfred", PhoneNumber: "+15555551212", Email: "patient@example.com", ParkedEntityID: "parkedEntityID"},
		},
	})
	test.OK(t, err)
	test.Equals(t, &invite.InvitePatientsResponse{}, ires)
}

func TestLookupInvite(t *testing.T) {
	dl := newMockDAL(t)
	snsC := mock.NewSNSAPI(t)
	defer mock.FinishAll(dl, snsC)
	srv := New(dl, nil, nil, nil, nil, snsC, nil, "", "", "", "", "", "")

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
	res, err := srv.LookupInvite(nil, &invite.LookupInviteRequest{
		InviteToken: "testtoken",
	})
	test.OK(t, err)
	test.Equals(t, &invite.LookupInviteResponse{
		Type: invite.LOOKUP_INVITE_RESPONSE_COLLEAGUE,
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

func TestLookupInvites(t *testing.T) {
	dl := newMockDAL(t)
	snsC := mock.NewSNSAPI(t)
	defer mock.FinishAll(dl, snsC)
	srv := New(dl, nil, nil, nil, nil, snsC, nil, "", "", "", "", "", "")

	dl.Expect(mock.NewExpectation(dl.InvitesForParkedEntityID, "parkedEntityID").WithReturns(
		[]*models.Invite{
			{
				Type:                 models.ColleagueInvite,
				Token:                "testtoken",
				ParkedEntityID:       "parkedEntityID",
				OrganizationEntityID: "org",
				InviterEntityID:      "ent",
				Email:                "someone@example.com",
				PhoneNumber:          "+15555551212",
				Created:              time.Now(),
				Values: map[string]string{
					"foo": "bar",
				},
			},
		}, nil))
	res, err := srv.LookupInvites(nil, &invite.LookupInvitesRequest{
		LookupKeyType: invite.LOOKUP_INVITES_KEY_PARKED_ENTITY_ID,
		Key: &invite.LookupInvitesRequest_ParkedEntityID{
			ParkedEntityID: "parkedEntityID",
		},
	})
	test.OK(t, err)
	test.Equals(t, &invite.LookupInvitesResponse{
		Type: invite.LOOKUP_INVITES_RESPONSE_PATIENT_LIST,
		List: &invite.LookupInvitesResponse_PatientInviteList{
			PatientInviteList: &invite.PatientInviteList{
				PatientInvites: []*invite.PatientInvite{
					{
						OrganizationEntityID: "org",
						InviterEntityID:      "ent",
						Patient: &invite.Patient{
							ParkedEntityID: "parkedEntityID",
							PhoneNumber:    "+15555551212",
						},
					},
				},
			},
		},
	}, res)
}

func TestMarkInviteConsumed(t *testing.T) {
	dl := newMockDAL(t)
	snsC := mock.NewSNSAPI(t)
	defer mock.FinishAll(dl, snsC)
	srv := New(dl, nil, nil, nil, nil, snsC, nil, "", "", "", "", "", "")

	dl.Expect(mock.NewExpectation(dl.DeleteInvite, "testtoken").WithReturns(nil))
	res, err := srv.MarkInviteConsumed(nil, &invite.MarkInviteConsumedRequest{Token: "testtoken"})
	test.OK(t, err)
	test.Equals(t, &invite.MarkInviteConsumedResponse{}, res)
}

type tserver struct {
	server    *server
	finishers []mock.Finisher
}

func TestCreateOrganizationInvite(t *testing.T) {
	orgID := "orgID"
	cases := map[string]struct {
		tserver     *tserver
		in          *invite.CreateOrganizationInviteRequest
		expectedOut *invite.CreateOrganizationInviteResponse
		expectedErr error
	}{
		"Err-OrganizationEntityIDRequired": {
			tserver:     &tserver{server: &server{}},
			in:          &invite.CreateOrganizationInviteRequest{},
			expectedOut: nil,
			expectedErr: grpc.Errorf(codes.InvalidArgument, "Organization Entity ID is required"),
		},
		"Err-OrgNotFound": {
			tserver: func() *tserver {
				dc := dirmock.New(t)
				dc.Expect(mock.NewExpectation(dc.LookupEntities, &directory.LookupEntitiesRequest{
					Key: &directory.LookupEntitiesRequest_EntityID{
						EntityID: orgID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				}).WithReturns(&directory.LookupEntitiesResponse{}, nil))
				return &tserver{
					server: &server{
						directoryClient: dc,
					},
					finishers: []mock.Finisher{dc},
				}
			}(),
			in: &invite.CreateOrganizationInviteRequest{
				OrganizationEntityID: orgID,
			},
			expectedOut: nil,
			expectedErr: errors.Errorf("Expected 1 entity got 0"),
		},
		"Err-BranchGenerationFailure": {
			tserver: func() *tserver {
				dc := dirmock.New(t)
				dc.Expect(mock.NewExpectation(dc.LookupEntities, &directory.LookupEntitiesRequest{
					Key: &directory.LookupEntitiesRequest_EntityID{
						EntityID: orgID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				}).WithReturns(&directory.LookupEntitiesResponse{
					Entities: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							Info: &directory.EntityInfo{
								DisplayName: "DisplayName",
							},
						},
					},
				}, nil))
				md := newMockDAL(t)
				mb := branchmock.New(t)
				clientData, err := clientdata.PatientInviteClientJSON(&directory.Entity{
					Type: directory.EntityType_ORGANIZATION,
					Info: &directory.EntityInfo{
						DisplayName: "DisplayName",
					},
				}, "", "", invite.LOOKUP_INVITE_RESPONSE_ORGANIZATION_CODE)
				test.OK(t, err)
				// Retry 5 times
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("", fmt.Errorf("Foo")))
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("", fmt.Errorf("Foo")))
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("", fmt.Errorf("Foo")))
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("", fmt.Errorf("Foo")))
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("", fmt.Errorf("Foo")))
				return &tserver{
					server: &server{
						dal:             md,
						directoryClient: dc,
						branch:          mb,
					},
					finishers: []mock.Finisher{dc, md, mb},
				}
			}(),
			in: &invite.CreateOrganizationInviteRequest{
				OrganizationEntityID: orgID,
			},
			expectedOut: nil,
			expectedErr: errors.Errorf("Failed to generate branch link and code"),
		},
		"Success": {
			tserver: func() *tserver {
				dc := dirmock.New(t)
				dc.Expect(mock.NewExpectation(dc.LookupEntities, &directory.LookupEntitiesRequest{
					Key: &directory.LookupEntitiesRequest_EntityID{
						EntityID: orgID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				}).WithReturns(&directory.LookupEntitiesResponse{
					Entities: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							Info: &directory.EntityInfo{
								DisplayName: "DisplayName",
							},
						},
					},
				}, nil))
				md := newMockDAL(t)
				mb := branchmock.New(t)
				clientData, err := clientdata.PatientInviteClientJSON(&directory.Entity{
					Type: directory.EntityType_ORGANIZATION,
					Info: &directory.EntityInfo{
						DisplayName: "DisplayName",
					},
				}, "", "", invite.LOOKUP_INVITE_RESPONSE_ORGANIZATION_CODE)
				test.OK(t, err)
				mb.Expect(mock.NewExpectation(mb.URL, map[string]interface{}{
					"invite_token": "simpleToken",
					"client_data":  clientData,
					"invite_type":  string(models.OrganizationCodeInvite),
				}).WithReturns("branckLink", nil))

				clk := clock.NewManaged(time.Now())
				md.Expect(mock.NewExpectation(md.InsertEntityToken, orgID, "simpleToken").WithReturns(nil))
				md.Expect(mock.NewExpectation(md.InsertInvite, &models.Invite{
					Token:                "simpleToken",
					Type:                 models.OrganizationCodeInvite,
					OrganizationEntityID: orgID,
					Created:              clk.Now(),
					URL:                  "branckLink",
					Values: map[string]string{
						"invite_token": "simpleToken",
						"client_data":  clientData,
						"invite_type":  string(models.OrganizationCodeInvite),
					},
				}).WithReturns(nil))
				return &tserver{
					server: &server{
						dal:             md,
						directoryClient: dc,
						branch:          mb,
						clk:             clk,
					},
					finishers: []mock.Finisher{dc, md, mb},
				}
			}(),
			in: &invite.CreateOrganizationInviteRequest{
				OrganizationEntityID: orgID,
			},
			expectedOut: &invite.CreateOrganizationInviteResponse{
				Organization: &invite.OrganizationInvite{
					OrganizationEntityID: orgID,
					Token:                "simpleToken",
				},
			},
			expectedErr: nil,
		},
	}

	for cn, c := range cases {
		out, err := c.tserver.server.CreateOrganizationInvite(context.Background(), c.in)
		test.EqualsCase(t, cn, errors.Cause(c.expectedErr), errors.Cause(err))
		test.EqualsCase(t, cn, c.expectedOut, out)
		mock.FinishAll(c.tserver.finishers...)
	}
}

func TestLookupOrganizationInvites(t *testing.T) {
	orgID := "orgID"
	cases := map[string]struct {
		tserver     *tserver
		in          *invite.LookupOrganizationInvitesRequest
		expectedOut *invite.LookupOrganizationInvitesResponse
		expectedErr error
	}{
		"Success": {
			tserver: func() *tserver {
				dc := dirmock.New(t)
				md := newMockDAL(t)
				mb := branchmock.New(t)
				clk := clock.NewManaged(time.Now())
				md.Expect(mock.NewExpectation(md.TokensForEntity, orgID).WithReturns([]string{"token1", "token2"}, nil))
				md.Expect(mock.NewExpectation(md.InviteForToken, "token1").WithReturns(&models.Invite{
					Type:                 models.OrganizationCodeInvite,
					Token:                "token1",
					OrganizationEntityID: orgID,
				}, nil))
				md.Expect(mock.NewExpectation(md.InviteForToken, "token2").WithReturns(&models.Invite{
					Type:                 models.OrganizationCodeInvite,
					Token:                "token2",
					OrganizationEntityID: orgID,
				}, nil))
				return &tserver{
					server: &server{
						dal:             md,
						directoryClient: dc,
						branch:          mb,
						clk:             clk,
					},
					finishers: []mock.Finisher{dc, md, mb},
				}
			}(),
			in: &invite.LookupOrganizationInvitesRequest{
				OrganizationEntityID: orgID,
			},
			expectedOut: &invite.LookupOrganizationInvitesResponse{
				OrganizationInvites: []*invite.OrganizationInvite{
					{
						OrganizationEntityID: orgID,
						Token:                "token1",
					},
					{
						OrganizationEntityID: orgID,
						Token:                "token2",
					},
				},
			},
			expectedErr: nil,
		},
	}

	for cn, c := range cases {
		out, err := c.tserver.server.LookupOrganizationInvites(context.Background(), c.in)
		test.EqualsCase(t, cn, c.expectedErr, err)
		test.EqualsCase(t, cn, c.expectedOut, out)
		mock.FinishAll(c.tserver.finishers...)
	}
}

func TestModifyOrganizationInvite(t *testing.T) {
	token := "token"
	orgID := "orgID"
	cases := map[string]struct {
		tserver     *tserver
		in          *invite.ModifyOrganizationInviteRequest
		expectedOut *invite.ModifyOrganizationInviteResponse
		expectedErr error
	}{
		"Success": {
			tserver: func() *tserver {
				dc := dirmock.New(t)
				md := newMockDAL(t)
				mb := branchmock.New(t)
				clk := clock.NewManaged(time.Now())
				md.Expect(mock.NewExpectation(md.UpdateInvite, token, &models.InviteUpdate{
					Tags: []string{"tag1", "tag2"},
				}).WithReturns(&models.Invite{
					Type:                 models.OrganizationCodeInvite,
					OrganizationEntityID: orgID,
					Token:                token,
					Tags:                 []string{"tag1", "tag2"},
				}, nil))
				return &tserver{
					server: &server{
						dal:             md,
						directoryClient: dc,
						branch:          mb,
						clk:             clk,
					},
					finishers: []mock.Finisher{dc, md, mb},
				}
			}(),
			in: &invite.ModifyOrganizationInviteRequest{
				Token: token,
				Tags:  []string{"tag1", "tag2"},
			},
			expectedOut: &invite.ModifyOrganizationInviteResponse{
				OrganizationInvite: &invite.OrganizationInvite{
					OrganizationEntityID: orgID,
					Token:                token,
					Tags:                 []string{"tag1", "tag2"},
				},
			},
			expectedErr: nil,
		},
	}

	for cn, c := range cases {
		out, err := c.tserver.server.ModifyOrganizationInvite(context.Background(), c.in)
		test.EqualsCase(t, cn, c.expectedErr, err)
		test.EqualsCase(t, cn, c.expectedOut, out)
		mock.FinishAll(c.tserver.finishers...)
	}
}
