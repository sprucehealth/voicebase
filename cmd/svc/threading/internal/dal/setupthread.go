package dal

import (
	"context"
	"database/sql"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

func (d *dal) CreateSetupThreadState(ctx context.Context, threadID models.ThreadID, entityID string) error {
	_, err := d.db.Exec(`INSERT INTO onboarding_threads (thread_id, entity_id, step) VALUES (?, ?, ?)`, threadID, entityID, 0)
	return errors.Trace(err)
}

func (d *dal) SetupThreadState(ctx context.Context, threadID models.ThreadID, opts ...QueryOption) (*models.SetupThreadState, error) {
	var forUpdateSQL string
	if queryOptions(opts).Has(ForUpdate) {
		forUpdateSQL = ` FOR UPDATE`
	}
	row := d.db.QueryRow(`SELECT thread_id, step FROM onboarding_threads WHERE thread_id = ?`+forUpdateSQL, threadID)
	var state models.SetupThreadState
	state.ThreadID = models.EmptyThreadID()
	if err := row.Scan(&state.ThreadID, &state.Step); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &state, nil
}

func (d *dal) SetupThreadStateForEntity(ctx context.Context, entityID string, opts ...QueryOption) (*models.SetupThreadState, error) {
	var forUpdateSQL string
	if queryOptions(opts).Has(ForUpdate) {
		forUpdateSQL = ` FOR UPDATE`
	}
	row := d.db.QueryRow(`SELECT thread_id, step FROM onboarding_threads WHERE entity_id = ?`+forUpdateSQL, entityID)
	var state models.SetupThreadState
	state.ThreadID = models.EmptyThreadID()
	if err := row.Scan(&state.ThreadID, &state.Step); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &state, nil
}

func (d *dal) UpdateSetupThreadState(ctx context.Context, threadID models.ThreadID, update *SetupThreadStateUpdate) error {
	args := dbutil.MySQLVarArgs()
	if update.Step != nil {
		args.Append("step", *update.Step)
	}
	if args.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE onboarding_threads SET `+args.ColumnsForUpdate()+` WHERE thread_id = ?`,
		append(args.Values(), threadID)...)
	return errors.Trace(err)
}
