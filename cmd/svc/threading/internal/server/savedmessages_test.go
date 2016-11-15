package server

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	mockdirectory "github.com/sprucehealth/backend/svc/directory/mock"
	mockmedia "github.com/sprucehealth/backend/svc/media/mock"
	mocksettings "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestCreateSavedMessage(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	smID, err := models.NewSavedMessageID()
	test.OK(t, err)
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.CreateSavedMessage, &models.SavedMessage{
		Title:           "foo",
		OrganizationID:  "org",
		OwnerEntityID:   "owner",
		CreatorEntityID: "creator",
		Internal:        true,
		Content: &models.Message{
			Title:       "bar",
			Text:        "text",
			Summary:     "summary",
			Attachments: nil,
		},
	}).WithReturns(smID, nil))

	dl.Expect(mock.NewExpectation(dl.SavedMessages, []models.SavedMessageID{smID}).WithReturns([]*models.SavedMessage{
		{
			ID:              smID,
			Title:           "foo",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &models.Message{
				Title:       "bar",
				Text:        "text",
				Summary:     "summary",
				Attachments: nil,
			},
			Created:  now,
			Modified: now,
		},
	}, nil))

	res, err := srv.CreateSavedMessage(context.Background(), &threading.CreateSavedMessageRequest{
		Title:           "foo",
		OrganizationID:  "org",
		OwnerEntityID:   "owner",
		CreatorEntityID: "creator",
		Content: &threading.CreateSavedMessageRequest_Message{
			Message: &threading.MessagePost{
				Internal: true,
				Title:    "bar",
				Text:     "text",
				Summary:  "summary",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateSavedMessageResponse{
		SavedMessage: &threading.SavedMessage{
			ID:              smID.String(),
			Title:           "foo",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &threading.SavedMessage_Message{
				Message: &threading.Message{
					Title:   "bar",
					Text:    "text",
					Summary: "summary",
				},
			},
			Created:  uint64(now.Unix()),
			Modified: uint64(now.Unix()),
		},
	}, res)
}

func TestDeleteSavedMessage(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	smID, err := models.NewSavedMessageID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.DeleteSavedMessages, []models.SavedMessageID{smID}))

	res, err := srv.DeleteSavedMessage(context.Background(), &threading.DeleteSavedMessageRequest{
		SavedMessageID: smID.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.DeleteSavedMessageResponse{}, res)
}

func TestSavedMessagesByID(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	smID, err := models.NewSavedMessageID()
	test.OK(t, err)
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.SavedMessages, []models.SavedMessageID{smID}).WithReturns([]*models.SavedMessage{
		{
			ID:              smID,
			Title:           "foo",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &models.Message{
				Title:       "bar",
				Text:        "text",
				Summary:     "summary",
				Attachments: nil,
			},
			Created:  now,
			Modified: now,
		},
	}, nil))

	res, err := srv.SavedMessages(context.Background(), &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_IDs{
			IDs: &threading.IDList{
				IDs: []string{smID.String()},
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.SavedMessagesResponse{
		SavedMessages: []*threading.SavedMessage{
			{
				ID:              smID.String(),
				Title:           "foo",
				OrganizationID:  "org",
				OwnerEntityID:   "owner",
				CreatorEntityID: "creator",
				Internal:        true,
				Content: &threading.SavedMessage_Message{
					Message: &threading.Message{
						Title:   "bar",
						Text:    "text",
						Summary: "summary",
					},
				},
				Created:  uint64(now.Unix()),
				Modified: uint64(now.Unix()),
			},
		},
	}, res)
}

func TestSavedMessagesByEntityID(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	smID, err := models.NewSavedMessageID()
	test.OK(t, err)
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.SavedMessagesForEntities, []string{"owner"}).WithReturns([]*models.SavedMessage{
		{
			ID:              smID,
			Title:           "foo",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &models.Message{
				Title:       "bar",
				Text:        "text",
				Summary:     "summary",
				Attachments: nil,
			},
			Created:  now,
			Modified: now,
		},
	}, nil))

	res, err := srv.SavedMessages(context.Background(), &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_EntityIDs{
			EntityIDs: &threading.IDList{
				IDs: []string{"owner"},
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.SavedMessagesResponse{
		SavedMessages: []*threading.SavedMessage{
			{
				ID:              smID.String(),
				Title:           "foo",
				OrganizationID:  "org",
				OwnerEntityID:   "owner",
				CreatorEntityID: "creator",
				Internal:        true,
				Content: &threading.SavedMessage_Message{
					Message: &threading.Message{
						Title:   "bar",
						Text:    "text",
						Summary: "summary",
					},
				},
				Created:  uint64(now.Unix()),
				Modified: uint64(now.Unix()),
			},
		},
	}, res)
}

func TestUpdateSavedMessage(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	smID, err := models.NewSavedMessageID()
	test.OK(t, err)
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.UpdateSavedMessage, smID, &dal.SavedMessageUpdate{
		Title: ptr.String("new title"),
		Content: &models.Message{
			Title:       "bar",
			Text:        "text",
			Summary:     "summary",
			Attachments: nil,
		},
	}))

	dl.Expect(mock.NewExpectation(dl.SavedMessages, []models.SavedMessageID{smID}).WithReturns([]*models.SavedMessage{
		{
			ID:              smID,
			Title:           "new title",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &models.Message{
				Title:       "bar",
				Text:        "text",
				Summary:     "summary",
				Attachments: nil,
			},
			Created:  now,
			Modified: now,
		},
	}, nil))

	res, err := srv.UpdateSavedMessage(context.Background(), &threading.UpdateSavedMessageRequest{
		SavedMessageID: smID.String(),
		Title:          "new title",
		Content: &threading.UpdateSavedMessageRequest_Message{
			Message: &threading.MessagePost{
				Internal: true,
				Title:    "bar",
				Text:     "text",
				Summary:  "summary",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.UpdateSavedMessageResponse{
		SavedMessage: &threading.SavedMessage{
			ID:              smID.String(),
			Title:           "new title",
			OrganizationID:  "org",
			OwnerEntityID:   "owner",
			CreatorEntityID: "creator",
			Internal:        true,
			Content: &threading.SavedMessage_Message{
				Message: &threading.Message{
					Title:   "bar",
					Text:    "text",
					Summary: "summary",
				},
			},
			Created:  uint64(now.Unix()),
			Modified: uint64(now.Unix()),
		},
	}, res)
}
