package dal

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

func TestBatchJobs(t *testing.T) {
	requestingEntity := "requestingEntity"
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	mClk := clock.NewManaged(time.Now())
	dal := New(dt.DB, mClk)
	ctx := context.Background()

	bJob := &models.BatchJob{
		Type:             models.BatchJobTypeBatchPostMessages,
		Status:           models.BatchJobStatusPending,
		TasksRequested:   100,
		RequestingEntity: requestingEntity,
	}
	err := dal.CreateBatchJobs(ctx, []*models.BatchJob{bJob})
	test.OK(t, err)
	test.Assert(t, bJob.ID.IsValid, "ID should be valid")

	oID := bJob.ID
	bJob, err = dal.BatchJob(ctx, bJob.ID)
	test.OK(t, err)
	test.Equals(t, oID, bJob.ID)

	jobStatusComplete := models.BatchJobStatusComplete
	affCt, err := dal.UpdateBatchJob(ctx, bJob.ID, &models.BatchJobUpdate{
		TasksCompleted: ptr.Uint64(1),
		TasksErrored:   ptr.Uint64(1),
		Status:         &jobStatusComplete,
	})
	test.OK(t, err)
	test.Equals(t, int64(1), affCt)

	bJob, err = dal.BatchJob(ctx, bJob.ID)
	test.OK(t, err)
	test.Equals(t, oID, bJob.ID)
	test.Equals(t, uint64(100), bJob.TasksRequested)
	test.Equals(t, uint64(1), bJob.TasksCompleted)
	test.Equals(t, uint64(1), bJob.TasksErrored)
	test.Equals(t, models.BatchJobStatusComplete, bJob.Status)

	data := []byte{1, 2}
	tasks := []*models.BatchTask{
		{
			Status:         models.BatchTaskStatusPending,
			Data:           data,
			Error:          "Error",
			BatchJobID:     bJob.ID,
			Type:           models.BatchTaskTypePostMessages,
			AvailableAfter: mClk.Now().Add(time.Second * 30),
		},
		{
			Status:         models.BatchTaskStatusPending,
			Data:           data,
			Error:          "Error",
			BatchJobID:     bJob.ID,
			Type:           models.BatchTaskTypePostMessages,
			AvailableAfter: mClk.Now().Add(-(time.Second * 30)),
		},
	}
	err = dal.CreateBatchTasks(ctx, tasks)
	test.OK(t, err)
	test.Assert(t, tasks[0].ID.IsValid, "ID should be valid")
	test.Assert(t, tasks[1].ID.IsValid, "ID should be valid")

	pendingTasks, err := dal.BatchTasksAvailableInStatus(ctx, models.BatchTaskStatusPending, 100)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingTasks))

	mClk.WarpForward(time.Second * 45)
	pendingTasks, err = dal.BatchTasksAvailableInStatus(ctx, models.BatchTaskStatusPending, 100)
	test.OK(t, err)
	test.Equals(t, 2, len(pendingTasks))
	test.Equals(t, models.BatchTaskStatusPending, pendingTasks[0].Status)
	test.Equals(t, data, pendingTasks[0].Data)
	test.Equals(t, "Error", pendingTasks[0].Error)
	test.Equals(t, bJob.ID, pendingTasks[0].BatchJobID)
	test.Equals(t, models.BatchTaskTypePostMessages, pendingTasks[0].Type)

	affCt, err = dal.LeaseBatchTasks(ctx, []models.BatchTaskID{pendingTasks[0].ID, pendingTasks[1].ID}, time.Second*30)
	test.OK(t, err)
	test.Equals(t, int64(2), affCt)
}
