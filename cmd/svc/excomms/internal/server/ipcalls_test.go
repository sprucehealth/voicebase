package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	notimock "github.com/sprucehealth/backend/svc/notification/mock"
	"github.com/sprucehealth/backend/svc/threading"
	threadmock "github.com/sprucehealth/backend/svc/threading/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestInitiateIPCall(t *testing.T) {
	dl := dalmock.New(t)
	dir := dirmock.New(t)
	noti := notimock.New(t)
	defer mock.FinishAll(dl, dir, noti)

	clk := clock.NewManaged(time.Unix(1e9, 0))
	svc := NewService("accountSID", "authToken", "appSID", "sigSID", "sig", "vidSID", dl,
		"apiURL", dir, nil, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, noti)

	identCounter := 0
	svc.(*excommsService).genIPCallIdentity = func() (string, error) {
		identCounter++
		return fmt.Sprintf("identity_%d", identCounter), nil
	}

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{IDs: []string{"entity_2", "entity_1"}},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:          "entity_2",
				AccountID:   "account_2",
				Memberships: []*directory.Entity{{ID: "org", Type: directory.EntityType_ORGANIZATION}},
			},
			{
				ID:          "entity_1",
				AccountID:   "account_1",
				Memberships: []*directory.Entity{{ID: "org", Type: directory.EntityType_ORGANIZATION}},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.CreateIPCall, &models.IPCall{
		Type:    models.IPCallTypeVideo,
		Pending: true,
		Participants: []*models.IPCallParticipant{
			{
				AccountID:   "account_2",
				EntityID:    "entity_2",
				Identity:    "identity_1",
				Role:        models.IPCallParticipantRoleRecipient,
				State:       models.IPCallStatePending,
				NetworkType: models.NetworkTypeUnknown,
			},
			{
				AccountID:   "account_1",
				EntityID:    "entity_1",
				Identity:    "identity_2",
				Role:        models.IPCallParticipantRoleCaller,
				State:       models.IPCallStateAccepted,
				NetworkType: models.NetworkTypeCellular,
			},
		},
	}).WithReturns(nil))

	noti.Expect(mock.NewExpectation(noti.SendNotification, &notification.Notification{
		Type:             notification.IncomingIPCall,
		CallID:           "",
		OrganizationID:   "org",
		EntitiesToNotify: []string{"entity_2"},
		DedupeKey:        "",
		CollapseKey:      string(notification.IncomingIPCall),
		ShortMessages: map[string]string{
			"entity_2": "☎️ Video call from your healthcare provider",
		},
	}).WithReturns(nil))

	res, err := svc.InitiateIPCall(nil, &excomms.InitiateIPCallRequest{
		Type:               excomms.IPCallType_VIDEO,
		CallerEntityID:     "entity_1",
		RecipientEntityIDs: []string{"entity_2"},
		NetworkType:        excomms.NetworkType_CELLULAR,
	})
	test.OK(t, err)
	test.Equals(t, &excomms.InitiateIPCallResponse{
		Call: &excomms.IPCall{
			Type:    excomms.IPCallType_VIDEO,
			Pending: true,
			Token:   res.Call.Token, // Not deterministic so can't test the exact value, but doesn't matter too much anyway as the token generation is tested elsewhere
			Participants: []*excomms.IPCallParticipant{
				{
					AccountID:   "account_2",
					EntityID:    "entity_2",
					Identity:    "identity_1",
					Role:        excomms.IPCallParticipantRole_RECIPIENT,
					State:       excomms.IPCallState_PENDING,
					NetworkType: excomms.NetworkType_UNKNOWN,
				},
				{
					AccountID:   "account_1",
					EntityID:    "entity_1",
					Identity:    "identity_2",
					Role:        excomms.IPCallParticipantRole_CALLER,
					State:       excomms.IPCallState_ACCEPTED,
					NetworkType: excomms.NetworkType_CELLULAR,
				},
			},
		},
	}, res)
}

func TestIPCall(t *testing.T) {
	dl := dalmock.New(t)
	defer dl.Finish()

	clk := clock.NewManaged(time.Unix(1e9, 0))
	svc := NewService("accountSID", "authToken", "appSID", "sigSID", "sig", "vidSID", dl,
		"apiURL", nil, nil, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, nil)

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
				AccountID:   "account_1",
				EntityID:    "entity_1",
				Identity:    "identity_1",
				Role:        models.IPCallParticipantRoleCaller,
				State:       models.IPCallStateAccepted,
				NetworkType: models.NetworkTypeCellular,
			},
			{
				AccountID:   "account_2",
				EntityID:    "entity_2",
				Identity:    "identity_2",
				Role:        models.IPCallParticipantRoleRecipient,
				State:       models.IPCallStatePending,
				NetworkType: models.NetworkTypeWiFi,
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
					AccountID:   "account_1",
					EntityID:    "entity_1",
					Identity:    "identity_1",
					Role:        excomms.IPCallParticipantRole_CALLER,
					State:       excomms.IPCallState_ACCEPTED,
					NetworkType: excomms.NetworkType_CELLULAR,
				},
				{
					AccountID:   "account_2",
					EntityID:    "entity_2",
					Identity:    "identity_2",
					Role:        excomms.IPCallParticipantRole_RECIPIENT,
					State:       excomms.IPCallState_PENDING,
					NetworkType: excomms.NetworkType_WIFI,
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
		"apiURL", nil, nil, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, nil)

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
						AccountID:   "account_1",
						EntityID:    "entity_1",
						Identity:    "identity_1",
						Role:        models.IPCallParticipantRoleCaller,
						State:       models.IPCallStateAccepted,
						NetworkType: models.NetworkTypeCellular,
					},
					{
						AccountID:   "account_2",
						EntityID:    "entity_2",
						Identity:    "identity_2",
						Role:        models.IPCallParticipantRoleRecipient,
						State:       models.IPCallStatePending,
						NetworkType: models.NetworkTypeUnknown,
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
						AccountID:   "account_1",
						EntityID:    "entity_1",
						Identity:    "identity_1",
						Role:        excomms.IPCallParticipantRole_CALLER,
						State:       excomms.IPCallState_ACCEPTED,
						NetworkType: excomms.NetworkType_CELLULAR,
					},
					{
						AccountID:   "account_2",
						EntityID:    "entity_2",
						Identity:    "identity_2",
						Role:        excomms.IPCallParticipantRole_RECIPIENT,
						State:       excomms.IPCallState_PENDING,
						NetworkType: excomms.NetworkType_UNKNOWN,
					},
				},
			},
		},
	}, res)
}

func TestUpdateIPCall(t *testing.T) {
	dl := dalmock.New(t)
	thr := threadmock.New(t)
	defer mock.FinishAll(dl, thr)

	clk := clock.NewManaged(time.Unix(1e9, 0))
	svc := NewService("accountSID", "authToken", "appSID", "sigSID", "sig", "vidSID", dl,
		"apiURL", nil, thr, nil, "extTopic", "evTopic", clk, nil, nil, nil, nil, nil)

	ipcid, err := models.NewIPCallID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.IPCall, ipcid).WithReturns((*models.IPCall)(nil), dal.ErrIPCallNotFound))
	_, err = svc.UpdateIPCall(nil, &excomms.UpdateIPCallRequest{
		IPCallID:  ipcid.String(),
		AccountID: "account_caller",
		State:     excomms.IPCallState_CONNECTED,
	})
	test.Equals(t, codes.NotFound, grpc.Code(err))

	dl.Expect(mock.NewExpectation(dl.IPCall, ipcid).WithReturns(
		&models.IPCall{
			ID:        ipcid,
			Type:      models.IPCallTypeVideo,
			Pending:   true,
			Initiated: clk.Now().Add(-110e9),
			Participants: []*models.IPCallParticipant{
				{
					EntityID:    "entity_caller",
					AccountID:   "account_caller",
					Identity:    "identity_caller",
					Role:        models.IPCallParticipantRoleCaller,
					State:       models.IPCallStateConnected,
					NetworkType: models.NetworkTypeUnknown,
				},
				{
					EntityID:    "entity_recipient",
					AccountID:   "account_recipient",
					Identity:    "identity_recipient",
					Role:        models.IPCallParticipantRoleRecipient,
					State:       models.IPCallStateAccepted,
					NetworkType: models.NetworkTypeUnknown,
				},
			},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateIPCallParticipant, ipcid, "account_caller", models.IPCallStateCompleted, models.NetworkTypeWiFi))
	dl.Expect(mock.NewExpectation(dl.UpdateIPCall, ipcid, false))

	thr.Expect(mock.NewExpectation(thr.ThreadsForMember, &threading.ThreadsForMemberRequest{
		EntityID:    "entity_recipient",
		PrimaryOnly: true,
	}).WithReturns(&threading.ThreadsForMemberResponse{
		Threads: []*threading.Thread{
			{ID: "thread"},
		},
	}, nil))

	thr.Expect(mock.NewExpectation(thr.PostMessage, &threading.PostMessageRequest{
		UUID:         ipcid.String(),
		ThreadID:     "thread",
		FromEntityID: "entity_caller",
		Title:        "Video call, 1:50s",
		Summary:      "Video call, 1:50s",
	}))

	res, err := svc.UpdateIPCall(nil, &excomms.UpdateIPCallRequest{
		IPCallID:    ipcid.String(),
		AccountID:   "account_caller",
		State:       excomms.IPCallState_COMPLETED,
		NetworkType: excomms.NetworkType_WIFI,
	})
	test.OK(t, err)
	test.Equals(t, &excomms.UpdateIPCallResponse{
		Call: &excomms.IPCall{
			ID:      ipcid.String(),
			Type:    excomms.IPCallType_VIDEO,
			Pending: false,
			Token:   res.Call.Token, // Not deterministic so can't test the exact value, but doesn't matter too much anyway as the token generation is tested elsewhere
			Participants: []*excomms.IPCallParticipant{
				{
					AccountID:   "account_caller",
					EntityID:    "entity_caller",
					Identity:    "identity_caller",
					Role:        excomms.IPCallParticipantRole_CALLER,
					State:       excomms.IPCallState_COMPLETED,
					NetworkType: excomms.NetworkType_WIFI,
				},
				{
					AccountID:   "account_recipient",
					EntityID:    "entity_recipient",
					Identity:    "identity_recipient",
					Role:        excomms.IPCallParticipantRole_RECIPIENT,
					State:       excomms.IPCallState_ACCEPTED,
					NetworkType: excomms.NetworkType_UNKNOWN,
				},
			},
		},
	}, res)
}
