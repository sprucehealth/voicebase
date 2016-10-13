package dal

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

func TestSavedMessages(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()
	now := time.Unix(1e9, 0)

	sm := &models.SavedMessage{
		CreatorEntityID: "creator",
		OwnerEntityID:   "owner",
		Internal:        true,
		Content:         &models.Message{Text: "foo"},
		Created:         now,
		Modified:        now,
	}
	id, err := dal.CreateSavedMessage(ctx, sm)
	test.OK(t, err)
	test.Assert(t, id.IsValid, "ID should be valid")
	test.Equals(t, id, sm.ID)

	sms, err := dal.SavedMessagesForEntities(ctx, []string{"owner"})
	test.OK(t, err)
	test.Equals(t, 1, len(sms))
	test.Equals(t, sm, sms[0])

	sm2 := &models.SavedMessage{
		CreatorEntityID: "creator2",
		OwnerEntityID:   "owner2",
		Internal:        false,
		Content:         &models.Message{Text: "bar"},
		Created:         now,
		Modified:        now,
	}
	id2, err := dal.CreateSavedMessage(ctx, sm2)
	test.OK(t, err)
	test.Assert(t, id2.IsValid, "ID should be valid")
	test.Equals(t, id2, sm2.ID)

	sms, err = dal.SavedMessagesForEntities(ctx, []string{"owner", "owner2"})
	test.OK(t, err)
	test.Equals(t, 2, len(sms))
	test.Equals(t, sm, sms[0])
	test.Equals(t, sm2, sms[1])

	n, err := dal.DeleteSavedMessages(ctx, []models.SavedMessageID{sm2.ID})
	test.OK(t, err)
	test.Equals(t, 1, n)

	n, err = dal.DeleteSavedMessages(ctx, []models.SavedMessageID{sm2.ID})
	test.OK(t, err)
	test.Equals(t, 0, n)

	sms, err = dal.SavedMessagesForEntities(ctx, []string{"owner", "owner2"})
	test.OK(t, err)
	test.Equals(t, 1, len(sms))
	test.Equals(t, sm, sms[0])

	test.OK(t, dal.UpdateSavedMessage(ctx, sms[0].ID, &SavedMessageUpdate{Title: ptr.String("zork"), Content: &models.Message{Text: "grue"}}))
	sm.Title = "zork"
	sm.Content = &models.Message{Text: "grue"}
	sms, err = dal.SavedMessagesForEntities(ctx, []string{"owner", "owner2"})
	test.OK(t, err)
	sm.Modified = sms[0].Modified
	test.Equals(t, 1, len(sms))
	test.Equals(t, sm, sms[0])
}
