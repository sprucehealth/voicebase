package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestIPCall(t *testing.T) {
	dl := dalmock.New(t)
	defer dl.Finish()

	clk := clock.NewManaged(time.Unix(1e9, 0))
	svc := NewService("accountSID", "authToken", "appSID", "sigSID", "sig", "vidSID", dl,
		"apiURL", nil, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, nil)

	ipcID, err := models.NewIPCallID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.IPCall, ipcID).WithReturns((*models.IPCall)(nil), dal.ErrIPCallNotFound))

	_, err = svc.IPCall(nil, &excomms.IPCallRequest{IPCallID: ipcID.String(), AccountID: "account_1"})
	test.Equals(t, codes.NotFound, grpc.Code(err))

	dl.Expect(mock.NewExpectation(dl.IPCall, ipcID).WithReturns(&models.IPCall{
		ID:        ipcID,
		Type:      models.IPCallTypeVideo,
		Pending:   true,
		Initiated: clk.Now(),
		Participants: []*models.IPCallParticipant{
			{
				AccountID: "account_1",
				EntityID:  "entity_1",
				Identity:  "identity_1",
				Role:      models.IPCallParticipantRoleCaller,
				State:     models.IPCallStateAccepted,
			},
			{
				AccountID: "account_2",
				EntityID:  "entity_2",
				Identity:  "identity_2",
				Role:      models.IPCallParticipantRoleRecipient,
				State:     models.IPCallStatePending,
			},
		}}, nil))

	res, err := svc.IPCall(nil, &excomms.IPCallRequest{IPCallID: ipcID.String(), AccountID: "account_1"})
	test.OK(t, err)
	test.Equals(t, &excomms.IPCallResponse{
		Call: &excomms.IPCall{
			ID:      ipcID.String(),
			Type:    excomms.IPCallType_VIDEO,
			Pending: true,
			Token:   res.Call.Token, // Not deterministic so can't test the exact value, but doesn't matter too much anyway as the token generation is tested elsewhere
			Participants: []*excomms.IPCallParticipant{
				{
					AccountID: "account_1",
					EntityID:  "entity_1",
					Identity:  "identity_1",
					Role:      excomms.IPCallParticipantRole_CALLER,
					State:     excomms.IPCallState_ACCEPTED,
				},
				{
					AccountID: "account_2",
					EntityID:  "entity_2",
					Identity:  "identity_2",
					Role:      excomms.IPCallParticipantRole_RECIPIENT,
					State:     excomms.IPCallState_PENDING,
				},
			},
		},
	}, res)
}

func TestPendingIPCalls(t *testing.T) {
	dl := dalmock.New(t)
	defer dl.Finish()

	clk := clock.NewManaged(time.Unix(1e9, 0))
	svc := NewService("accountSID", "authToken", "appSID", "sigSID", "sig", "vidSID", dl,
		"apiURL", nil, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, nil)

	dl.Expect(mock.NewExpectation(dl.PendingIPCallsForAccount, "account_1").WithReturns([]*models.IPCall{}, nil))
	res, err := svc.PendingIPCalls(nil, &excomms.PendingIPCallsRequest{AccountID: "account_1"})
	test.OK(t, err)
	test.Equals(t, 0, len(res.Calls))

	ipcID, err := models.NewIPCallID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.PendingIPCallsForAccount, "account_1").WithReturns(
		[]*models.IPCall{
			{
				ID:        ipcID,
				Type:      models.IPCallTypeVideo,
				Pending:   true,
				Initiated: clk.Now(),
				Participants: []*models.IPCallParticipant{
					{
						AccountID: "account_1",
						EntityID:  "entity_1",
						Identity:  "identity_1",
						Role:      models.IPCallParticipantRoleCaller,
						State:     models.IPCallStateAccepted,
					},
					{
						AccountID: "account_2",
						EntityID:  "entity_2",
						Identity:  "identity_2",
						Role:      models.IPCallParticipantRoleRecipient,
						State:     models.IPCallStatePending,
					},
				},
			},
		}, nil))
	res, err = svc.PendingIPCalls(nil, &excomms.PendingIPCallsRequest{AccountID: "account_1"})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Calls))
	test.Equals(t, &excomms.PendingIPCallsResponse{
		Calls: []*excomms.IPCall{
			{
				ID:      ipcID.String(),
				Type:    excomms.IPCallType_VIDEO,
				Pending: true,
				Token:   res.Calls[0].Token, // Not deterministic so can't test the exact value, but doesn't matter too much anyway as the token generation is tested elsewhere
				Participants: []*excomms.IPCallParticipant{
					{
						AccountID: "account_1",
						EntityID:  "entity_1",
						Identity:  "identity_1",
						Role:      excomms.IPCallParticipantRole_CALLER,
						State:     excomms.IPCallState_ACCEPTED,
					},
					{
						AccountID: "account_2",
						EntityID:  "entity_2",
						Identity:  "identity_2",
						Role:      excomms.IPCallParticipantRole_RECIPIENT,
						State:     excomms.IPCallState_PENDING,
					},
				},
			},
		},
	}, res)
}
