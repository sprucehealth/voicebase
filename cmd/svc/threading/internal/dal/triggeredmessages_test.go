package dal

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

func TestTriggeredMessages(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB, clock.New())
	ctx := context.Background()

	tm := &models.TriggeredMessage{
		OrganizationEntityID: "OrganizationEntityID",
		ActorEntityID:        "ActorEntityID",
		TriggerKey:           "TriggerKey",
		TriggerSubkey:        "TriggerSubkey",
		Enabled:              true,
	}
	id, err := dal.CreateTriggeredMessage(ctx, tm)
	test.OK(t, err)
	test.Assert(t, id.IsValid, "ID should be valid")
	test.Equals(t, id, tm.ID)

	tm2 := &models.TriggeredMessage{
		OrganizationEntityID: "OrganizationEntityID",
		ActorEntityID:        "ActorEntityID",
		TriggerKey:           "TriggerKey2",
		TriggerSubkey:        "TriggerSubkey",
		Enabled:              true,
	}
	tm3 := &models.TriggeredMessage{
		OrganizationEntityID: "OrganizationEntityID",
		ActorEntityID:        "ActorEntityID",
		TriggerKey:           "TriggerKey",
		TriggerSubkey:        "TriggerSubkey3",
		Enabled:              false,
	}
	err = dal.CreateTriggeredMessages(ctx, []*models.TriggeredMessage{tm2, tm3})
	test.OK(t, err)
	test.Assert(t, tm2.ID.IsValid, "ID should be valid")
	test.Assert(t, tm3.ID.IsValid, "ID should be valid")

	tm3, err = dal.TriggeredMessage(ctx, tm3.ID)
	test.OK(t, err)
	test.Assert(t, tm3.ID.IsValid, "ID should be valid")
	test.Assert(t, !tm3.Enabled, "Should not be enabled")

	aff, err := dal.UpdateTriggeredMessage(ctx, tm3.ID, &models.TriggeredMessageUpdate{
		Enabled: ptr.Bool(true),
	})
	test.OK(t, err)
	test.Equals(t, aff, int64(1))

	tm3, err = dal.TriggeredMessage(ctx, tm3.ID)
	test.OK(t, err)
	test.Assert(t, tm3.ID.IsValid, "ID should be valid")
	test.Assert(t, tm3.Enabled, "Should now be enabled")

	aff, err = dal.DeleteTriggeredMessage(ctx, tm3.ID)
	test.OK(t, err)
	test.Equals(t, int64(1), aff)

	tm3, err = dal.TriggeredMessage(ctx, tm3.ID)
	test.Equals(t, ErrNotFound, errors.Cause(err))

	tmi := &models.TriggeredMessageItem{
		TriggeredMessageID: tm.ID,
		Ordinal:            0,
		ActorEntityID:      "ActorEntityID",
		Internal:           true,
		Data:               &models.Message{},
	}
	tmiID, err := dal.CreateTriggeredMessageItem(ctx, tmi)
	test.OK(t, err)
	test.Assert(t, tmi.ID.IsValid, "ID should be valid")
	test.Equals(t, tmiID, tmi.ID)

	tmi, err = dal.TriggeredMessageItem(ctx, tmi.ID)
	test.OK(t, err)
	test.Equals(t, tmiID, tmi.ID)

	tmi2 := &models.TriggeredMessageItem{
		TriggeredMessageID: tm.ID,
		Ordinal:            1,
		ActorEntityID:      "ActorEntityID",
		Internal:           true,
		Data:               &models.Message{},
	}
	tmiID, err = dal.CreateTriggeredMessageItem(ctx, tmi2)
	test.OK(t, err)
	test.Assert(t, tmi2.ID.IsValid, "ID should be valid")
	test.Equals(t, tmiID, tmi2.ID)

	tmis, err := dal.TriggeredMessageItemsForTriggeredMessage(ctx, tm.ID)
	test.OK(t, err)
	test.Equals(t, 2, len(tmis))
	test.Equals(t, int64(0), tmis[0].Ordinal)
	test.Equals(t, int64(1), tmis[1].Ordinal)

	aff, err = dal.DeleteTriggeredMessageItemsForTriggeredMessage(ctx, tm.ID)
	test.OK(t, err)
	test.Equals(t, int64(2), aff)

	tmis, err = dal.TriggeredMessageItemsForTriggeredMessage(ctx, tm.ID)
	test.OK(t, err)
	test.Equals(t, 0, len(tmis))
}
