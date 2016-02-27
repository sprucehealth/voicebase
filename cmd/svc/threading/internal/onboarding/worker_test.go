package onboarding

import (
	"github.com/sprucehealth/backend/svc/threading"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/test"
)

func TestWorker_Step1_Done(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingStateForEntity, "ent", true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     0,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "Success! Your patients can now reach you at (555) 111-2222. Next let’s set up you up to send and receive email through Spruce.\n\n<a href=\"https://WEBDOMAIN/org/orgid/settings/email\">Set up email support</a>\nor type \"Skip\" to set it up later",
		Summary:      "Setup: Success! Your patients can now reach you at (555) 111-2222. Next let’s set up you up to send and receive email through Spruce.\n\nSet up email support\nor type \"Skip\" to set it up later",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(1),
	}))

	test.OK(t, w.processEvent(&events.Envelope{
		Service: events.Service_EXCOMMS,
		Event: serializeEvent(t, &excomms.Event{
			Type: excomms.Event_PROVISIONED_ENDPOINT,
			Details: &excomms.Event_ProvisionedEndpoint{
				ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
					ForEntityID:  "ent",
					EndpointType: excomms.EndpointType_PHONE,
					Endpoint:     "+15551112222",
				},
			},
		}),
	}))
}

func TestWorker_Step1_Skip(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingState, thid, true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     0,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "You can set up your Spruce number at any time from the settings menu. Would you like to set up your account to send and receive email through Spruce?\n\n<a href=\"https://WEBDOMAIN/org/orgid/settings/email\">Set up email support</a>\nor type \"Skip\" to set it up later",
		Summary:      "Setup: You can set up your Spruce number at any time from the settings menu. Would you like to set up your account to send and receive email through Spruce?\n\nSet up email support\nor type \"Skip\" to set it up later",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(1),
	}))

	test.OK(t, w.processThreadItem(&threading.PublishedThreadItem{
		ThreadID: thid.String(),
		Item: &threading.ThreadItem{
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Skip",
				},
			},
		},
	}))
}

func TestWorker_Step2_Done(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingStateForEntity, "ent", true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     1,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "Great! Your patients can now reach you at foo@bar.com. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.\n\n<a href=\"https://WEBDOMAIN/org/orgid/invite\">Add a colleague to your organization</a>\nor type \"Skip\" to send invites later",
		Summary:      "Setup: Great! Your patients can now reach you at foo@bar.com. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.\n\nAdd a colleague to your organization\nor type \"Skip\" to send invites later",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(2),
	}))

	test.OK(t, w.processEvent(&events.Envelope{
		Service: events.Service_EXCOMMS,
		Event: serializeEvent(t, &excomms.Event{
			Type: excomms.Event_PROVISIONED_ENDPOINT,
			Details: &excomms.Event_ProvisionedEndpoint{
				ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
					ForEntityID:  "ent",
					EndpointType: excomms.EndpointType_EMAIL,
					Endpoint:     "foo@bar.com",
				},
			},
		}),
	}))
}

func TestWorker_Step2_Skip(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingState, thid, true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     1,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "You can set up your Spruce email at any time from the settings menu. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.\n\n<a href=\"https://WEBDOMAIN/org/orgid/invite\">Add a colleague to your organization</a>\nor type \"Skip\" to send invites later",
		Summary:      "Setup: You can set up your Spruce email at any time from the settings menu. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.\n\nAdd a colleague to your organization\nor type \"Skip\" to send invites later",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(2),
	}))

	test.OK(t, w.processThreadItem(&threading.PublishedThreadItem{
		ThreadID: thid.String(),
		Item: &threading.ThreadItem{
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Skip",
				},
			},
		},
	}))
}

func TestWorker_Step3_Done(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingStateForEntity, "org", true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     2,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "We’ve sent your invite to colleague. Once they’ve joined, you can communicate with them about care, right from a patient’s conversation thread.\n\nTo send internal messages or notes in a patient thread, simply tap the lock icon while writing a message to mark it as internal. You can test it out right here.\n\nThat’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the Team Spruce conversation thread and a real human will respond.",
		Summary:      "Setup: We’ve sent your invite to colleague. Once they’ve joined, you can communicate with them about care, right from a patient’s conversation thread.\n\nTo send internal messages or notes in a patient thread, simply tap the lock icon while writing a message to mark it as internal. You can test it out right here.\n\nThat’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the Team Spruce conversation thread and a real human will respond.",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(3),
	}))

	test.OK(t, w.processEvent(&events.Envelope{
		Service: events.Service_INVITE,
		Event: serializeEvent(t, &invite.Event{
			Type: invite.Event_INVITED_COLLEAGUES,
			Details: &invite.Event_InvitedColleagues{
				InvitedColleagues: &invite.InvitedColleagues{
					OrganizationEntityID: "org",
					InviterEntityID:      "inviter",
				},
			},
		}),
	}))
}

func TestWorker_Step3_Skip(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingState, thid, true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     2,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:              thid,
		PrimaryEntityID: "pent",
		OrganizationID:  "orgid",
	}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "pent",
		Internal:     false,
		Title:        "",
		Text:         "You can invite a colleague any time from the settings menu. Until then, you can still make internal notes on a patient conversation thread. These will only be visible to you until you add colleagues. \n\nYou can test out internal messaging by writing a message in this conversation and tapping the lock icon before sending it.\n\nThat’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the Team Spruce conversation thread and a real human will respond.",
		Summary:      "Setup: You can invite a colleague any time from the settings menu. Until then, you can still make internal notes on a patient conversation thread. These will only be visible to you until you add colleagues. \n\nYou can test out internal messaging by writing a message in this conversation and tapping the lock icon before sending it.\n\nThat’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the Team Spruce conversation thread and a real human will respond.",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateOnboardingState, thid, &dal.OnboardingStateUpdate{
		Step: ptr.Int(3),
	}))

	test.OK(t, w.processThreadItem(&threading.PublishedThreadItem{
		ThreadID: thid.String(),
		Item: &threading.ThreadItem{
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Skip",
				},
			},
		},
	}))
}

func TestWorker_Step4_NOOP(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	w := NewWorker(nil, dl, "WEBDOMAIN", "", "")

	thid, err := models.NewThreadID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.OnboardingState, thid, true).WithReturns(&models.OnboardingState{
		ThreadID: thid,
		Step:     3,
	}, nil))

	test.OK(t, w.processThreadItem(&threading.PublishedThreadItem{
		ThreadID: thid.String(),
		Item: &threading.ThreadItem{
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Skip",
				},
			},
		},
	}))
}

type marshaler interface {
	Marshal() ([]byte, error)
}

func serializeEvent(t *testing.T, m marshaler) []byte {
	data, err := m.Marshal()
	test.OK(t, err)
	return data
}
