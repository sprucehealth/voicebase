package dal

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
	"golang.org/x/net/context"
)

const schemaGlob = "schema/*.sql"

func TestIPCalls(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	clk := clock.NewManaged(time.Unix(1e9, 0))

	dal := New(dt.DB, clk)
	ctx := context.Background()

	call := &models.IPCall{
		Type: models.IPCallTypeVideo,
		Participants: []*models.IPCallParticipant{
			{
				AccountID:   "account_1",
				EntityID:    "entity_1",
				Identity:    "identity1",
				Role:        models.IPCallParticipantRoleCaller,
				State:       models.IPCallStateAccepted,
				NetworkType: models.NetworkTypeCellular,
			},
			{
				AccountID:   "account_2",
				EntityID:    "entity_2",
				Identity:    "identity2",
				Role:        models.IPCallParticipantRoleRecipient,
				State:       models.IPCallStatePending,
				NetworkType: models.NetworkTypeUnknown,
			},
		},
	}
	test.OK(t, dal.CreateIPCall(ctx, call))
	test.Equals(t, clk.Now(), call.Initiated)

	calls, err := dal.PendingIPCallsForAccount(ctx, "account_nonexistant")
	test.OK(t, err)
	test.Equals(t, 0, len(calls))

	calls, err = dal.PendingIPCallsForAccount(ctx, "account_1")
	test.OK(t, err)
	test.Equals(t, 1, len(calls))

	calls, err = dal.PendingIPCallsForAccount(ctx, "account_2")
	test.OK(t, err)
	test.Equals(t, 1, len(calls))
	test.Equals(t, call, calls[0])

	test.OK(t, dal.UpdateIPCallParticipant(ctx, call.ID, "account_2", models.IPCallStateConnected, models.NetworkTypeWiFi))
	call.Participants[1].State = models.IPCallStateConnected
	call.Participants[1].NetworkType = models.NetworkTypeWiFi

	calls, err = dal.PendingIPCallsForAccount(ctx, "account_2")
	test.OK(t, err)
	test.Equals(t, 1, len(calls))
	test.Equals(t, call, calls[0])

	test.OK(t, dal.UpdateIPCall(ctx, call.ID, false))
	call.Pending = false

	calls, err = dal.PendingIPCallsForAccount(ctx, "account_1")
	test.OK(t, err)
	test.Equals(t, 0, len(calls)) // Only pending calls should be not returned, not completed

	calls, err = dal.PendingIPCallsForAccount(ctx, "account_2")
	test.OK(t, err)
	test.Equals(t, 0, len(calls)) // Only pending calls should be not returned, not completed

	call2, err := dal.IPCall(ctx, call.ID)
	test.OK(t, err)
	test.Equals(t, call, call2)

	call2, err = dal.IPCall(ctx, call.ID, ForUpdate)
	test.OK(t, err)
	test.Equals(t, call, call2)
}
