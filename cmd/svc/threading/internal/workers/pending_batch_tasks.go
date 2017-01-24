package workers

import (
	"context"

	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/smet"
)

type batchTasksThreadClient interface {
	processPostMessagesThreadClient
}

var taskTypeDuration = map[models.BatchTaskType]time.Duration{
	// Allot 15 seconds to post messages - This is insanely long
	models.BatchTaskTypePostMessages: time.Second * 15,
}

const (
	// MaxPendingTasksToProcess represents the max pending tasks to process in parallel at one time
	// TODO: Regulate this by tracking the number of actual in flight tasks rather than X per batch
	MaxPendingTasksToProcess = 200
	// InternalErrorMessage represents the common user message to return to users when we don't know
	// how to express what has gone wrong
	InternalErrorMessage = "Internal Error - Something went wrong"
)

var batchJobStatusComplete = models.BatchJobStatusComplete

// processPendingBatchTasks gathers groups of pending batch tasks for processing
func (w *Workers) processPendingBatchTasks() {
	ctx := context.Background()
	var tasksByType map[models.BatchTaskType][]*models.BatchTask
	if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Find some work to do, lock the rows for a quick moment and take the lease
		batchTasks, err := dl.BatchTasksAvailableInStatus(ctx, models.BatchTaskStatusPending, MaxPendingTasksToProcess, dal.ForUpdate)
		if err != nil {
			return errors.Wrapf(err, "Encountered error looking for pending batch tasks")
		}
		if len(batchTasks) == 0 {
			return nil
		}
		golog.Infof("Found %d Pending BatchTasks for processing", len(batchTasks))
		tasksByType = make(map[models.BatchTaskType][]*models.BatchTask)
		for _, task := range batchTasks {
			if _, ok := tasksByType[task.Type]; !ok {
				tasksByType[task.Type] = make([]*models.BatchTask, 0, len(batchTasks))
			}
			tasksByType[task.Type] = append(tasksByType[task.Type], task)
		}
		for taskType, tasks := range tasksByType {
			// Should never be empty but be defensive
			if len(tasks) == 0 {
				continue
			}
			taskIDs := make([]models.BatchTaskID, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			if _, err := dl.LeaseBatchTasks(ctx, taskIDs, taskTypeDuration[taskType]); err != nil {
				return errors.Wrapf(err, "Encountered error while taking lease for %d tasks", len(taskIDs))
			}
		}
		return nil
	}); err != nil {
		smet.Error(workerErrMetricName, err)
		return
	}

	for taskType, tasks := range tasksByType {
		for _, t := range tasks {
			task := t
			conc.Go(func() {
				// TODO: Build generic pattern for cloning context
				cctx := context.Background()
				var err error
				var userErrorMsg string
				switch taskType {
				case models.BatchTaskTypePostMessages:
					userErrorMsg, err = processBatchTaskPostMessages(cctx, task, w.dal, w.threadingCli)
				default:
					golog.Errorf("Unknown BatchTask type %s - CANNOT PROCESS", taskType)
				}
				if err != nil {
					golog.Errorf("Encountered error while processing batch task %s: %s", task.ID, err)
				}
				if err := w.dal.Transact(cctx, func(ctx context.Context, dl dal.DAL) error {
					lockedBatchJob, err := dl.BatchJob(ctx, task.BatchJobID, dal.ForUpdate)
					if err != nil {
						return errors.Wrapf(err, "Error while locking BatchTask %s for update", task.ID)
					}

					var taskError string
					taskCompleted := w.clk.Now()
					taskStatus := models.BatchTaskStatusComplete
					tasksCompleted := lockedBatchJob.TasksCompleted + 1
					tasksErrored := lockedBatchJob.TasksErrored
					if userErrorMsg != "" {
						tasksErrored++
						taskError = userErrorMsg
						taskStatus = models.BatchTaskStatusError
					}

					var jobStatus *models.BatchJobStatus
					var jobCompleted *time.Time
					if tasksCompleted == lockedBatchJob.TasksRequested {
						jobStatus = &batchJobStatusComplete
						jobCompleted = ptr.Time(w.clk.Now())
					}

					if _, err := dl.UpdateBatchTask(ctx, task.ID, &models.BatchTaskUpdate{
						Error:     ptr.String(taskError),
						Status:    &taskStatus,
						Completed: ptr.Time(taskCompleted),
					}); err != nil {
						return errors.Wrapf(err, "Error while updating BatchTask %s", task.ID)
					}

					if _, err := dl.UpdateBatchJob(ctx, lockedBatchJob.ID, &models.BatchJobUpdate{
						TasksCompleted: ptr.Uint64(tasksCompleted),
						TasksErrored:   ptr.Uint64(tasksErrored),
						Status:         jobStatus,
						Completed:      jobCompleted,
					}); err != nil {
						return errors.Wrapf(err, "Error while updating BatchJob %s", lockedBatchJob.ID)
					}
					return nil
				}); err != nil {
					golog.Errorf("Error while updating BatchTask %s: %s", task.ID, err)
				}
				golog.Debugf("Completed BatchTask %s", task.ID)
			})
		}
	}
}
