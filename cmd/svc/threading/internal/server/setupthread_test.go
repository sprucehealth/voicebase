package server

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	mock_media "github.com/sprucehealth/backend/svc/media/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func TestCreateOnboardingThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, "WEBDOMAIN")

	now := clk.Now()

	thid, err := models.NewThreadID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.CreateThread, &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "Setup: Welcome! How would you like to use Spruce? (You can tap on multiple options) Second phone line for calls and texts with patients, without disclosing your personal number. Automated answering service that transcribes urgent voicemails and notifies you. Secure team chat and care coordination. Digital care and telemedicine.",
		Type:               models.ThreadTypeSetup,
	}).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e2",
		Internal:     false,
		Text:         "Welcome! How would you like to use Spruce? (You can tap on multiple options)\n\n<a href=\"https://WEBDOMAIN/org/o1/settings/phone\">Second phone line</a> for calls and texts with patients, without disclosing your personal number.\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_answering_service&amp;org_id=o1&amp;refresh_thread=1\">Automated answering service</a> that transcribes urgent voicemails and notifies you.\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_team_messaging&amp;org_id=o1&amp;refresh_thread=1\">Secure team chat and care coordination</a>.\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_telemedicine&amp;org_id=o1&amp;refresh_thread=1\">Digital care and telemedicine</a>.",
		Summary:      "Setup: Welcome! How would you like to use Spruce? (You can tap on multiple options) Second phone line for calls and texts with patients, without disclosing your personal number. Automated answering service that transcribes urgent voicemails and notifies you. Secure team chat and care coordination. Digital care and telemedicine.",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateOnboardingState, thid, "o1").WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{
		{
			ID:                   thid,
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageSummary:   "Setup: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"Skip\" to get it later",
			LastMessageTimestamp: now,
			Created:              now,
		},
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
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(1)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "You can now use your Spruce number (415) 555-1212 for calls, texts, and voicemails with patients. Just message us in <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">Spruce Support</a> if you have any questions or problems.",
		Summary:  "Setup: You can now use your Spruce number (415) 555-1212 for calls, texts, and voicemails with patients. Just message us in Spruce Support if you have any questions or problems.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

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
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

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
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(2)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "As a paid feature your Spruce line can triage and transcribe patient voicemails, notifying you via text when an urgent voicemail is received. You can also add teammates to create an on-call rotation.\n\nTo do this, first <a href=\"https://WEBDOMAIN/org/org/settings/phone\">set up your Spruce number</a> if you haven’t already. Then tell us in <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">Spruce Support</a> that you would like to enable the answering service feature.",
		Summary:  "Setup: As a paid feature your Spruce line can triage and transcribe patient voicemails, notifying you via text when an urgent voicemail is received. You can also add teammates to create an on-call rotation. To do this, first set up your Spruce number if you haven’t already. Then tell us in Spruce Support that you would like to enable the answering service feature.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

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
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(4)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "After <a href=\"https://WEBDOMAIN/org/org/invite\">adding teammates</a>, you can start a new team conversation from the home screen and message 1:1 or in group chats.\n\nYou can also collaborate and make notes within patient conversations (patients won’t see this activity, but your teammates will).",
		Summary:  "Setup: After adding teammates, you can start a new team conversation from the home screen and message 1:1 or in group chats. You can also collaborate and make notes within patient conversations (patients won’t see this activity, but your teammates will).",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

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
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, "WEBDOMAIN")

	setupTID, err := models.NewThreadID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SetupThreadStateForEntity, "org").WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))
	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "org", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SetupThreadState, setupTID).WithReturns(&models.SetupThreadState{ThreadID: setupTID, Step: 0}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(8)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "Interested in engaging patients digitally with virtual visits, video calls, care plans (including e-prescribing), mobile payment, appointment reminders, and satisfaction surveys?\n\nDigital care on Spruce enables you to offer a standout patient experience and streamline your practice efficiency. The Digital Practice offering on Spruce is coming soon: message us in <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">Spruce Support</a> if you would like to be a part of the private beta.",
		Summary:  "Setup: Interested in engaging patients digitally with virtual visits, video calls, care plans (including e-prescribing), mobile payment, appointment reminders, and satisfaction surveys? Digital care on Spruce enables you to offer a standout patient experience and streamline your practice efficiency. The Digital Practice offering on Spruce is coming soon: message us in Spruce Support if you would like to be a part of the private beta.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

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
