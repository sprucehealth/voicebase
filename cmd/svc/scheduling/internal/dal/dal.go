package dal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// QueryOption represents an option available to a query
type QueryOption int

const (
	// ForUpdate represents the FOR UPDATE to be appended to a query
	ForUpdate QueryOption = 1 << iota
)

type queryOptions []QueryOption

func (qos queryOptions) Has(opt QueryOption) bool {
	for _, o := range qos {
		if o == opt {
			return true
		}
	}
	return false
}

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("scheduling/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(ctx context.Context, trans func(ctx context.Context, dal DAL) error) (err error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(ctx context.Context, trans func(ctx context.Context, dal DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}
