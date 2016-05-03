package server

import (
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

const (
	eventSetupAnsweringService = "setup_answering_service"
	eventSetupTeamMessaging    = "setup_team_messaging"
	eventSetupTelemedicine     = "setup_telemedicine"
)

// CreateOnboardingThread create a new onboarding thread
func (s *threadsServer) CreateOnboardingThread(ctx context.Context, in *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}

	phoneSetupURL := deeplink.OrgSettingsPhoneURL(s.webDomain, in.OrganizationID)
	answeringServiceURL := deeplink.PostEventURL(s.webDomain, eventSetupAnsweringService, map[string][]string{
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
	msgBML := bml.BML{
		"Welcome to Spruce! How would you like to use Spruce?\n\n",

		&bml.Anchor{HREF: phoneSetupURL, Text: "Second phone line"},
		" to exchange calls and texts with patients, without disclosing your personal number\n\n",

		&bml.Anchor{HREF: answeringServiceURL, Text: "Automated answering service"},
		" that takes and transcribes urgent voicemails and notifies you\n\n",

		"Secure, HIPAA-compliant collaboration and ", &bml.Anchor{HREF: teamMessagingURL, Text: "messaging with teammates"}, " about care\n\n",

		"Digital care and ", &bml.Anchor{HREF: telemedicineURL, Text: "telemedicine"}, " with patients",
	}
	msg, err := msgBML.Format()
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "Failed to format setup thread BML: %s", err)
	}
	summary, err := models.SummaryFromText("Setup: " + msg)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "Failed to generate summary: %s", err)
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
		return nil, grpcErrorf(codes.Internal, errors.Trace(err).Error())
	}
	thread, err := s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, false)
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
	case threading.OnboardingThreadEventRequest_THREAD_ID:
		id, err := models.ParseThreadID(in.GetThreadID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread ID")
		}
		state, err = s.dal.SetupThreadState(ctx, id)
	case threading.OnboardingThreadEventRequest_ENTITY_ID:
		state, err = s.dal.SetupThreadStateForEntity(ctx, in.GetEntityID())
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown lookup by type %s", in.LookupBy)
	}
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Onboarding state not found")
	}
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	// The events only have an effect when the thread is in its initial (first message) state
	if state.Step == 0 {
		setupThread, err := s.dal.Thread(ctx, state.ThreadID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		supportThreads, err := s.dal.ThreadsForOrg(ctx, setupThread.OrganizationID, models.ThreadTypeSupport, 1)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		// Really should be exactly one, but to not blow up for our own support forum only err when none exist.
		if len(supportThreads) < 1 {
			return nil, grpcErrorf(codes.Internal, "Expected at least 1 support thread for org %s", setupThread.OrganizationID)
		}
		supportThread := supportThreads[0]

		var newStep int
		var msgBML bml.BML
		switch in.EventType {
		case threading.OnboardingThreadEventRequest_PROVISIONED_PHONE:
			// Second phone line
			pn, err := phone.ParseNumber(in.GetProvisionedPhone().PhoneNumber)
			if err != nil {
				return nil, grpcErrorf(codes.Internal, "Invalid phone number '%s' for org %s", in.GetProvisionedPhone().PhoneNumber, setupThread.OrganizationID)
			}
			supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
			prettyPhone, err := pn.Format(phone.Pretty)
			if err != nil {
				golog.Errorf("Failed to format provisioned number '%s' for org %s", pn, setupThread.OrganizationID)
				prettyPhone = pn.String()
			}
			msgBML = bml.BML{
				`Success! You can now use your Spruce number ` + prettyPhone + ` for calls, texts, and voicemails with patients.`,
				` If you have any questions about using Spruce, just message us in `,
				&bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"}, ` and we’ll reply to help you.`,
			}
			newStep = 1
		case threading.OnboardingThreadEventRequest_GENERIC_SETUP:
			ev := in.GetGenericSetup()
			switch ev.Name {
			case eventSetupAnsweringService:
				phoneSetupURL := deeplink.OrgSettingsPhoneURL(s.webDomain, setupThread.OrganizationID)
				supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
				msgBML = bml.BML{
					`Great! First let’s `, &bml.Anchor{HREF: phoneSetupURL, Text: "set up your Spruce number"},
					` if you haven’t already. Voicemails left with your Spruce number will be automatically transcribed,`,
					` and you will receive a notifications on your phone. We can also enable more advanced configurations,`,
					` such as custom greetings, special handling for urgent voicemails, and after-hours oncall rotations.`,
					` Just send us a message in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"},
					` with the configuration you would like, and someone from Spruce will reply to help you.`,
				}
				newStep = 2
			case eventSetupTeamMessaging:
				inviteURL := deeplink.OrgColleagueInviteURL(s.webDomain, setupThread.OrganizationID)
				msgBML = bml.BML{
					`After `, &bml.Anchor{HREF: inviteURL, Text: "adding teammates"}, `, you can start a new team conversation`,
					` from the home screen and message 1:1 or in group chats. You can also collaborate and make notes within`,
					` the context of patient conversations (patients won’t see this activity, but your teammates will).`,
				}
				newStep = 3
			case eventSetupTelemedicine:
				supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
				msgBML = bml.BML{
					`Interested in engaging patients digitally with e-visits, video calls, care plans (including eprescribing),`,
					` mobile billpay, appointment reminders, and satisfaction surveys? Digital care on Spruce enables you to`,
					` simultaneously offer a standout patient experience and streamline your practice efficiency. The Digital`,
					` Practice offering on Spruce is coming soon: just message us in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"},
					` if you would like to be a part of the private beta.`,
				}
				newStep = 4
			default:
				return nil, grpcErrorf(codes.Internal, "Unhandled onboarding setup event '%s' for org %s", in.Event, setupThread.OrganizationID)
			}
		default:
			golog.Debugf("Unhandled onboarding thread event '%s'", in.Event)
		}
		if msgBML != nil {
			msg, err := msgBML.Format()
			if err != nil {
				return nil, grpcErrorf(codes.Internal, "invalid onboarding message BML: %s", err)
			}
			var summary string
			if msg != "" {
				summary, err = models.SummaryFromText("Setup: " + msg)
				if err != nil {
					return nil, grpcErrorf(codes.Internal, "Failed to generate summary for event %s", in.EventType)
				}
			}
			if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
				// Query the state again so we can lock it and recheck the step to make sure there's no concurrent update
				state, err := dl.SetupThreadState(ctx, setupThread.ID, dal.ForUpdate)
				if err != nil {
					return errors.Trace(err)
				}
				if state.Step != 0 {
					// Not an error, someone just got here before us
					return nil
				}
				if err := dl.UpdateSetupThreadState(ctx, setupThread.ID, &dal.SetupThreadStateUpdate{Step: &newStep}); err != nil {
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
				return nil, grpcErrorf(codes.Internal, err.Error())
			}
		}
	}

	thread, err := s.dal.Thread(ctx, state.ThreadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.OnboardingThreadEventResponse{Thread: th}, nil
}
