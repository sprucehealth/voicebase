package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestBatchPostMessagesTest(t *testing.T) {
	ctx := context.Background()
	t.Run("Error-TooManyThreads", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		tooManyThreads := make([]string, batchPostMessagesMaxThreads+1)
		for i := range tooManyThreads {
			tooManyThreads[i] = "threadID"
		}
		res, err := batchPostMessages(ctx, tGQL.svc, tGQL.ra, &batchPostMessagesInput{
			ThreadIDs: tooManyThreads,
		})
		test.OK(t, err)
		test.Equals(t, &batchPostMessagesOutput{
			ErrorCode:    batchPostMessagesErrorCodeTooManyThreads,
			ErrorMessage: fmt.Sprintf("A maximum of %d threads are allowed in a single batch post", batchPostMessagesMaxThreads),
		}, res)
	})
	t.Run("Error-AtLeastOneThread", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		res, err := batchPostMessages(ctx, tGQL.svc, tGQL.ra, &batchPostMessagesInput{
			ThreadIDs: []string{},
		})
		test.OK(t, err)
		test.Equals(t, &batchPostMessagesOutput{
			ErrorCode:    batchPostMessagesErrorCodeInvalidInput,
			ErrorMessage: "At least 1 thread id is required",
		}, res)
	})
	t.Run("Error-AtLeastOneMessage", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		res, err := batchPostMessages(ctx, tGQL.svc, tGQL.ra, &batchPostMessagesInput{
			ThreadIDs: []string{"threadID1", "threadID2"},
		})
		test.OK(t, err)
		test.Equals(t, &batchPostMessagesOutput{
			ErrorCode:    batchPostMessagesErrorCodeInvalidInput,
			ErrorMessage: "At least 1 message is required",
		}, res)
	})
	t.Run("Error-CrossOrg", func(t *testing.T) {
		tGQL := newGQL(t)
		defer tGQL.finish()
		threadIDs := []string{"threadID1", "threadID2"}
		text := "text"

		// Lookup the threads
		tGQL.ra.Expect(mock.NewExpectation(tGQL.ra.Threads, &threading.ThreadsRequest{ThreadIDs: threadIDs}).WithReturns(&threading.ThreadsResponse{
			Threads: []*threading.Thread{
				&threading.Thread{OrganizationID: "org1"},
				&threading.Thread{OrganizationID: "org2"},
			},
		}, nil))

		res, err := batchPostMessages(ctx, tGQL.svc, tGQL.ra, &batchPostMessagesInput{
			ThreadIDs: threadIDs,
			Messages: []*messageInput{
				&messageInput{
					Text: text,
				},
			},
		})
		test.OK(t, err)
		test.Equals(t, &batchPostMessagesOutput{
			ErrorCode:    batchPostMessagesErrorCodeAcrossOrgs,
			ErrorMessage: "Batch messages cannot be applied across organizations",
		}, res)
	})
}
