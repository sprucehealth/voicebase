package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func TestCreateOnboardingThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()

	now := time.Unix(1e7, 0)

	thid, err := models.NewThreadID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.CreateThread, &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "Spruce Assistant: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"skip\" to get it later",
	}).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e2",
		Internal:     false,
		Text:         "Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\n<a href=\"https://WEBDOMAIN/org/o1/settings/phone\">Get your Spruce number</a>\nor type \"skip\" to get it later",
		Summary:      "Spruce Assistant: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"skip\" to get it later",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateOnboardingState, thid, "o1").WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(&models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e2",
		LastMessageSummary:   "Spruce Assistant: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"skip\" to get it later",
		LastMessageTimestamp: now,
		Created:              now,
	}, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, "WEBDOMAIN")
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
			LastMessageSummary:   "Spruce Assistant: Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.\n\nGet your Spruce number\nor type \"skip\" to get it later",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
		},
	}, res)
}
