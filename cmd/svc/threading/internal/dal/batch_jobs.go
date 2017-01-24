package dal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

// CreateBatchTasks inserts batch_tasks records
func (d *dal) CreateBatchTasks(ctx context.Context, ms []*models.BatchTask) error {
	for i, model := range ms {
		if !model.ID.IsValid {
			id, err := models.NewBatchTaskID()
			if err != nil {
				return errors.Trace(err)
			}
			ms[i].ID = id
		}
	}

	ins := dbutil.MySQLMultiInsert(len(ms))
	for _, model := range ms {
		ins.Append(model.Type, model.AvailableAfter, model.ID, model.BatchJobID, model.Error, model.Completed, model.Status, model.Data)
	}

	_, err := d.db.Exec(
		`INSERT INTO batch_tasks (type, available_after, id, batch_job_id, error, completed, status, data)
			VALUES `+ins.Query(), ins.Values()...)
	return errors.Trace(err)
}

// BatchTask retrieves a batch_tasks record
func (d *dal) BatchTask(ctx context.Context, id models.BatchTaskID, opts ...QueryOption) (*models.BatchTask, error) {
	q := selectBatchTask + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanBatchTask(ctx, row, "id = %v", id)
	return model, errors.Trace(err)
}

// BatchTask retrieves a batch_tasks record
func (d *dal) BatchTasksAvailableInStatus(ctx context.Context, status models.BatchTaskStatus, maxTasks uint64, opts ...QueryOption) ([]*models.BatchTask, error) {
	now := d.clk.Now()
	q := selectBatchTask + ` WHERE status = ? AND available_after < ? ORDER BY created ASC LIMIT ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}

	rows, err := d.db.Query(q, status, now, maxTasks)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ms []*models.BatchTask
	for rows.Next() {
		m, err := scanBatchTask(ctx, rows, "status = %v, available_after < %v, limit = %v", status, now, maxTasks)

		if err != nil {
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, errors.Trace(rows.Err())
}

// BatchJobTasksInStatus retrieves batch_task records for the job group with a given status
func (d *dal) BatchJobTasksInStatus(ctx context.Context, id models.BatchJobID, status models.BatchTaskStatus, opts ...QueryOption) ([]*models.BatchTask, error) {
	q := selectBatchTask + ` WHERE batch_job_id = ? AND status = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, id, status)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ms []*models.BatchTask
	for rows.Next() {
		m, err := scanBatchTask(ctx, rows, "batch_job_id = %v, status = %v", id, status)

		if err != nil {
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, errors.Trace(rows.Err())
}

// UpdateBatchTask updates the mutable aspects of a batch_tasks record
func (d *dal) UpdateBatchTask(ctx context.Context, id models.BatchTaskID, update *models.BatchTaskUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.Error != nil {
		args.Append("error", *update.Error)
	}
	if update.Completed != nil {
		args.Append("completed", *update.Completed)
	}
	if update.AvailableAfter != nil {
		args.Append("available_after", *update.AvailableAfter)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE batch_tasks
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// LeaseBatchTasks updates the lease on a set of batch tasks for processing
func (d *dal) LeaseBatchTasks(ctx context.Context, ids []models.BatchTaskID, leaseDuration time.Duration) (int64, error) {
	vals := make([]interface{}, len(ids)+1)
	for i, id := range ids {
		vals[i+1] = id
	}
	// Assign this last to keep the delta between now and lease time as low as possible
	vals[0] = d.clk.Now().Add(leaseDuration)
	res, err := d.db.Exec(
		`UPDATE batch_tasks
		  SET available_after = ? WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`, vals...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteBatchTask deletes a batch_tasks record
func (d *dal) DeleteBatchTask(ctx context.Context, id models.BatchTaskID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM batch_tasks
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// CreateBatchJobs inserts batch_jobs records
func (d *dal) CreateBatchJobs(ctx context.Context, ms []*models.BatchJob) error {
	for i, model := range ms {
		if !model.ID.IsValid {
			id, err := models.NewBatchJobID()
			if err != nil {
				return errors.Trace(err)
			}
			ms[i].ID = id
		}
	}

	ins := dbutil.MySQLMultiInsert(len(ms))
	for _, model := range ms {
		ins.Append(model.Type, model.TasksCompleted, model.ID, model.Status, model.TasksRequested, model.TasksErrored, model.Completed, model.RequestingEntity)
	}

	_, err := d.db.Exec(
		`INSERT INTO batch_jobs (type, tasks_completed, id, status, tasks_requested, tasks_errored, completed, requesting_entity)
			VALUES `+ins.Query(), ins.Values()...)
	return errors.Trace(err)
}

// BatchJob retrieves a batch_jobs record
func (d *dal) BatchJob(ctx context.Context, id models.BatchJobID, opts ...QueryOption) (*models.BatchJob, error) {
	q := selectBatchJob + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanBatchJob(ctx, row, "id = %v", id)
	return model, errors.Trace(err)
}

// BatchJobsCompletedBefore retrieves batch_jobs with the completed record set to before the given time
func (d *dal) BatchJobsCompletedBefore(ctx context.Context, completed time.Time, opts ...QueryOption) ([]*models.BatchJob, error) {
	q := selectBatchJob + ` WHERE completed < ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, completed)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ms []*models.BatchJob
	for rows.Next() {
		m, err := scanBatchJob(ctx, rows, "completed < %v", completed)

		if err != nil {
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, errors.Trace(rows.Err())
}

// BatchJobsForRequestingEntity retrieves batch_jobs for the provided requesting entity id
func (d *dal) BatchJobsForRequestingEntity(ctx context.Context, entityID string, opts ...QueryOption) ([]*models.BatchJob, error) {
	q := selectBatchJob + ` WHERE requesting_entity = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var ms []*models.BatchJob
	for rows.Next() {
		m, err := scanBatchJob(ctx, rows, "requesting_entity = %v", entityID)

		if err != nil {
			return nil, errors.Trace(err)
		}
		ms = append(ms, m)
	}
	return ms, errors.Trace(rows.Err())
}

// UpdateBatchJob updates the mutable aspects of a batch_jobs record
func (d *dal) UpdateBatchJob(ctx context.Context, id models.BatchJobID, update *models.BatchJobUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.TasksCompleted != nil {
		args.Append("tasks_completed", *update.TasksCompleted)
	}
	if update.TasksErrored != nil {
		args.Append("tasks_errored", *update.TasksErrored)
	}
	if update.Completed != nil {
		args.Append("completed", *update.Completed)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.TasksRequested != nil {
		args.Append("tasks_requested", *update.TasksRequested)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE batch_jobs
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteBatchJob deletes a batch_jobs record
func (d *dal) DeleteBatchJob(ctx context.Context, id models.BatchJobID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM batch_jobs
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectBatchTask = `
    SELECT batch_tasks.modified, batch_tasks.status, batch_tasks.data, batch_tasks.error, batch_tasks.completed, batch_tasks.created, batch_tasks.id, batch_tasks.batch_job_id, batch_tasks.type, batch_tasks.available_after
      FROM batch_tasks`

func scanBatchTask(ctx context.Context, row dbutil.Scanner, contextFormat string, args ...interface{}) (*models.BatchTask, error) {
	var m models.BatchTask
	m.ID = models.EmptyBatchTaskID()
	m.BatchJobID = models.EmptyBatchJobID()

	err := row.Scan(&m.Modified, &m.Status, &m.Data, &m.Error, &m.Completed, &m.Created, &m.ID, &m.BatchJobID, &m.Type, &m.AvailableAfter)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(ErrNotFound, "No rows found - threading.BatchTask - Context: "+fmt.Sprintf(contextFormat, args...))
	}
	return &m, errors.Trace(err)
}

const selectBatchJob = `
    SELECT batch_jobs.type, batch_jobs.tasks_completed, batch_jobs.created, batch_jobs.modified, batch_jobs.id, batch_jobs.status, batch_jobs.tasks_requested, batch_jobs.tasks_errored, batch_jobs.completed, batch_jobs.requesting_entity
      FROM batch_jobs`

func scanBatchJob(ctx context.Context, row dbutil.Scanner, contextFormat string, args ...interface{}) (*models.BatchJob, error) {
	var m models.BatchJob
	m.ID = models.EmptyBatchJobID()

	err := row.Scan(&m.Type, &m.TasksCompleted, &m.Created, &m.Modified, &m.ID, &m.Status, &m.TasksRequested, &m.TasksErrored, &m.Completed, &m.RequestingEntity)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(ErrNotFound, "No rows found - threading.BatchJob - Context: "+fmt.Sprintf(contextFormat, args...))
	}
	return &m, errors.Trace(err)
}
