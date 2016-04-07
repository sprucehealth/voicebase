package server

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func TestCreateOnboardingThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, "WEBDOMAIN")

	now := clk.Now()

	thid, err := models.NewThreadID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.CreateThread, &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "Setup: Welcome to Spruce! How would you like to use Spruce? Second phone line to exchange calls and texts with patients, without disclosing your personal number Automated answering service that takes and transcribes urgent voicemails and notifies you Secure, HIPAA-compliant collaboration and messaging with teammates about care Digital care and telemedicine with patients",
		Type:               models.ThreadTypeSetup,
	}).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e2",
		Internal:     false,
		Text:         "Welcome to Spruce! How would you like to use Spruce?\n\n<a href=\"https://WEBDOMAIN/org/o1/settings/phone\">Second phone line</a> to exchange calls and texts with patients, without disclosing your personal number\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_answering_service&amp;org_id=o1&amp;refresh_thread=1\">Automated answering service</a> that takes and transcribes urgent voicemails and notifies you\n\nSecure, HIPAA-compliant collaboration and <a href=\"https://WEBDOMAIN/post_event?name=setup_team_messaging&amp;org_id=o1&amp;refresh_thread=1\">messaging with teammates</a> about care\n\nDigital care and <a href=\"https://WEBDOMAIN/post_event?name=setup_telemedicine&amp;org_id=o1&amp;refresh_thread=1\">telemedicine</a> with patients",
		Summary:      "Setup: Welcome to Spruce! How would you like to use Spruce? Second phone line to exchange calls and texts with patients, without disclosing your personal number Automated answering service that takes and transcribes urgent voicemails and notifies you Secure, HIPAA-compliant collaboration and messaging with teammates about care Digital care and telemedicine with patients",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateOnboardingState, thid, "o1").WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e2",
		LastMessageSummary:   "Setup: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"Skip\" to get it later",
		LastMessageTimestamp: now,
		Created:              now,
	}, nil))

	res, err := srv.CreateOnboardingThread(nil, &threading.CreateOnboardingThreadRequest{
		OrganizationID:  "o1",
		PrimaryEntityID: "e2",
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateOnboardingThreadResponse{
		Thread: &threading.Thread{
			ID:                   thid.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "Setup: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"Skip\" to get it later",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
		},
	}, res)
}

func TestOnboardingThreadEvent_PROVISIONED_PHONE(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(1)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "Success! You can now use your Spruce number (415) 555-1212 for calls, texts, and voicemails with patients. If you have any questions about using Spruce, just message us in <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">Spruce Support</a> and we’ll reply to help you.",
		Summary:  "Setup: Success! You can now use your Spruce number (415) 555-1212 for calls, texts, and voicemails with patients. If you have any questions about using Spruce, just message us in Spruce Support and we’ll reply to help you.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))

	res, err := srv.OnboardingThreadEvent(nil, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.OnboardingThreadEventRequest_PROVISIONED_PHONE,
		Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
			ProvisionedPhone: &threading.ProvisionedPhoneEvent{
				PhoneNumber: "+14155551212",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)

	// Case where setup thread isn't at step 0

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 1}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))

	res, err = srv.OnboardingThreadEvent(nil, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.OnboardingThreadEventRequest_PROVISIONED_PHONE,
		Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
			ProvisionedPhone: &threading.ProvisionedPhoneEvent{
				PhoneNumber: "+14155551212",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}

func TestOnboardingThreadEvent_GENERIC_SETUP_eventSetupAnsweringService(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(2)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "Great! First let’s <a href=\"https://WEBDOMAIN/org/org/settings/phone\">set up your Spruce number</a> if you haven’t already. Voicemails left with your Spruce number will be automatically transcribed, and you will receive a notifications on your phone. We can also enable more advanced configurations, such as custom greetings, special handling for urgent voicemails, and after-hours oncall rotations. Just send us a message in <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">Spruce Support</a> with the configuration you would like, and someone from Spruce will reply to help you.",
		Summary:  "Setup: Great! First let’s set up your Spruce number if you haven’t already. Voicemails left with your Spruce number will be automatically transcribed, and you will receive a notifications on your phone. We can also enable more advanced configurations, such as custom greetings, special handling for urgent voicemails, and after-hours oncall rotations. Just send us a message in Spruce Support with the configuration you would like, and someone from Spruce will reply to help you.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))

	res, err := srv.OnboardingThreadEvent(nil, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.OnboardingThreadEventRequest_GENERIC_SETUP,
		Event: &threading.OnboardingThreadEventRequest_GenericSetup{
			GenericSetup: &threading.GenericSetupEvent{
				Name: eventSetupAnsweringService,
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}

func TestOnboardingThreadEvent_GENERIC_SETUP_eventSetupTeamMessaging(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(3)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "After <a href=\"https://WEBDOMAIN/org/org/invite\">adding teammates</a>, you can start a new team conversation from the home screen and message 1:1 or in group chats. You can also collaborate and make notes within the context of patient conversations (patients won’t see this activity, but your teammates will).",
		Summary:  "Setup: After adding teammates, you can start a new team conversation from the home screen and message 1:1 or in group chats. You can also collaborate and make notes within the context of patient conversations (patients won’t see this activity, but your teammates will).",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))

	res, err := srv.OnboardingThreadEvent(nil, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.OnboardingThreadEventRequest_GENERIC_SETUP,
		Event: &threading.OnboardingThreadEventRequest_GenericSetup{
			GenericSetup: &threading.GenericSetupEvent{
				Name: eventSetupTeamMessaging,
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}

func TestOnboardingThreadEvent_GENERIC_SETUP_eventSetupTelemedicine(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(4)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "Interested in engaging patients digitally with e-visits, video calls, care plans (including eprescribing), mobile billpay, appointment reminders, and satisfaction surveys? Digital care on Spruce enables you to simultaneously offer a standout patient experience and streamline your practice efficiency. The Digital Practice offering on Spruce is coming soon: just message us in Spruce Support if you would like to be a part of the private beta.",
		Summary:  "Setup: Interested in engaging patients digitally with e-visits, video calls, care plans (including eprescribing), mobile billpay, appointment reminders, and satisfaction surveys? Digital care on Spruce enables you to simultaneously offer a standout patient experience and streamline your practice efficiency. The Digital Practice offering on Spruce is coming soon: just message us in Spruce Support if you would like to be a part of the private beta.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, setupTID).WithReturns(&models.Thread{ID: setupTID, OrganizationID: "org"}, nil))

	res, err := srv.OnboardingThreadEvent(nil, &threading.OnboardingThreadEventRequest{
		LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.OnboardingThreadEventRequest_GENERIC_SETUP,
		Event: &threading.OnboardingThreadEventRequest_GenericSetup{
			GenericSetup: &threading.GenericSetupEvent{
				Name: eventSetupTelemedicine,
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}
