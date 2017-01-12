package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	eventSetupPhoneLine        = "setup_phone_line"
	eventSetupTeamMessaging    = "setup_team_messaging"
	eventSetupTelemedicine     = "setup_telemedicine"
	eventSetupPatientMessaging = "setup_patient_messaging"
	eventSetupAnsweringService = "setup_answering_service"

	stepProvisionedPhoneNumber = 1
	stepAnsweringService       = 2
	stepTeamMessaging          = 4
	stepTelemedicine           = 8
	stepPatientMessaging       = 16
	stepPhoneLine              = 32
)

// CreateOnboardingThread create a new onboarding thread
func (s *threadsServer) CreateOnboardingThread(ctx context.Context, in *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID is required")
	}

	phoneSetupURL := deeplink.PostEventURL(s.webDomain, eventSetupPhoneLine, map[string][]string{
		"refresh_thread": {"1"},
		"org_id":         {in.OrganizationID},
	})
	teamMessagingURL := deeplink.PostEventURL(s.webDomain, eventSetupTeamMessaging, map[string][]string{
		"refresh_thread": {"1"},
		"org_id":         {in.OrganizationID},
	})
	telemedicineURL := deeplink.PostEventURL(s.webDomain, eventSetupTelemedicine, map[string][]string{
		"refresh_thread": {"1"},
		"org_id":         {in.OrganizationID},
	})
	patientMessagingURL := deeplink.PostEventURL(s.webDomain, eventSetupPatientMessaging, map[string][]string{
		"refresh_thread": {"1"},
		"org_id":         {in.OrganizationID},
	})

	supportThreads, err := s.dal.ThreadsForOrg(ctx, in.OrganizationID, models.ThreadTypeSupport, 1)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Really should be exactly one, but to not blow up for our own support forum only err when none exist.
	if len(supportThreads) < 1 {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Expected at least 1 support thread for org %s", in.OrganizationID)
	}
	supportThread := supportThreads[0]

	supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, in.OrganizationID, supportThread.ID.String())

	msgBML := bml.BML{
		"👋 Hi! Spruce can do many things to help you provide great care. To learn more about what Spruce can do for you, tap any item below and we’ll guide you through it...\n\n ",

		&bml.Anchor{HREF: phoneSetupURL, Text: "📞  2nd phone line"}, "\n\n",

		&bml.Anchor{HREF: patientMessagingURL, Text: "💬  Patient messaging"}, "\n\n",

		&bml.Anchor{HREF: teamMessagingURL, Text: "👥  Team chat & care coordination"}, "\n\n",

		&bml.Anchor{HREF: telemedicineURL, Text: "⚡  Telemedicine"}, "\n\n",

		"If you have questions at any time, just ",
		&bml.Anchor{HREF: supportThreadURL, Text: "message us"},
		" or check out our ",
		&bml.Anchor{HREF: "https://intercom.help/spruce", Text: "Knowledge Center"}, ".",
	}
	msg, err := msgBML.Format()
	if err != nil {
		return nil, errors.Errorf("Failed to format setup thread BML: %s", err)
	}
	summary, err := models.SummaryFromText("Setup: " + msg)
	if err != nil {
		return nil, errors.Errorf("Failed to generate summary: %s", err)
	}

	var threadID models.ThreadID
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		threadID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:     in.OrganizationID,
			PrimaryEntityID:    in.PrimaryEntityID,
			LastMessageSummary: summary,
			Type:               models.ThreadTypeSetup,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if err := dl.AddThreadMembers(ctx, threadID, []string{in.OrganizationID}); err != nil {
			return errors.Trace(err)
		}
		if _, err := dl.PostMessage(ctx, &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.PrimaryEntityID,
			Internal:     false,
			Text:         msg,
			Summary:      summary,
		}); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.CreateSetupThreadState(ctx, threadID, in.OrganizationID))
	}); err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, errors.Errorf("thread %s not found", threadID)
	}
	if _, err := s.updateSavedQueriesAddThread(ctx, threads[0], []string{in.OrganizationID}); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query when adding thread: %s", threads[0].ID)
	}

	th, err := transformThreadToResponse(threads[0], false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateOnboardingThreadResponse{
		Thread: th,
	}, nil
}

// OnboardingThreadEvent
func (s *threadsServer) OnboardingThreadEvent(ctx context.Context, in *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error) {
	var state *models.SetupThreadState
	var err error
	switch in.LookupByType {
	case threading.ONBOARDING_THREAD_LOOKUP_BY_THREAD_ID:
		id, err := models.ParseThreadID(in.GetThreadID())
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread ID '%s'", in.GetThreadID())
		}
		state, err = s.dal.SetupThreadState(ctx, id)
	case threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID:
		state, err = s.dal.SetupThreadStateForEntity(ctx, in.GetEntityID())
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown lookup by type %s", in.LookupBy)
	}
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Onboarding state not found")
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{state.ThreadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, errors.Errorf("thread %q not found", state.ThreadID)
	}
	setupThread := threads[0]

	supportThreads, err := s.dal.ThreadsForOrg(ctx, setupThread.OrganizationID, models.ThreadTypeSupport, 1)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Really should be exactly one, but to not blow up for our own support forum only err when none exist.
	if len(supportThreads) < 1 {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Expected at least 1 support thread for org %s", setupThread.OrganizationID)
	}
	supportThread := supportThreads[0]

	var newStepBit int
	var msgBML bml.BML
	switch in.EventType {
	case threading.ONBOARDING_THREAD_EVENT_TYPE_PROVISIONED_PHONE:
		if state.Step&stepProvisionedPhoneNumber == 0 {
			// Second phone line
			pn, err := phone.ParseNumber(in.GetProvisionedPhone().PhoneNumber)
			if err != nil {
				return nil, grpc.Errorf(codes.InvalidArgument, "Invalid phone number '%s' for org %s", in.GetProvisionedPhone().PhoneNumber, setupThread.OrganizationID)
			}
			supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
			prettyPhone, err := pn.Format(phone.Pretty)
			if err != nil {
				golog.ContextLogger(ctx).Errorf("Failed to format provisioned number '%s' for org %s", pn, setupThread.OrganizationID)
				prettyPhone = pn.String()
			}
			msgBML = bml.BML{
				`💥  ` + prettyPhone + " is your Spruce number.\n\n",
				"To place a call from you Spruce number:\n",
				"1. Return to the home screen and press the ➕  button\n",
				"2. Select ‘Dialpad’\n",
				"3. Enter the number you’d like to call or select a number from your phone's contacts\n",
				"To manage your Spruce number return to the home screen, tap the settings icon and select your number from the menu.\n\n",
				"If you’d like to learn more about using your Spruce number, visit our ",
				&bml.Anchor{HREF: "https://intercom.help/spruce/your-professional-phone-number", Text: "phone guide"},
				" or ", &bml.Anchor{HREF: supportThreadURL, Text: "message us"}, ".",
			}
			newStepBit = stepProvisionedPhoneNumber
		}
	case threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP:
		ev := in.GetGenericSetup()
		switch ev.Name {
		case eventSetupAnsweringService:
			if state.Step&stepAnsweringService == 0 {
				phoneSetupURL := deeplink.OrgSettingsPhoneURL(s.webDomain, setupThread.OrganizationID)
				supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
				msgBML = bml.BML{
					`As a paid feature your Spruce line can triage patient voicemails, notifying you`,
					" via text when an urgent voicemail is received. You can also add teammates to create an on-call rotation.\n\n",
					`To do this, first `, &bml.Anchor{HREF: phoneSetupURL, Text: "set up your Spruce number"},
					` if you haven’t already. Then tell us in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"},
					` that you would like to enable the answering service feature.`,
				}
				newStepBit = stepAnsweringService
			}
		case eventSetupPhoneLine:
			if state.Step&stepPhoneLine == 0 {
				phoneSetupURL := deeplink.OrgSettingsPhoneURL(s.webDomain, setupThread.OrganizationID)
				msgBML = bml.BML{
					"Create a second phone line to make calls to your patients without disclosing your personal number.\n\n",
					&bml.Anchor{HREF: phoneSetupURL, Text: "📱  Claim your Spruce Number now"}, "\n\n",
					"or...\n\n",
					&bml.Anchor{HREF: "https://intercom.help/spruce/your-professional-phone-number/phone-basics/setting-up-your-spruce-number", Text: "📖 Learn more about how it works"},
				}
				newStepBit = stepPhoneLine
			}

		case eventSetupPatientMessaging:
			if state.Step&stepPatientMessaging == 0 {
				msgBML = bml.BML{
					"Send and receive secure messages or standard SMS messages and emails (when appropriate). It’s free to try for 30 days!\n\n",
					&bml.Anchor{HREF: "https://vimeo.com/183376736", Text: "Check out this video overview"}, " of the ins and outs of patient messaging on Spruce, then start a new patient conversation in a few easy steps:\n\n",
					"1. Return to the home screen and press the ➕ button\n",
					"2. Select 👤 Patient Conversation\n",
					"3. Choose 🔒 Secure Conversations for conversations involving protected health information (PHI)\n",
					"4. Or choose 💬 Standard Conversations to send traditional SMS or email messages\n\n",
					"To learn more about messaging patients using Spruce, ", &bml.Anchor{HREF: "https://intercom.help/spruce/getting-started-with-spruce/quick-set-up-guides/patient-conversation-basics", Text: "check out this guide"}, " we put together.",
				}
				newStepBit = stepPatientMessaging
			}
		case eventSetupTeamMessaging:
			if state.Step&stepTeamMessaging == 0 {
				inviteURL := deeplink.OrgColleagueInviteURL(s.webDomain, setupThread.OrganizationID)
				msgBML = bml.BML{
					"🙌 Spruce is built for teams! To invite a teammate to join your practice, return to the home screen, tap the settings icon and select ", &bml.Anchor{HREF: inviteURL, Text: "Invite Teammates"}, ".\n\n",
					"When you invite a teammate to join your Spruce organization you unlock:\n",
					"📥  A Shared Team Inbox - keep everyone in sync with one inbox that gives all teammates the ability to see and respond to incoming patient communication\n",
					"🔒  Secure Team Chats - coordinate care in a private team-only conversation\n",
					"📝  Internal Notes and @Pages - Tap ‘Internal’ to create “sticky notes” in patient conversations that are only visible to teammates. Use the ‘@’ sign to explicitly notify a teammate to something important\n\n",
					&bml.Anchor{HREF: "https://vimeo.com/176232003", Text: "See team chat in action"}, " or ", &bml.Anchor{HREF: "https://intercom.help/spruce/getting-started-with-spruce/quick-set-up-guides/inviting-teammates", Text: "visit our Knowledge Center"},
					" to learn more about adding teammates to your practice.",
				}
				newStepBit = stepTeamMessaging
			}
		case eventSetupTelemedicine:
			if state.Step&stepTelemedicine == 0 {
				msgBML = bml.BML{
					"✨ With Spruce’s Digital Practice plan you can provide care outside the exam room with video visits, Spruce visits (asynchronous clinical question sets), care plans and mobile billpay. It’s free to try for 30 days!\n\n",
					"The best way to learn about Spruce’s telemedicine features is to experience them first hand. Fill out this quick survey so we can customize a test patient which will be added your account within 24 hours (or Monday if it's the weekend).\n\n",
					&bml.Anchor{HREF: "https://sprucehealthsurvey.typeform.com/to/oY215t", Text: "✍  Fill out the survey for a test patient"}, "\n\n",
					&bml.Anchor{HREF: "https://vimeo.com/179789289", Text: "📚  Check out this video to see telemedicine in action"},
				}
				newStepBit = stepTelemedicine
			}
		default:
			return nil, errors.Errorf("Unhandled onboarding setup event %q for org %s", in.Event, setupThread.OrganizationID)
		}
	default:
		golog.Debugf("Unhandled onboarding thread event '%s'", in.Event)
	}
	if msgBML != nil {
		msg, err := msgBML.Format()
		if err != nil {
			return nil, errors.Errorf("invalid onboarding message BML: %s", err)
		}
		var summary string
		if msg != "" {
			summary, err = models.SummaryFromText("Setup: " + msg)
			if err != nil {
				return nil, errors.Errorf("Failed to generate summary for event %s", in.EventType)
			}
		}
		if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
			// Query the state again so we can lock it and recheck the step to make sure there's no concurrent update
			state, err := dl.SetupThreadState(ctx, setupThread.ID, dal.ForUpdate)
			if err != nil {
				return errors.Trace(err)
			}
			if state.Step&newStepBit != 0 {
				// Not an error, someone just got here before us
				return nil
			}
			if err := dl.UpdateSetupThreadState(ctx, setupThread.ID, &dal.SetupThreadStateUpdate{Step: ptr.Int(state.Step | newStepBit)}); err != nil {
				return errors.Trace(err)
			}
			_, err = dl.PostMessage(ctx, &dal.PostMessageRequest{
				ThreadID:     setupThread.ID,
				FromEntityID: setupThread.PrimaryEntityID,
				Text:         msg,
				Summary:      summary,
			})
			return errors.Trace(err)
		}); err != nil {
			return nil, errors.Trace(err)
		}
	}

	threads, err = s.dal.Threads(ctx, []models.ThreadID{state.ThreadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "thread not found")
	}
	thread := threads[0]

	if _, err := s.updateSavedQueriesForThread(ctx, thread); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}

	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.OnboardingThreadEventResponse{Thread: th}, nil
}
