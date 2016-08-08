package dal

import (
	"database/sql"
	"errors"

	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("payments/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface{}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{db: tsql.AsDB(db)}
}
