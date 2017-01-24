package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestLookupRequestStatus(t *testing.T) {
	ctx := context.Background()
	requestID := "requestID"
	t.Run("Error-NoRequestFound", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		tGQL.ra.Expect(mock.NewExpectation(tGQL.ra.BatchJobs, &threading.BatchJobsRequest{
			LookupKey: &threading.BatchJobsRequest_ID{
				ID: requestID,
			},
		}).WithReturns((*threading.BatchJobsResponse)(nil), grpc.Errorf(codes.NotFound, "NotFound")))
		res, err := lookupRequestStatus(ctx, tGQL.ra, requestID)
		test.AssertNil(t, res)
		test.Equals(t, errors.Cause(errors.Errorf("No request status found for ID %s", requestID)), errors.Cause(err))
	})
	t.Run("Error-ExpectedOnlyOne", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		tGQL.ra.Expect(mock.NewExpectation(tGQL.ra.BatchJobs, &threading.BatchJobsRequest{
			LookupKey: &threading.BatchJobsRequest_ID{
				ID: requestID,
			},
		}).WithReturns(&threading.BatchJobsResponse{BatchJobs: []*threading.BatchJob{
			{}, {},
		}}, nil))
		res, err := lookupRequestStatus(ctx, tGQL.ra, requestID)
		test.AssertNil(t, res)
		test.Equals(t, errors.Cause(errors.Errorf("Expected 1 result for batch jobs id query %v, but got %d", requestID, 2)), errors.Cause(err))
	})
	t.Run("Success", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		batchJob := &threading.BatchJob{
			ID:               requestID,
			Type:             threading.BATCH_JOB_TYPE_BATCH_POST_MESSAGES,
			Status:           threading.BATCH_JOB_STATUS_COMPLETE,
			TasksRequested:   1,
			TasksCompleted:   1,
			TasksErrored:     0,
			RequestingEntity: "requestingEntity",
		}
		tGQL.ra.Expect(mock.NewExpectation(tGQL.ra.BatchJobs, &threading.BatchJobsRequest{
			LookupKey: &threading.BatchJobsRequest_ID{
				ID: requestID,
			},
		}).WithReturns(&threading.BatchJobsResponse{BatchJobs: []*threading.BatchJob{
			batchJob,
		}}, nil))
		res, err := lookupRequestStatus(ctx, tGQL.ra, requestID)
		test.OK(t, err)
		test.Equals(t, transformRequestStatusToResponse(ctx, batchJob), res)
	})
}
