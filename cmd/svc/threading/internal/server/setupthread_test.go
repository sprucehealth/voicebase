package server

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mockdirectory "github.com/sprucehealth/backend/svc/directory/mock"
	mockmedia "github.com/sprucehealth/backend/svc/media/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestCreateOnboardingThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, mm, dir)
	clk := clock.New()

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, nil, mm, nil, nil, nil, nil, "WEBDOMAIN")

	now := clk.Now()

	thid, err := models.NewThreadID()
	test.OK(t, err)
	sqid, err := models.NewSavedQueryID()
	test.OK(t, err)
	supportTID, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.ThreadsForOrg, "o1", models.ThreadTypeSupport, 1).WithReturns(
		[]*models.Thread{
			{ID: supportTID, Type: models.ThreadTypeSupport},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateThread, &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "Setup: 👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...  📞 2nd phone line 💬 Patient messaging 👥 Team chat & care coordination ⚡ Telemedicine If you have questions at any time, just message us or check out our Knowledge Center.",
		Type:               models.ThreadTypeSetup,
	}).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, thid, []string{"o1"}))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e2",
		Internal:     false,
		Text:         "👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...\n\n <a href=\"https://WEBDOMAIN/post_event?name=setup_phone_line&amp;org_id=o1&amp;refresh_thread=1\">📞  2nd phone line</a>\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_patient_messaging&amp;org_id=o1&amp;refresh_thread=1\">💬  Patient messaging</a>\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_team_messaging&amp;org_id=o1&amp;refresh_thread=1\">👥  Team chat &amp; care coordination</a>\n\n<a href=\"https://WEBDOMAIN/post_event?name=setup_telemedicine&amp;org_id=o1&amp;refresh_thread=1\">⚡  Telemedicine</a>\n\nIf you have questions at any time, just <a href=\"https://WEBDOMAIN/org/o1/thread/" + supportTID.String() + "\">message us</a> or check out our <a href=\"https://intercom.help/spruce\">Knowledge Center</a>.",
		Summary:      "Setup: 👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...  📞 2nd phone line 💬 Patient messaging 👥 Team chat & care coordination ⚡ Telemedicine If you have questions at any time, just message us or check out our Knowledge Center.",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateOnboardingState, thid, "o1").WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{
		{
			ID:                   thid,
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageSummary:   "Setup: 👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...  📞 2nd phone line 💬 Patient messaging 👥 Team chat & care coordination ⚡ Telemedicine If you have questions at any time, just message us or check out our Knowledge Center.",
			LastMessageTimestamp: now,
			Created:              now,
		},
	}, nil))

	// Update saved query indexes

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"o1"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o1",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{ID: "e1", Type: directory.EntityType_INTERNAL},
				},
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e1").WithReturns([]*models.SavedQuery{{ID: sqid, EntityID: "e1", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: thid, SavedQueryID: sqid, Timestamp: now}}))

	res, err := srv.CreateOnboardingThread(context.Background(), &threading.CreateOnboardingThreadRequest{
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
			LastMessageSummary:   "Setup: 👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...  📞 2nd phone line 💬 Patient messaging 👥 Team chat & care coordination ⚡ Telemedicine If you have questions at any time, just message us or check out our Knowledge Center.",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
		},
	}, res)
}

func TestOnboardingThreadEvent_PROVISIONED_PHONE(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, mm, dir)
	clk := clock.New()

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, nil, mm, nil, nil, nil, nil, "WEBDOMAIN")

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
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(stepProvisionedPhoneNumber)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "💥  (415) 555-1212 is your Spruce number.\n\nTo place a call from you Spruce number:\n\n1. Return to the home screen and press the ➕ button\n2. Select ‘Dialpad’\n3. Enter the number you’d like to call or select a number from your phone&#39;s contacts\n\nTo manage your Spruce number return to the home screen, tap the settings icon and select your number from the menu.\n\nIf you’d like to learn more about using your Spruce number, visit our <a href=\"https://intercom.help/spruce/your-professional-phone-number\">phone guide</a> or <a href=\"https://WEBDOMAIN/org/org/thread/" + supportTID.String() + "\">message us</a>.",
		Summary:  "Setup: 💥 (415) 555-1212 is your Spruce number. To place a call from you Spruce number: 1. Return to the home screen and press the ➕ button 2. Select ‘Dialpad’ 3. Enter the number you’d like to call or select a number from your phone's contacts To manage your Spruce number return to the home screen, tap the settings icon and select your number from the menu. If you’d like to learn more about using your Spruce number, visit our phone guide or message us.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	// Update saved query indexes
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, setupTID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, setupTID))

	res, err := srv.OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
		LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_PROVISIONED_PHONE,
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

	// Update saved query indexes
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, setupTID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, setupTID))

	res, err = srv.OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
		LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_PROVISIONED_PHONE,
		Event: &threading.OnboardingThreadEventRequest_ProvisionedPhone{
			ProvisionedPhone: &threading.ProvisionedPhoneEvent{
				PhoneNumber: "+14155551212",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}

func TestOnboardingThreadEvent_GENERIC_SETUP_eventSetupPatientMessaging(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	mm := mockmedia.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, nil, nil, nil, nil, "WEBDOMAIN")

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
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(stepPatientMessaging)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "Send and receive secure messages or standard SMS messages and emails (when appropriate). It’s free to try for 30 days!\n\n<a href=\"https://vimeo.com/183376736\">Check out this video overview</a> of the ins and outs of patient messaging on Spruce, then start a new patient conversation in a few easy steps:\n\n1. Return to the home screen and press the ➕ button\n2. Select 👤 Patient Conversation\n3. Choose 🔒 Secure Conversations for conversations involving protected health information (PHI)\n4. Or choose 💬 Standard Conversations to send traditional SMS or email messages\n\nTo learn more about messaging patients using Spruce, <a href=\"https://intercom.help/spruce/getting-started-with-spruce/quick-set-up-guides/patient-conversation-basics\">check out this guide</a> we put together.",
		Summary:  "Setup: Send and receive secure messages or standard SMS messages and emails (when appropriate). It’s free to try for 30 days! Check out this video overview of the ins and outs of patient messaging on Spruce, then start a new patient conversation in a few easy steps: 1. Return to the home screen and press the ➕ button 2. Select 👤 Patient Conversation 3. Choose 🔒 Secure Conversations for conversations involving protected health information (PHI) 4. Or choose 💬 Standard Conversations to send traditional SMS or email messages To learn more about messaging patients using Spruce, check out this guide we put together.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	// Update saved query indexes
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, setupTID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, setupTID))

	res, err := srv.OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
		LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP,
		Event: &threading.OnboardingThreadEventRequest_GenericSetup{
			GenericSetup: &threading.GenericSetupEvent{
				Name: eventSetupPatientMessaging,
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
	mm := mockmedia.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, nil, nil, nil, nil, "WEBDOMAIN")

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
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(stepTeamMessaging)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "🙌 Spruce is built for teams! To invite a teammate to join your practice, return to the home screen, tap the settings icon and select <a href=\"https://WEBDOMAIN/org/org/invite\">Invite Teammates</a>.\n\nWhen you invite a teammate to join your Spruce organization you unlock:\n\n📥  A Shared Team Inbox - keep everyone in sync with one inbox that gives all teammates the ability to see and respond to incoming patient communication\n🔒  Secure Team Chats - coordinate care in a private team-only conversation\n📝  Internal Notes and @Pages - Tap ‘Internal’ to create “sticky notes” in patient conversations that are only visible to teammates. Use the ‘@’ sign to explicitly notify a teammate to something important\n\n<a href=\"https://vimeo.com/176232003\">See team chat in action</a> or <a href=\"https://intercom.help/spruce/getting-started-with-spruce/quick-set-up-guides/inviting-teammates\">visit our Knowledge Center</a> to learn more about adding teammates to your practice.",
		Summary:  "Setup: 🙌 Spruce is built for teams! To invite a teammate to join your practice, return to the home screen, tap the settings icon and select Invite Teammates. When you invite a teammate to join your Spruce organization you unlock: 📥 A Shared Team Inbox - keep everyone in sync with one inbox that gives all teammates the ability to see and respond to incoming patient communication 🔒 Secure Team Chats - coordinate care in a private team-only conversation 📝 Internal Notes and @Pages - Tap ‘Internal’ to create “sticky notes” in patient conversations that are only visible to teammates. Use the ‘@’ sign to explicitly notify a teammate to something important See team chat in action or visit our Knowledge Center to learn more about adding teammates to your practice.",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	// Update saved query indexes
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, setupTID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, setupTID))

	res, err := srv.OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
		LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP,
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
	mm := mockmedia.New(t)
	defer mm.Finish()
	clk := clock.New()
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, nil, mm, nil, nil, nil, nil, "WEBDOMAIN")

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
	dl.Expect(mock.NewExpectation(dl.UpdateSetupThreadState, setupTID, &dal.SetupThreadStateUpdate{Step: ptr.Int(stepTelemedicine)}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		Text:     "✨ With Spruce’s Digital Practice plan you can provide care outside the exam room with video visits, Spruce visits (asynchronous clinical question sets), care plans and mobile billpay. It’s free to try for 30 days!\n\nThe best way to learn about Spruce’s telemedicine features is to experience them first hand. Fill out this quick survey so we can customize a test patient which will be added your account within 24 hours (or Monday if it&#39;s the weekend).\n\n<a href=\"https://sprucehealthsurvey.typeform.com/to/oY215t\">✍  Fill out the survey for a test patient</a>\n\n<a href=\"https://vimeo.com/179789289\">📚  Check out this video to see telemedicine in action</a>",
		Summary:  "Setup: ✨ With Spruce’s Digital Practice plan you can provide care outside the exam room with video visits, Spruce visits (asynchronous clinical question sets), care plans and mobile billpay. It’s free to try for 30 days! The best way to learn about Spruce’s telemedicine features is to experience them first hand. Fill out this quick survey so we can customize a test patient which will be added your account within 24 hours (or Monday if it's the weekend). ✍ Fill out the survey for a test patient 📚 Check out this video to see telemedicine in action",
		ThreadID: setupTID,
	}).WithReturns(&models.ThreadItem{}, nil))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{setupTID}).WithReturns([]*models.Thread{{ID: setupTID, OrganizationID: "org"}}, nil))

	// Update saved query indexes
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, setupTID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, setupTID))

	res, err := srv.OnboardingThreadEvent(context.Background(), &threading.OnboardingThreadEventRequest{
		LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
		LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
			EntityID: "org",
		},
		EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP,
		Event: &threading.OnboardingThreadEventRequest_GenericSetup{
			GenericSetup: &threading.GenericSetupEvent{
				Name: eventSetupTelemedicine,
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, setupTID.String(), res.Thread.ID)
}
