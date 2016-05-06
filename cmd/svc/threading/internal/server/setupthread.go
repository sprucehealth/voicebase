package server

import (
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
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
		"Welcome! How would you like to use Spruce? (You can tap on multiple options)\n\n",

		&bml.Anchor{HREF: phoneSetupURL, Text: "Second phone line"},
		" for calls and texts with patients, without disclosing your personal number.\n\n",

		&bml.Anchor{HREF: answeringServiceURL, Text: "Automated answering service"},
		" that transcribes urgent voicemails and notifies you.\n\n",

		&bml.Anchor{HREF: teamMessagingURL, Text: "Secure team chat and care coordination"}, ".\n\n",

		"Digital care and ", &bml.Anchor{HREF: telemedicineURL, Text: "telemedicine"}, ".",
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
		return nil, grpcErrorf(codes.FailedPrecondition, "Expected at least 1 support thread for org %s", setupThread.OrganizationID)
	}
	supportThread := supportThreads[0]

	var newStepBit int
	var msgBML bml.BML
	switch in.EventType {
	case threading.OnboardingThreadEventRequest_PROVISIONED_PHONE:
		if state.Step&1 == 0 {
			// Second phone line
			pn, err := phone.ParseNumber(in.GetProvisionedPhone().PhoneNumber)
			if err != nil {
				return nil, grpcErrorf(codes.InvalidArgument, "Invalid phone number '%s' for org %s", in.GetProvisionedPhone().PhoneNumber, setupThread.OrganizationID)
			}
			supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
			prettyPhone, err := pn.Format(phone.Pretty)
			if err != nil {
				golog.Errorf("Failed to format provisioned number '%s' for org %s", pn, setupThread.OrganizationID)
				prettyPhone = pn.String()
			}
			msgBML = bml.BML{
				`You can now use your Spruce number ` + prettyPhone + ` for calls, texts, and voicemails with patients.`,
				` Just message us in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"}, ` if you have any questions or problems.`,
			}
			newStepBit = 1
		}
	case threading.OnboardingThreadEventRequest_GENERIC_SETUP:
		ev := in.GetGenericSetup()
		switch ev.Name {
		case eventSetupAnsweringService:
			if state.Step&2 == 0 {
				phoneSetupURL := deeplink.OrgSettingsPhoneURL(s.webDomain, setupThread.OrganizationID)
				supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
				msgBML = bml.BML{
					`For $25/provider/month your Spruce line can triage and transcribe patient voicemails, notifying you`,
					` via text when an urgent voicemail is received. You can also add teammates to create an on-call rotation.`,
					` To do this, first `, &bml.Anchor{HREF: phoneSetupURL, Text: "set up your Spruce number"},
					` if you haven’t already. Then tell us in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"},
					` that you would like to enable the answering service feature.`,
				}
				newStepBit = 2
			}
		case eventSetupTeamMessaging:
			if state.Step&4 == 0 {
				inviteURL := deeplink.OrgColleagueInviteURL(s.webDomain, setupThread.OrganizationID)
				msgBML = bml.BML{
					`After `, &bml.Anchor{HREF: inviteURL, Text: "adding teammates"}, `, you can start a new team conversation`,
					` from the home screen and message 1:1 or in group chats. You can also collaborate and make notes within`,
					` patient conversations (patients won’t see this activity, but your teammates will).`,
				}
				newStepBit = 4
			}
		case eventSetupTelemedicine:
			if state.Step&8 == 0 {
				supportThreadURL := deeplink.ThreadURLShareable(s.webDomain, setupThread.OrganizationID, supportThread.ID.String())
				msgBML = bml.BML{
					`Interested in engaging patients digitally with virtual visits, video calls, care plans (including e-prescribing),`,
					` mobile payment, appointment reminders, and satisfaction surveys? Digital care on Spruce enables you to`,
					` offer a standout patient experience and streamline your practice efficiency. The Digital Practice offering`,
					` on Spruce is coming soon: message us in `, &bml.Anchor{HREF: supportThreadURL, Text: "Spruce Support"},
					` if you would like to be a part of the private beta.`,
				}
				newStepBit = 8
			}
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
			return nil, grpcErrorf(codes.Internal, err.Error())
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
