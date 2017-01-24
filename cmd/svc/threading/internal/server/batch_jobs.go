package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *threadsServer) BatchJobs(ctx context.Context, in *threading.BatchJobsRequest) (*threading.BatchJobsResponse, error) {
	var batchJobs []*models.BatchJob
	switch key := in.LookupKey.(type) {
	case *threading.BatchJobsRequest_ID:
		if key.ID == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "ID Required")
		}
		batchJobID, err := models.ParseBatchJobID(key.ID)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
		}
		batchJob, err := s.dal.BatchJob(ctx, batchJobID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "BatchJob %s Not Found", batchJobID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		batchJobs = []*models.BatchJob{batchJob}
	case *threading.BatchJobsRequest_RequestingEntity:
		// TODO: Implement before admin integration
		return nil, grpc.Errorf(codes.InvalidArgument, "Not Implemented")
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown lookup key type")
	}

	rBatchJobs := make([]*threading.BatchJob, len(batchJobs))
	for i, batchJob := range batchJobs {
		// for now only collect errors if we're completely done, we would need
		// to lock the set of jobs to assert the count of errors and errored match otherwise
		var errs []string
		if batchJob.Status == models.BatchJobStatusComplete && batchJob.TasksErrored != 0 {
			erroredTasks, err := s.dal.BatchJobTasksInStatus(ctx, batchJob.ID, models.BatchTaskStatusError)
			if err != nil {
				return nil, errors.Trace(err)
			}
			errs := make([]string, len(erroredTasks))
			for i, task := range erroredTasks {
				errs[i] = task.Error
			}
		}
		rBatchJob, err := transformBatchJobToResponse(batchJob, errs)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rBatchJobs[i] = rBatchJob
	}
	return &threading.BatchJobsResponse{
		BatchJobs: rBatchJobs,
	}, nil
}

func (s *threadsServer) BatchPostMessages(ctx context.Context, in *threading.BatchPostMessagesRequest) (*threading.BatchPostMessagesResponse, error) {
	if len(in.PostMessagesRequests) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "At least 1 PostMessagesRequest is required")
	}
	if in.RequestingEntity == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "RequestingEntity Required")
	}
	batchJobID, err := models.NewBatchJobID()
	if err != nil {
		return nil, errors.Trace(err)
	}
	// TODO: Should we validate input at post time or processing?

	tasks := make([]*models.BatchTask, len(in.PostMessagesRequests))
	for i, postMessagesRequest := range in.PostMessagesRequests {
		if postMessagesRequest.ThreadID == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID on PostMessagesRequest Required")
		}
		if postMessagesRequest.FromEntityID == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID on PostMessagesRequest Required")
		}
		if len(postMessagesRequest.Messages) == 0 {
			return nil, grpc.Errorf(codes.InvalidArgument, "At least 1 Message is required in PostMessagesRequest")
		}
		bRequest, err := postMessagesRequest.Marshal()
		if err != nil {
			return nil, errors.Trace(err)
		}
		tasks[i] = &models.BatchTask{
			BatchJobID:     batchJobID,
			Type:           models.BatchTaskTypePostMessages,
			Status:         models.BatchTaskStatusPending,
			Data:           bRequest,
			AvailableAfter: s.clk.Now(),
		}
	}
	var batchJob *models.BatchJob
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := dl.CreateBatchJobs(ctx, []*models.BatchJob{
			&models.BatchJob{
				ID:               batchJobID,
				RequestingEntity: in.RequestingEntity,
				Status:           models.BatchJobStatusPending,
				TasksRequested:   uint64(len(tasks)),
				Type:             models.BatchJobTypeBatchPostMessages,
			},
		}); err != nil {
			return errors.Trace(err)
		}
		if err := dl.CreateBatchTasks(ctx, tasks); err != nil {
			return errors.Trace(err)
		}
		// Reread our job
		batchJob, err = dl.BatchJob(ctx, batchJobID)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	rBatchJob, err := transformBatchJobToResponse(batchJob, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// TODO: Trigger a pass of the worker here...
	return &threading.BatchPostMessagesResponse{
		BatchJob: rBatchJob,
	}, nil
}
