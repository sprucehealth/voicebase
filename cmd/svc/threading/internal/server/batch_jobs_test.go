package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestBatchJobs(t *testing.T) {
	requestingEntity := "requestingEntity"
	mClk := clock.NewManaged(time.Now())
	batchJobID, err := models.NewBatchJobID()
	test.OK(t, err)
	t.Run("Error-UnknownLookupKey", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchJobs(t, st, &threading.BatchJobsRequest{}, nil, grpc.Errorf(codes.InvalidArgument, "Unknown lookup key type"))
	})
	t.Run("Error-IDRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchJobs(t, st, &threading.BatchJobsRequest{
			LookupKey: &threading.BatchJobsRequest_ID{},
		}, nil, grpc.Errorf(codes.InvalidArgument, "ID Required"))
	})
	t.Run("Success-ID-NoErrors", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		batchJob := &models.BatchJob{
			ID:               batchJobID,
			Type:             models.BatchJobTypeBatchPostMessages,
			Status:           models.BatchJobStatusComplete,
			TasksRequested:   100,
			TasksCompleted:   100,
			TasksErrored:     0,
			Completed:        ptr.Time(mClk.Now()),
			Created:          mClk.Now(),
			Modified:         mClk.Now(),
			RequestingEntity: requestingEntity,
		}
		st.dal.Expect(mock.NewExpectation(st.dal.BatchJob, batchJobID, []interface{}{}).WithReturns(batchJob, nil))
		rBatchJob, err := transformBatchJobToResponse(batchJob, nil)
		test.OK(t, err)
		testBatchJobs(t, st, &threading.BatchJobsRequest{
			LookupKey: &threading.BatchJobsRequest_ID{
				ID: batchJobID.String(),
			},
		}, &threading.BatchJobsResponse{
			BatchJobs: []*threading.BatchJob{rBatchJob},
		}, nil)
	})
}

func testBatchJobs(
	t *testing.T,
	st *serverTest,
	in *threading.BatchJobsRequest,
	exp *threading.BatchJobsResponse,
	expErr error) {
	resp, err := st.server.BatchJobs(st.ctx, in)
	test.Equals(t, expErr, err)
	test.Equals(t, exp, resp)
	st.Finish()
}

func TestBatchPostMessages(t *testing.T) {
	requestingEntity := "requestingEntity"
	threaID, err := models.NewThreadID()
	test.OK(t, err)
	t.Run("Error-AtLeastOnePostRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchPostMessages(t, st, &threading.BatchPostMessagesRequest{}, nil, grpc.Errorf(codes.InvalidArgument, "At least 1 PostMessagesRequest is required"))
	})
	t.Run("Error-RequestingEntityRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchPostMessages(t, st, &threading.BatchPostMessagesRequest{
			PostMessagesRequests: []*threading.PostMessagesRequest{
				{UUID: "UUID"},
			},
		}, nil, grpc.Errorf(codes.InvalidArgument, "RequestingEntity Required"))
	})
	t.Run("Error-PostMessageRequest-ThreadIDRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchPostMessages(t, st, &threading.BatchPostMessagesRequest{
			RequestingEntity: requestingEntity,
			PostMessagesRequests: []*threading.PostMessagesRequest{
				{UUID: "UUID"},
			},
		}, nil, grpc.Errorf(codes.InvalidArgument, "ThreadID on PostMessagesRequest Required"))
	})
	t.Run("Error-PostMessageRequest-FromEntityIDRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchPostMessages(t, st, &threading.BatchPostMessagesRequest{
			RequestingEntity: requestingEntity,
			PostMessagesRequests: []*threading.PostMessagesRequest{
				{
					UUID:     "UUID",
					ThreadID: threaID.String(),
				},
			},
		}, nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID on PostMessagesRequest Required"))
	})
	t.Run("Error-PostMessageRequest-AtLeastOneMessageRequired", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testBatchPostMessages(t, st, &threading.BatchPostMessagesRequest{
			RequestingEntity: requestingEntity,
			PostMessagesRequests: []*threading.PostMessagesRequest{
				{
					UUID:         "UUID",
					ThreadID:     threaID.String(),
					FromEntityID: requestingEntity,
				},
			},
		}, nil, grpc.Errorf(codes.InvalidArgument, "At least 1 Message is required in PostMessagesRequest"))
	})
	// TODO: Success case... Issue with generating the batch job ID outside of the DAL for ordering reasons
}

func testBatchPostMessages(
	t *testing.T,
	st *serverTest,
	in *threading.BatchPostMessagesRequest,
	exp *threading.BatchPostMessagesResponse,
	expErr error) {
	resp, err := st.server.BatchPostMessages(st.ctx, in)
	test.Equals(t, expErr, err)
	test.Equals(t, exp, resp)
	st.Finish()
}
